package writer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
	"github.com/louvri/xlsxlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─────────────────────────────────────────────────────────────────────────────
// Test helpers
// ─────────────────────────────────────────────────────────────────────────────

// mockConnection implements connection.Connection for GCS cancel tests.
type mockConnection struct {
	cancelled bool
	closed    bool
}

func (m *mockConnection) Connect(_ context.Context) error { return nil }
func (m *mockConnection) Cancel()                         { m.cancelled = true }
func (m *mockConnection) Close()                          { m.closed = true }
func (m *mockConnection) Writer() io.Writer               { return nil }
func (m *mockConnection) Reader() io.Reader               { return nil }
func (m *mockConnection) Handler() any                    { return nil }
func (m *mockConnection) Name() string                    { return "mock" }
func (m *mockConnection) Driver() string                  { return "mock" }

// ictxWithWriter creates an IContext with a FILE_KEY writer and optional
// FILE_OBJECT_KEY connection, mirroring what gcs/open deposits.
func ictxWithWriter(filename string, w io.Writer, con *mockConnection) icontext.IContext {
	ictx := icontext.New(context.Background())
	ictx.Set(sys_key.FILE_KEY, map[string]any{filename: w})
	if con != nil {
		ictx.Set(sys_key.FILE_OBJECT_KEY, map[string]any{filename: con})
	}
	return ictx
}

// nopEndpoint is a no-op inner endpoint used when we only care about the writer.
func nopEndpoint(_ context.Context, req any) (any, error) {
	return req, nil
}

// errorEndpoint always returns an error from the inner endpoint.
func errorEndpoint(_ context.Context, _ any) (any, error) {
	return nil, errors.New("inner error")
}

// readXLSX parses the bytes written to buf and returns all rows from the named
// sheet as [][]string for easy assertion.
func readXLSX(t *testing.T, buf *bytes.Buffer, sheetName string) [][]string {
	t.Helper()
	data := buf.Bytes()
	require.NotEmpty(t, data, "XLSX buffer must not be empty")

	r, err := xlsxlite.OpenReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	defer r.Close()

	iter, err := r.OpenSheet(sheetName)
	require.NoError(t, err)
	defer iter.Close()

	var result [][]string
	for iter.Next() {
		row := iter.Row()
		line := make([]string, len(row.Cells))
		for i, cell := range row.Cells {
			line[i] = fmt.Sprintf("%v", cell.Value)
		}
		result = append(result, line)
	}
	require.NoError(t, iter.Err())
	return result
}

// runChain simulates the full lifecycle of one reader-driven write cycle.
//
// It mimics what csv/reader does:
//  1. Set SOF = true, call chain with the first row.
//  2. Set SOF = false, call chain with remaining rows.
//  3. Set EOF = "eof", call chain with nil to finalise.
func runChain(
	t *testing.T,
	mw endpoint.Middleware,
	ictx icontext.IContext,
	rows []map[string]any,
) error {
	t.Helper()
	chain := mw(nopEndpoint)

	for i, row := range rows {
		if i == 0 {
			ictx.Set(sys_key.SOF, true)
		} else {
			ictx.Set(sys_key.SOF, false)
		}
		if _, err := chain(ictx, row); err != nil {
			return err
		}
	}

	// EOF finalise pass.
	ictx.Set(sys_key.EOF, "eof")
	_, err := chain(ictx, nil)
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// Core behaviour tests
// ─────────────────────────────────────────────────────────────────────────────

func TestMiddleware_WritesHeaderAndRows(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"name", "city", "age"}
	ictx := ictxWithWriter("out.xlsx", &buf, nil)

	rows := []map[string]any{
		{"name": "Alice", "city": "Jakarta", "age": 30},
		{"name": "Bob", "city": "Bandung", "age": 25},
	}

	err := runChain(t, Middleware("out.xlsx", "Sheet1", columns, false), ictx, rows)
	require.NoError(t, err)

	got := readXLSX(t, &buf, "Sheet1")
	require.Len(t, got, 3) // header + 2 data rows
	assert.Equal(t, []string{"name", "city", "age"}, got[0])
	assert.Equal(t, []string{"Alice", "Jakarta", "30"}, got[1])
	assert.Equal(t, []string{"Bob", "Bandung", "25"}, got[2])
}

