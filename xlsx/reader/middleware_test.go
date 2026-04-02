package reader

import (
	"archive/zip"
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

// sheetDef describes one sheet: its name and the rows to write (row 0 = header).
type sheetDef struct {
	name string
	rows [][]string
}

// makeXLSXMulti builds an in-memory XLSX file with one or more named sheets.
func makeXLSXMulti(t *testing.T, sheets []sheetDef) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := xlsxlite.NewWriter(&buf)
	for _, sd := range sheets {
		sw, err := w.NewSheet(xlsxlite.SheetConfig{Name: sd.name})
		require.NoError(t, err)
		for _, row := range sd.rows {
			cells := make([]xlsxlite.Cell, len(row))
			for i, v := range row {
				cells[i] = xlsxlite.StringCell(v)
			}
			require.NoError(t, sw.WriteRow(xlsxlite.Row{Cells: cells}))
		}
		require.NoError(t, sw.Close())
	}
	require.NoError(t, w.Close())
	return buf.Bytes()
}

// makeXLSX builds a single-sheet ("Sheet1") XLSX for backward-compat tests.
func makeXLSX(t *testing.T, rows [][]string) []byte {
	t.Helper()
	return makeXLSXMulti(t, []sheetDef{{name: "Sheet1", rows: rows}})
}

// makeEmptyXLSX builds an XLSX with a sheet that has only a header row.
func makeEmptyXLSX(t *testing.T) []byte {
	t.Helper()
	return makeXLSX(t, [][]string{{"name", "age"}})
}

// ictxWithFile constructs an IContext that carries the given bytes as an
// io.Reader under sys_key.FILE_KEY["filename"], replicating what gcs/open does.
func ictxWithFile(filename string, data []byte) icontext.IContext {
	base := icontext.New(context.Background())
	base.Set(sys_key.FILE_KEY, map[string]any{filename: bytes.NewReader(data)})
	return base
}

// collectEndpoint accumulates every non-nil request as a map[string]any.
func collectEndpoint(collected *[]map[string]any) endpoint.Endpoint {
	return func(_ context.Context, req any) (any, error) {
		if req == nil {
			return nil, nil
		}
		row, ok := req.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("unexpected type %T", req)
		}
		*collected = append(*collected, row)
		return nil, nil
	}
}

// run is a shorthand: build the chain, call it, return (rows, err).
func run(t *testing.T, mw endpoint.Middleware, data []byte, filename string) ([]map[string]any, error) {
	t.Helper()
	var rows []map[string]any
	_, err := mw(collectEndpoint(&rows))(ictxWithFile(filename, data), nil)
	return rows, err
}

// ─────────────────────────────────────────────────────────────────────────────
// Single-sheet tests (backward-compatible — sheets: []string{"Sheet1"})
// ─────────────────────────────────────────────────────────────────────────────

func TestMiddleware_HappyPath(t *testing.T) {
	data := makeXLSX(t, [][]string{
		{"name", "city"},
		{"Alice", "Jakarta"},
		{"Bob", "Bandung"},
	})

	rows, err := run(t, Middleware("f.xlsx", []string{"Sheet1"}, nil, false), data, "f.xlsx")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "Alice", rows[0]["name"])
	assert.Equal(t, "Jakarta", rows[0]["city"])
	assert.Equal(t, "Bob", rows[1]["name"])
	assert.Equal(t, "Bandung", rows[1]["city"])
}

func TestMiddleware_SheetNameInjected(t *testing.T) {
	data := makeXLSX(t, [][]string{{"col"}, {"val"}})
	rows, err := run(t, Middleware("f.xlsx", []string{"Sheet1"}, nil, false), data, "f.xlsx")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Sheet1", rows[0][RowKeySheetName])
}

func TestMiddleware_LineNumberKey(t *testing.T) {
	data := makeXLSX(t, [][]string{
		{"col"},
		{"a"},
		{"b"},
	})

	rows, err := run(t, Middleware("f.xlsx", []string{"Sheet1"}, nil, false), data, "f.xlsx")
	require.NoError(t, err)
	// lineNumber is 1-based and excludes the header row.
	assert.Equal(t, 2, rows[0][RowKeyLineNumber])
	assert.Equal(t, 3, rows[1][RowKeyLineNumber])
}