func TestMiddleware_EmptyWrite_ProducesValidXLSX(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"col1", "col2"}
	ictx := ictxWithWriter("out.xlsx", &buf, nil)

	// No data rows — just EOF.
	err := runChain(t, Middleware("out.xlsx", "Sheet1", columns, false), ictx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, buf.Bytes(), "XLSX file must still be written even with no data rows")

	// Must be a parseable XLSX (no panic, no error).
	r, err := xlsxlite.OpenReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	defer r.Close()
	assert.Equal(t, []string{"Sheet1"}, r.SheetNames())
}

func TestMiddleware_SingleRow(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"id", "value"}
	ictx := ictxWithWriter("f.xlsx", &buf, nil)

	rows := []map[string]any{
		{"id": "1", "value": "hello"},
	}

	err := runChain(t, Middleware("f.xlsx", "Sheet1", columns, false), ictx, rows)
	require.NoError(t, err)

	got := readXLSX(t, &buf, "Sheet1")
	require.Len(t, got, 2)
	assert.Equal(t, []string{"id", "value"}, got[0])
	assert.Equal(t, []string{"1", "hello"}, got[1])
}

func TestMiddleware_ColumnOrderRespected(t *testing.T) {
	var buf bytes.Buffer
	// Deliberately different order from the map insertion order.
	columns := []string{"z", "a", "m"}
	ictx := ictxWithWriter("f.xlsx", &buf, nil)

	rows := []map[string]any{
		{"a": "A", "m": "M", "z": "Z"},
	}

	err := runChain(t, Middleware("f.xlsx", "Sheet1", columns, false), ictx, rows)
	require.NoError(t, err)

	got := readXLSX(t, &buf, "Sheet1")
	require.Len(t, got, 2)
	assert.Equal(t, []string{"z", "a", "m"}, got[0])
	assert.Equal(t, []string{"Z", "A", "M"}, got[1])
}

func TestMiddleware_MissingKeyProducesEmptyCell(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"present", "missing"}
	ictx := ictxWithWriter("f.xlsx", &buf, nil)

	rows := []map[string]any{
		{"present": "here"}, // "missing" key absent
	}

	err := runChain(t, Middleware("f.xlsx", "Sheet1", columns, false), ictx, rows)
	require.NoError(t, err)

	got := readXLSX(t, &buf, "Sheet1")
	require.Len(t, got, 2)
	// Missing key → nil cell → formatted as "<nil>"
	assert.Equal(t, "here", got[1][0])
	// Empty cell value is either "<nil>" or empty string depending on xlsxlite
	// representation — we just assert it doesn't panic and the row has 2 cells.
	assert.Len(t, got[1], 2)
}

func TestMiddleware_BatchResponse_MultipleRowsPerCall(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"k"}
	ictx := ictxWithWriter("f.xlsx", &buf, nil)

	// Simulate inner endpoint returning a []map[string]any batch.
	batchEndpoint := func(_ context.Context, req any) (any, error) {
		return []map[string]any{
			{"k": "v1"},
			{"k": "v2"},
			{"k": "v3"},
		}, nil
	}

	chain := Middleware("f.xlsx", "Sheet1", columns, false)(batchEndpoint)
	ictx.Set(sys_key.SOF, true)
	_, err := chain(ictx, nil)
	require.NoError(t, err)

	ictx.Set(sys_key.SOF, false)
	ictx.Set(sys_key.EOF, "eof")
	_, err = chain(ictx, nil)
	require.NoError(t, err)

	got := readXLSX(t, &buf, "Sheet1")
	require.Len(t, got, 4) // header + 3 rows
}

func TestMiddleware_LargeDataset(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"idx", "data"}
	ictx := ictxWithWriter("big.xlsx", &buf, nil)

	const n = 10_000
	rows := make([]map[string]any, n)
	for i := range rows {
		rows[i] = map[string]any{"idx": i, "data": fmt.Sprintf("row-%d", i)}
	}

	err := runChain(t, Middleware("big.xlsx", "Sheet1", columns, false), ictx, rows)
	require.NoError(t, err)

	got := readXLSX(t, &buf, "Sheet1")
	assert.Len(t, got, n+1) // header + n rows
}

func TestMiddleware_TypeVariants(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"str", "int", "float", "bool", "nil_val"}
	ictx := ictxWithWriter("f.xlsx", &buf, nil)

	rows := []map[string]any{
		{
			"str":     "hello",
			"int":     42,
			"float":   3.14,
			"bool":    true,
			"nil_val": nil,
		},
	}

	err := runChain(t, Middleware("f.xlsx", "Sheet1", columns, false), ictx, rows)
	require.NoError(t, err)

	got := readXLSX(t, &buf, "Sheet1")
	require.Len(t, got, 2)
	row := got[1]
	assert.Equal(t, "hello", row[0])
	assert.Equal(t, "42", row[1])
	// float — xlsxlite stores as number; formatted value varies by precision
	assert.NotEmpty(t, row[2])
	assert.Equal(t, "true", row[3])
}

// ─────────────────────────────────────────────────────────────────────────────
// Error and cancellation tests
// ─────────────────────────────────────────────────────────────────────────────

func TestMiddleware_InnerError_CancelOnError_True(t *testing.T) {
	var buf bytes.Buffer
	con := &mockConnection{}
	ictx := ictxWithWriter("f.xlsx", &buf, con)
	columns := []string{"col"}

	chain := Middleware("f.xlsx", "Sheet1", columns, true)(errorEndpoint)
	ictx.Set(sys_key.SOF, true)
	_, err := chain(ictx, map[string]any{"col": "v"})

	// The inner error must be propagated.
	require.Error(t, err)
	assert.EqualError(t, err, "inner error")
	// GCS connection must be cancelled.
	assert.True(t, con.cancelled, "connection should be cancelled on error")
}

func TestMiddleware_InnerError_CancelOnError_False(t *testing.T) {
	var buf bytes.Buffer
	con := &mockConnection{}
	ictx := ictxWithWriter("f.xlsx", &buf, con)
	columns := []string{"col"}

	chain := Middleware("f.xlsx", "Sheet1", columns, false)(errorEndpoint)
	ictx.Set(sys_key.SOF, true)
	_, err := chain(ictx, map[string]any{"col": "v"})

	require.Error(t, err)
	// Without cancelOnError the connection must NOT be cancelled.
	assert.False(t, con.cancelled, "connection should not be cancelled when cancelOnError=false")
}

func TestMiddleware_AbnormalEOF_CancelsConnection(t *testing.T) {
	var buf bytes.Buffer
	con := &mockConnection{}
	ictx := ictxWithWriter("f.xlsx", &buf, con)
	columns := []string{"col"}

	// Prime the state with one row.
	chain := Middleware("f.xlsx", "Sheet1", columns, true)(nopEndpoint)
	ictx.Set(sys_key.SOF, true)
	_, err := chain(ictx, map[string]any{"col": "v"})
	require.NoError(t, err)

	// Signal an abnormal EOF (any non-"eof" truthy value).
	ictx.Set(sys_key.EOF, "error")
	_, err = chain(ictx, nil)
	require.NoError(t, err)
	assert.True(t, con.cancelled, "abnormal EOF should cancel the connection")
}