func TestMiddleware_WithDecoder(t *testing.T) {
	type Record struct{ Name string }

	data := makeXLSX(t, [][]string{{"name"}, {"Charlie"}})
	decoder := func(raw any) any {
		m := raw.(map[string]any)
		return Record{Name: fmt.Sprintf("%v", m["name"])}
	}

	var got []Record
	ep := Middleware("f.xlsx", []string{"Sheet1"}, decoder, false)(
		func(_ context.Context, req any) (any, error) {
			if req == nil {
				return nil, nil
			}
			got = append(got, req.(Record))
			return nil, nil
		},
	)
	_, err := ep(ictxWithFile("f.xlsx", data), nil)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "Charlie", got[0].Name)
}

func TestMiddleware_EmptySheet_NoDataRows(t *testing.T) {
	rows, err := run(t, Middleware("f.xlsx", []string{"Sheet1"}, nil, false), makeEmptyXLSX(t), "f.xlsx")
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestMiddleware_SingleEmptyString_OpensFirstSheet(t *testing.T) {
	data := makeXLSX(t, [][]string{{"key"}, {"value1"}})
	// []string{""} → first sheet only
	rows, err := run(t, Middleware("f.xlsx", []string{""}, nil, false), data, "f.xlsx")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "value1", rows[0]["key"])
}

func TestMiddleware_NilSheets_OpensAllSheets(t *testing.T) {
	data := makeXLSXMulti(t, []sheetDef{
		{name: "A", rows: [][]string{{"x"}, {"a1"}}},
		{name: "B", rows: [][]string{{"x"}, {"b1"}}},
	})
	// nil → AllSheets
	rows, err := run(t, Middleware("f.xlsx", nil, nil, false), data, "f.xlsx")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "A", rows[0][RowKeySheetName])
	assert.Equal(t, "B", rows[1][RowKeySheetName])
}

func TestMiddleware_IgnoreError_CollectsAllErrors(t *testing.T) {
	data := makeXLSX(t, [][]string{{"id"}, {"1"}, {"2"}, {"3"}})

	callCount := 0
	ep := func(_ context.Context, req any) (any, error) {
		if req == nil {
			return nil, nil
		}
		callCount++
		return nil, fmt.Errorf("row error %d", callCount)
	}

	_, err := Middleware("f.xlsx", []string{"Sheet1"}, nil, true)(ep)(ictxWithFile("f.xlsx", data), nil)
	assert.Equal(t, 3, callCount)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "row error 1")
	assert.Contains(t, err.Error(), "row error 2")
	assert.Contains(t, err.Error(), "row error 3")
}