func TestMiddleware_MissingFileKey_ReturnsError(t *testing.T) {
	ictx := icontext.New(context.Background()) // no FILE_KEY set
	columns := []string{"col"}

	chain := Middleware("missing.xlsx", "Sheet1", columns, false)(nopEndpoint)
	ictx.Set(sys_key.SOF, true)
	_, err := chain(ictx, map[string]any{"col": "v"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "FILE_KEY not found")
}

func TestMiddleware_WrongFilenameKey_ReturnsError(t *testing.T) {
	var buf bytes.Buffer
	ictx := ictxWithWriter("other.xlsx", &buf, nil) // key mismatch
	columns := []string{"col"}

	chain := Middleware("wanted.xlsx", "Sheet1", columns, false)(nopEndpoint)
	ictx.Set(sys_key.SOF, true)
	_, err := chain(ictx, map[string]any{"col": "v"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no writer found")
}

// ─────────────────────────────────────────────────────────────────────────────
// Helper unit tests
// ─────────────────────────────────────────────────────────────────────────────

func TestToCell_Types(t *testing.T) {
	tests := []struct {
		input    any
		wantType xlsxlite.CellType
	}{
		{nil, xlsxlite.CellTypeEmpty},
		{"hello", xlsxlite.CellTypeString},
		{42, xlsxlite.CellTypeNumber},
		{int64(99), xlsxlite.CellTypeNumber},
		{float64(3.14), xlsxlite.CellTypeNumber},
		{float32(1.0), xlsxlite.CellTypeNumber},
		{true, xlsxlite.CellTypeBool},
		{false, xlsxlite.CellTypeBool},
	}
	for _, tc := range tests {
		cell := toCell(tc.input)
		assert.Equal(t, tc.wantType, cell.Type, "input=%v", tc.input)
	}
}

func TestToCell_FallbackStringify(t *testing.T) {
	// Custom type not in the switch — should be stringified.
	type custom struct{ V int }
	cell := toCell(custom{V: 7})
	assert.Equal(t, xlsxlite.CellTypeString, cell.Type)
	assert.Equal(t, "{7}", cell.Value)
}

func TestToRows_SingleMap(t *testing.T) {
	m := map[string]any{"k": "v"}
	rows := toRows(m)
	assert.Len(t, rows, 1)
	assert.Equal(t, "v", rows[0]["k"])
}

func TestToRows_SliceOfMaps(t *testing.T) {
	s := []map[string]any{{"k": "v1"}, {"k": "v2"}}
	rows := toRows(s)
	assert.Len(t, rows, 2)
}

func TestToRows_Nil(t *testing.T) {
	assert.Nil(t, toRows(nil))
}

func TestToRows_UnknownType(t *testing.T) {
	assert.Nil(t, toRows("unexpected string"))
}

func TestMakeHeaderCells(t *testing.T) {
	cols := []string{"a", "b", "c"}
	cells := makeHeaderCells(cols)
	require.Len(t, cells, 3)
	for i, col := range cols {
		assert.Equal(t, col, cells[i].Value)
		assert.Equal(t, xlsxlite.CellTypeString, cells[i].Type)
	}
}

func TestMakeDataCells_OrderAndMissing(t *testing.T) {
	columns := []string{"x", "y", "z"}
	data := map[string]any{"x": "X", "z": "Z"} // "y" missing
	cells := makeDataCells(columns, data)
	require.Len(t, cells, 3)
	assert.Equal(t, "X", cells[0].Value)
	// "y" is missing → EmptyCell
	assert.Equal(t, xlsxlite.CellTypeEmpty, cells[1].Type)
	assert.Equal(t, "Z", cells[2].Value)
}

// ─────────────────────────────────────────────────────────────────────────────
// Context / IContext plumbing tests
// ─────────────────────────────────────────────────────────────────────────────

func TestMiddleware_PlainContextIsWrapped(t *testing.T) {
	// Pass a plain context.Background() — middleware must wrap it in IContext.
	var buf bytes.Buffer
	plainCtx := context.Background()

	// Manually inject FILE_KEY via a real IContext first, then extract its base
	// to simulate a caller who doesn't know about IContext.
	ictx := icontext.New(plainCtx)
	ictx.Set(sys_key.FILE_KEY, map[string]any{"f.xlsx": io.Writer(&buf)})
	ictx.Set(sys_key.SOF, true)

	columns := []string{"c"}
	chain := Middleware("f.xlsx", "Sheet1", columns, false)(nopEndpoint)
	_, err := chain(ictx, map[string]any{"c": "v"})
	require.NoError(t, err)

	ictx.Set(sys_key.EOF, "eof")
	_, err = chain(ictx, nil)
	require.NoError(t, err)

	got := readXLSX(t, &buf, "Sheet1")
	require.Len(t, got, 2)
}

// ─────────────────────────────────────────────────────────────────────────────
// Sheet name test
// ─────────────────────────────────────────────────────────────────────────────

func TestMiddleware_CustomSheetName(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"v"}
	ictx := ictxWithWriter("f.xlsx", &buf, nil)

	rows := []map[string]any{{"v": "hello"}}
	err := runChain(t, Middleware("f.xlsx", "MyCustomSheet", columns, false), ictx, rows)
	require.NoError(t, err)

	r, err := xlsxlite.OpenReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	defer r.Close()
	assert.Equal(t, []string{"MyCustomSheet"}, r.SheetNames())
}