func TestMiddleware_StopOnError_ShortCircuits(t *testing.T) {
	data := makeXLSX(t, [][]string{{"id"}, {"1"}, {"2"}, {"3"}})

	callCount := 0
	ep := func(_ context.Context, req any) (any, error) {
		if req == nil {
			return nil, nil
		}
		callCount++
		return nil, errors.New("boom")
	}

	_, err := Middleware("f.xlsx", []string{"Sheet1"}, nil, false)(ep)(ictxWithFile("f.xlsx", data), nil)
	assert.Equal(t, 1, callCount, "should stop after first error")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestMiddleware_DuplicateErrors_Deduplicated(t *testing.T) {
	data := makeXLSX(t, [][]string{{"x"}, {"a"}, {"b"}})
	ep := func(_ context.Context, req any) (any, error) {
		if req == nil {
			return nil, nil
		}
		return nil, errors.New("same error")
	}
	_, err := Middleware("f.xlsx", []string{"Sheet1"}, nil, true)(ep)(ictxWithFile("f.xlsx", data), nil)
	require.Error(t, err)
	assert.Equal(t, "same error", err.Error())
}

func TestMiddleware_SOFAndEOFSignals(t *testing.T) {
	data := makeXLSX(t, [][]string{{"col"}, {"row1"}})

	var sofSeen, eofSeen bool
	ep := func(ctx context.Context, req any) (any, error) {
		ictx, ok := ctx.(icontext.IContext)
		if !ok {
			return nil, nil
		}
		if sof, ok := ictx.Get(sys_key.SOF).(bool); ok && sof {
			sofSeen = true
		}
		if eof := ictx.Get(sys_key.EOF); eof != nil && eof == "eof" {
			eofSeen = true
		}
		return nil, nil
	}

	_, err := Middleware("f.xlsx", []string{"Sheet1"}, nil, false)(ep)(ictxWithFile("f.xlsx", data), nil)
	require.NoError(t, err)
	assert.True(t, sofSeen, "SOF should be set on first data row")
	assert.True(t, eofSeen, "EOF should be set after all rows")
}

func TestMiddleware_MissingFileKey_ReturnsError(t *testing.T) {
	mw := Middleware("missing.xlsx", []string{"Sheet1"}, nil, false)
	_, err := mw(func(_ context.Context, _ any) (any, error) { return nil, nil })(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FILE_KEY not found")
}

func TestMiddleware_WrongFilenameKey_ReturnsError(t *testing.T) {
	data := makeXLSX(t, [][]string{{"col"}, {"val"}})
	mw := Middleware("correct.xlsx", []string{"Sheet1"}, nil, false)
	_, err := mw(func(_ context.Context, _ any) (any, error) { return nil, nil })(ictxWithFile("wrong.xlsx", data), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no reader found")
}

func TestMiddleware_CorruptXLSX_ReturnsError(t *testing.T) {
	mw := Middleware("bad.xlsx", []string{"Sheet1"}, nil, false)
	_, err := mw(func(_ context.Context, _ any) (any, error) { return nil, nil })(ictxWithFile("bad.xlsx", []byte("not xlsx")), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse xlsx")
}

func TestMiddleware_EmptyFile_ReturnsError(t *testing.T) {
	mw := Middleware("empty.xlsx", []string{"Sheet1"}, nil, false)
	_, err := mw(func(_ context.Context, _ any) (any, error) { return nil, nil })(ictxWithFile("empty.xlsx", []byte{}), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is empty")
}

func TestMiddleware_UnknownSheet_ReturnsError(t *testing.T) {
	data := makeXLSX(t, [][]string{{"col"}, {"val"}})
	mw := Middleware("f.xlsx", []string{"DoesNotExist"}, nil, false)
	_, err := mw(func(_ context.Context, _ any) (any, error) { return nil, nil })(ictxWithFile("f.xlsx", data), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMiddleware_ShortDataRow_NilPaddedColumns(t *testing.T) {
	data := makeXLSX(t, [][]string{{"a", "b", "c"}, {"only_a"}})
	rows, err := run(t, Middleware("f.xlsx", []string{"Sheet1"}, nil, false), data, "f.xlsx")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "only_a", rows[0]["a"])
	assert.Nil(t, rows[0]["b"])
	assert.Nil(t, rows[0]["c"])
}

func TestMiddleware_ResponsePropagation(t *testing.T) {
	data := makeXLSX(t, [][]string{{"n"}, {"1"}})
	sentinel := map[string]any{"result": "ok"}
	ep := func(_ context.Context, req any) (any, error) {
		if req == nil {
			return nil, nil
		}
		return sentinel, nil
	}
	resp, err := Middleware("f.xlsx", []string{"Sheet1"}, nil, false)(ep)(ictxWithFile("f.xlsx", data), nil)
	require.NoError(t, err)
	assert.Equal(t, sentinel, resp)
}

// ─────────────────────────────────────────────────────────────────────────────
// Multi-sheet tests
// ─────────────────────────────────────────────────────────────────────────────

func TestMiddleware_MultiSheet_ExplicitNames(t *testing.T) {
	data := makeXLSXMulti(t, []sheetDef{
		{name: "Jan", rows: [][]string{{"amount"}, {"100"}, {"200"}}},
		{name: "Feb", rows: [][]string{{"amount"}, {"300"}}},
		{name: "Mar", rows: [][]string{{"amount"}, {"400"}}},
	})

	// Read only Jan and Mar — skip Feb.
	rows, err := run(t, Middleware("f.xlsx", []string{"Jan", "Mar"}, nil, false), data, "f.xlsx")
	require.NoError(t, err)
	require.Len(t, rows, 3) // Jan:2 + Mar:1

	sheets := []string{rows[0][RowKeySheetName].(string), rows[1][RowKeySheetName].(string), rows[2][RowKeySheetName].(string)}
	assert.Equal(t, []string{"Jan", "Jan", "Mar"}, sheets)
}

func TestMiddleware_MultiSheet_AllSheets_Sentinel(t *testing.T) {
	data := makeXLSXMulti(t, []sheetDef{
		{name: "X", rows: [][]string{{"v"}, {"x1"}}},
		{name: "Y", rows: [][]string{{"v"}, {"y1"}}},
		{name: "Z", rows: [][]string{{"v"}, {"z1"}}},
	})

	rows, err := run(t, Middleware("f.xlsx", AllSheets, nil, false), data, "f.xlsx")
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, "X", rows[0][RowKeySheetName])
	assert.Equal(t, "Y", rows[1][RowKeySheetName])
	assert.Equal(t, "Z", rows[2][RowKeySheetName])
}

func TestMiddleware_MultiSheet_LineNumberResetsPerSheet(t *testing.T) {
	// lineNumber is per-sheet (1-based from the data row, header = row 1).
	data := makeXLSXMulti(t, []sheetDef{
		{name: "S1", rows: [][]string{{"c"}, {"r1"}, {"r2"}}},
		{name: "S2", rows: [][]string{{"c"}, {"r1"}}},
	})

	rows, err := run(t, Middleware("f.xlsx", AllSheets, nil, false), data, "f.xlsx")
	require.NoError(t, err)
	require.Len(t, rows, 3)

	// S1 rows
	assert.Equal(t, 2, rows[0][RowKeyLineNumber]) // first data row is line 2 (header is 1)
	assert.Equal(t, 3, rows[1][RowKeyLineNumber])
	// S2 row — counter resets
	assert.Equal(t, 2, rows[2][RowKeyLineNumber])
}

func TestMiddleware_MultiSheet_SheetNameInEveryRow(t *testing.T) {
	data := makeXLSXMulti(t, []sheetDef{
		{name: "Alpha", rows: [][]string{{"col"}, {"a"}, {"b"}}},
		{name: "Beta", rows: [][]string{{"col"}, {"c"}}},
	})

	rows, err := run(t, Middleware("f.xlsx", AllSheets, nil, false), data, "f.xlsx")
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, "Alpha", rows[0][RowKeySheetName])
	assert.Equal(t, "Alpha", rows[1][RowKeySheetName])
	assert.Equal(t, "Beta", rows[2][RowKeySheetName])
}

func TestMiddleware_MultiSheet_DecoderReceivesSheetName(t *testing.T) {
	type Rec struct {
		Sheet string
		Val   string
	}
	data := makeXLSXMulti(t, []sheetDef{
		{name: "P", rows: [][]string{{"v"}, {"hello"}}},
		{name: "Q", rows: [][]string{{"v"}, {"world"}}},
	})

	decoder := func(raw any) any {
		m := raw.(map[string]any)
		return Rec{Sheet: fmt.Sprintf("%v", m[RowKeySheetName]), Val: fmt.Sprintf("%v", m["v"])}
	}

	var got []Rec
	ep := Middleware("f.xlsx", AllSheets, decoder, false)(func(_ context.Context, req any) (any, error) {
		if req == nil {
			return nil, nil
		}
		got = append(got, req.(Rec))
		return nil, nil
	})
	_, err := ep(ictxWithFile("f.xlsx", data), nil)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, Rec{Sheet: "P", Val: "hello"}, got[0])
	assert.Equal(t, Rec{Sheet: "Q", Val: "world"}, got[1])
}

func TestMiddleware_MultiSheet_IgnoreError_ContinuesAcrossSheets(t *testing.T) {
	data := makeXLSXMulti(t, []sheetDef{
		{name: "A", rows: [][]string{{"c"}, {"1"}, {"2"}}},
		{name: "B", rows: [][]string{{"c"}, {"3"}}},
	})

	callCount := 0
	ep := func(_ context.Context, req any) (any, error) {
		if req == nil {
			return nil, nil
		}
		callCount++
		return nil, fmt.Errorf("err%d", callCount)
	}

	_, err := Middleware("f.xlsx", AllSheets, nil, true)(ep)(ictxWithFile("f.xlsx", data), nil)
	assert.Equal(t, 3, callCount, "all rows across all sheets should be processed")
	require.Error(t, err)
}

func TestMiddleware_MultiSheet_StopOnError_StopsAcrossSheets(t *testing.T) {
	data := makeXLSXMulti(t, []sheetDef{
		{name: "A", rows: [][]string{{"c"}, {"1"}, {"2"}}},
		{name: "B", rows: [][]string{{"c"}, {"3"}}},
	})

	callCount := 0
	ep := func(_ context.Context, req any) (any, error) {
		if req == nil {
			return nil, nil
		}
		callCount++
		return nil, errors.New("stop")
	}

	_, err := Middleware("f.xlsx", AllSheets, nil, false)(ep)(ictxWithFile("f.xlsx", data), nil)
	assert.Equal(t, 1, callCount, "should stop on first error, not continue to next sheet")
	require.Error(t, err)
}

func TestMiddleware_MultiSheet_OneSheetEmpty_OtherReadFully(t *testing.T) {
	data := makeXLSXMulti(t, []sheetDef{
		{name: "Empty", rows: [][]string{{"col"}}},         // header only
		{name: "Full", rows: [][]string{{"col"}, {"val"}}}, // one data row
	})

	rows, err := run(t, Middleware("f.xlsx", AllSheets, nil, false), data, "f.xlsx")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Full", rows[0][RowKeySheetName])
}

func TestMiddleware_MultiSheet_UnknownSheetInList_ReturnsError(t *testing.T) {
	data := makeXLSXMulti(t, []sheetDef{
		{name: "Real", rows: [][]string{{"c"}, {"v"}}},
	})
	mw := Middleware("f.xlsx", []string{"Real", "Ghost"}, nil, false)
	_, err := mw(func(_ context.Context, _ any) (any, error) { return nil, nil })(ictxWithFile("f.xlsx", data), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Ghost")
	assert.Contains(t, err.Error(), "not found")
}

func TestMiddleware_MultiSheet_DifferentHeaders(t *testing.T) {
	// Each sheet can have completely different columns.
	data := makeXLSXMulti(t, []sheetDef{
		{name: "Users", rows: [][]string{{"name", "email"}, {"Alice", "a@x.com"}}},
		{name: "Orders", rows: [][]string{{"order_id", "total"}, {"ORD1", "99.9"}}},
	})

	rows, err := run(t, Middleware("f.xlsx", AllSheets, nil, false), data, "f.xlsx")
	require.NoError(t, err)
	require.Len(t, rows, 2)

	// Users row has name/email keys
	assert.Equal(t, "Alice", rows[0]["name"])
	assert.Equal(t, "a@x.com", rows[0]["email"])
	assert.Nil(t, rows[0]["order_id"]) // not present in Users sheet

	// Orders row has order_id/total keys
	assert.Equal(t, "ORD1", rows[1]["order_id"])
	assert.Equal(t, "99.9", rows[1]["total"])
	assert.Nil(t, rows[1]["name"]) // not present in Orders sheet
}

// ─────────────────────────────────────────────────────────────────────────────
// resolveSheets unit tests
// ─────────────────────────────────────────────────────────────────────────────

func TestResolveSheets_SingleEmptyString_ReturnsFirst(t *testing.T) {
	data := makeXLSXMulti(t, []sheetDef{
		{name: "First", rows: nil},
		{name: "Second", rows: nil},
	})
	r, err := xlsxlite.OpenReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	defer r.Close()

	got, err := resolveSheets(r, []string{""})
	require.NoError(t, err)
	assert.Equal(t, []string{"First"}, got)
}

func TestResolveSheets_Nil_ReturnsAll(t *testing.T) {
	data := makeXLSXMulti(t, []sheetDef{
		{name: "A", rows: nil},
		{name: "B", rows: nil},
		{name: "C", rows: nil},
	})
	r, err := xlsxlite.OpenReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	defer r.Close()

	got, err := resolveSheets(r, nil)
	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B", "C"}, got)
}

func TestResolveSheets_ExplicitList_Validated(t *testing.T) {
	data := makeXLSXMulti(t, []sheetDef{{name: "Only", rows: nil}})
	r, err := xlsxlite.OpenReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	defer r.Close()

	// Valid
	got, err := resolveSheets(r, []string{"Only"})
	require.NoError(t, err)
	assert.Equal(t, []string{"Only"}, got)

	// Invalid
	_, err = resolveSheets(r, []string{"Only", "Missing"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Missing")
}

// ─────────────────────────────────────────────────────────────────────────────
// Unexported helper unit tests
// ─────────────────────────────────────────────────────────────────────────────

func TestIsRowEmpty(t *testing.T) {
	tests := []struct {
		name  string
		cells []xlsxlite.Cell
		want  bool
	}{
		{"all nil", []xlsxlite.Cell{{Value: nil}, {Value: nil}}, true},
		{"all empty string", []xlsxlite.Cell{{Value: ""}, {Value: ""}}, true},
		{"one non-empty", []xlsxlite.Cell{{Value: nil}, {Value: "x"}}, false},
		{"number value", []xlsxlite.Cell{{Value: 42.0}}, false},
		{"no cells", []xlsxlite.Cell{}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isRowEmpty(tc.cells))
		})
	}
}

func TestBuildRowMap(t *testing.T) {
	columns := []string{"name", "age"}
	cells := []xlsxlite.Cell{{Value: "Alice"}, {Value: 30.0}}
	row := buildRowMap(columns, cells, 5, "Sheet1")
	assert.Equal(t, "Alice", row["name"])
	assert.Equal(t, 30.0, row["age"])
	assert.Equal(t, 5, row[RowKeyLineNumber])
	assert.Equal(t, "Sheet1", row[RowKeySheetName])
}

func TestBuildRowMap_ShortRow(t *testing.T) {
	columns := []string{"a", "b", "c"}
	cells := []xlsxlite.Cell{{Value: "only"}}
	row := buildRowMap(columns, cells, 2, "S")
	assert.Equal(t, "only", row["a"])
	assert.Nil(t, row["b"])
	assert.Nil(t, row["c"])
}

func TestDeduplicateErrors(t *testing.T) {
	errs := []error{errors.New("foo"), errors.New("bar"), errors.New("foo")}
	result := deduplicateErrors(errs)
	require.NotNil(t, result)
	assert.Contains(t, result.Error(), "foo")
	assert.Contains(t, result.Error(), "bar")
	assert.Equal(t, len("foobar"), len(result.Error()))
}

func TestDeduplicateErrors_AllSame(t *testing.T) {
	result := deduplicateErrors([]error{errors.New("x"), errors.New("x")})
	assert.Equal(t, "x", result.Error())
}

func TestDeduplicateErrors_Empty(t *testing.T) {
	assert.Nil(t, deduplicateErrors(nil))
	assert.Nil(t, deduplicateErrors([]error{}))
}

// ─────────────────────────────────────────────────────────────────────────────
// makeXLSX integrity check
// ─────────────────────────────────────────────────────────────────────────────

func TestMakeXLSX_ProducesValidZip(t *testing.T) {
	data := makeXLSX(t, [][]string{{"h1"}, {"v1"}})
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	assert.Contains(t, names, "xl/workbook.xml")

	r, err := xlsxlite.OpenReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	defer r.Close()

	iter, err := r.OpenSheet("Sheet1")
	require.NoError(t, err)
	defer iter.Close()

	var rows []*xlsxlite.Row
	for iter.Next() {
		rows = append(rows, iter.Row())
	}
	require.NoError(t, iter.Err())
	require.Len(t, rows, 2) // header + 1 data row

	rc, err := io.ReadAll(bytes.NewReader(data))
	require.NoError(t, err)
	assert.NotEmpty(t, rc)
}
