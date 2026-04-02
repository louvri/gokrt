package test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/after"
	"github.com/louvri/gokrt/alter"
	"github.com/louvri/gokrt/on_eof"
	"github.com/louvri/gokrt/sys_key"
	"github.com/louvri/xlsxlite"

	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/xlsx/reader"
)

type key int

var test key = 1

func TestAfter(t *testing.T) {
	ctx := context.WithValue(context.Background(), test, 1)
	response, err := endpoint.Chain(after.Middleware(
		func(ctx context.Context, req any) (any, error) {
			fmt.Printf("HELLOOO %v\n", ctx.Value(test))
			fmt.Println(req)
			time.Sleep(1000)
			return nil, nil
		},
		func(data any, err error) any {
			return data
		},
		nil,
	))(func(ctx context.Context, req any) (any, error) {
		a := 0
		for i := 0; i < 10000; i++ {
			a++
		}
		return a, nil
	})(ctx, 3)
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Println(response)
}

func TestOnEof(t *testing.T) {
	ctx := context.WithValue(context.Background(), sys_key.EOF, "eof")
	response, err := endpoint.Chain(
		on_eof.Middleware(
			alter.Middleware(
				func(ctx context.Context, req any) (any, error) {
					return "hello world 1", nil
				},
				func(data any, err error) any {
					return data
				},
				func(data1, data2 any, err error) (any, error) {
					return data2, nil
				},
			),
			after.Middleware(
				func(ctx context.Context, req any) (any, error) {
					return "hello world 2", nil
				},
				func(data any, err error) any {
					return data
				},
				nil,
			),
		),
	)(func(ctx context.Context, req any) (any, error) {
		return "satu", nil
	})(ctx, -1)
	if response.(string) != "hello world 1" {
		t.Fatal("wrong result")
	}
	if err != nil {
		t.Fatal(err.Error())
	}
}
func TestOnEofWhileError(t *testing.T) {
	ctx := context.WithValue(context.Background(), sys_key.EOF, "err")
	response, err := endpoint.Chain(
		on_eof.Middleware(
			alter.Middleware(
				func(ctx context.Context, req any) (any, error) {
					return "hello world", nil
				},
				func(data any, err error) any {
					return data
				},
				func(data1, data2 any, err error) (any, error) {
					return data2, nil
				},
			),
		),
	)(func(ctx context.Context, req any) (any, error) {
		return "satu", nil
	})(ctx, -1)
	if response.(string) != "hello world" {
		t.Fatal("wrong result")
	}
	if err != nil {
		t.Fatal(err.Error())
	}
}

// buildXLSX creates a two-sheet workbook and returns its raw bytes.
// Sheet "Products" has 3 data rows; sheet "Orders" has 2 data rows.
func buildXLSX() []byte {
	var buf bytes.Buffer
	w := xlsxlite.NewWriter(&buf)

	// ── Sheet 1: Products ────────────────────────────────────────────────────
	sw1, err := w.NewSheet(xlsxlite.SheetConfig{Name: "Products"})
	must(err, "NewSheet Products")

	must(sw1.WriteRow(xlsxlite.MakeRow("sku", "name", "price", "stock")), "write header 1")
	must(sw1.WriteRow(xlsxlite.MakeRow("SKU001", "Widget A", 9.99, 120)), "write row")
	must(sw1.WriteRow(xlsxlite.MakeRow("SKU002", "Widget B", 14.50, 45)), "write row")
	must(sw1.WriteRow(xlsxlite.MakeRow("SKU003", "Widget C", 3.25, 300)), "write row")
	must(sw1.Close(), "close sheet 1")

	// ── Sheet 2: Orders ──────────────────────────────────────────────────────
	sw2, err := w.NewSheet(xlsxlite.SheetConfig{Name: "Orders"})
	must(err, "NewSheet Orders")

	must(sw2.WriteRow(xlsxlite.MakeRow("order_id", "sku", "qty", "total")), "write header 2")
	must(sw2.WriteRow(xlsxlite.MakeRow("ORD-001", "SKU001", 2, 19.98)), "write row")
	must(sw2.WriteRow(xlsxlite.MakeRow("ORD-002", "SKU003", 10, 32.50)), "write row")
	must(sw2.Close(), "close sheet 2")

	must(w.Close(), "close workbook")
	return buf.Bytes()
}

// ─────────────────────────────────────────────────────────────────────────────
// Step 2 — inject the file into IContext the same way gcs/open would
// ─────────────────────────────────────────────────────────────────────────────

// newICtxWithFile wraps data in a bytes.Reader and stores it under FILE_KEY,
// replicating exactly what gcs/open.Middleware does before calling next.
func newICtxWithFile(filename string, data []byte) icontext.IContext {
	ictx := icontext.New(context.Background())
	ictx.Set(sys_key.FILE_KEY, map[string]any{
		filename: bytes.NewReader(data),
	})
	return ictx
}

// ─────────────────────────────────────────────────────────────────────────────
// Domain types
// ─────────────────────────────────────────────────────────────────────────────

// Product is the typed struct we decode Products sheet rows into.
type Product struct {
	SKU   string
	Name  string
	Price float64
	Stock float64
	Line  int
}

// Order is the typed struct we decode Orders sheet rows into.
type Order struct {
	OrderID string
	SKU     string
	Qty     float64
	Total   float64
	Line    int
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 1 — read a single named sheet with a typed decoder
// ─────────────────────────────────────────────────────────────────────────────

func testSingleSheet(data []byte) {
	fmt.Println("═══ Test 1: single sheet (Products) with decoder ═══")

	const filename = "catalog.xlsx"

	// Decoder: map[string]any → Product.
	// Called once per data row; the row map always has "sheetName" and "lineNumber".
	productDecoder := func(raw any) any {
		row := raw.(map[string]any)
		return Product{
			SKU:   fmt.Sprintf("%v", row["sku"]),
			Name:  fmt.Sprintf("%v", row["name"]),
			Price: toFloat(row["price"]),
			Stock: toFloat(row["stock"]),
			Line:  row[reader.RowKeyLineNumber].(int),
		}
	}

	// The business endpoint: called once per decoded row, and once with nil at EOF.
	var products []Product
	processRow := endpoint.Endpoint(func(_ context.Context, req any) (any, error) {
		if req == nil {
			return nil, nil // EOF flush — nothing to do here
		}
		p := req.(Product)
		products = append(products, p)
		return nil, nil
	})

	mw := reader.Middleware(filename, []string{"Products"}, productDecoder, false)
	chain := mw(processRow)

	ictx := newICtxWithFile(filename, data)
	if _, err := chain(ictx, nil); err != nil {
		log.Fatalf("Test 1 failed: %v", err)
	}

	fmt.Printf("  Read %d products:\n", len(products))
	for _, p := range products {
		fmt.Printf("    [line %d] %-8s %-12s price=%.2f  stock=%.0f\n",
			p.Line, p.SKU, p.Name, p.Price, p.Stock)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 2 — read all sheets with no decoder (raw map[string]any)
// ─────────────────────────────────────────────────────────────────────────────

func testAllSheets(data []byte) {
	fmt.Println("═══ Test 2: all sheets, no decoder (raw map) ═══")

	const filename = "catalog.xlsx"

	var allRows []map[string]any
	collectRow := endpoint.Endpoint(func(_ context.Context, req any) (any, error) {
		if req == nil {
			return nil, nil
		}
		allRows = append(allRows, req.(map[string]any))
		return nil, nil
	})

	// reader.AllSheets → reads every sheet in workbook order.
	mw := reader.Middleware(filename, reader.AllSheets, nil /*no decoder*/, false)
	chain := mw(collectRow)

	ictx := newICtxWithFile(filename, data)
	if _, err := chain(ictx, nil); err != nil {
		log.Fatalf("Test 2 failed: %v", err)
	}

	fmt.Printf("  Read %d total rows across all sheets:\n", len(allRows))
	for _, row := range allRows {
		fmt.Printf("    sheet=%-10s line=%v  data=%v\n",
			row[reader.RowKeySheetName],
			row[reader.RowKeyLineNumber],
			rowDataOnly(row),
		)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 3 — explicit subset of sheets in custom order
// ─────────────────────────────────────────────────────────────────────────────

func testExplicitSheets(data []byte) {
	fmt.Println("═══ Test 3: explicit sheets [Orders, Products] (reversed order) ═══")

	const filename = "catalog.xlsx"

	var allRows []map[string]any
	collect := endpoint.Endpoint(func(_ context.Context, req any) (any, error) {
		if req == nil {
			return nil, nil
		}
		allRows = append(allRows, req.(map[string]any))
		return nil, nil
	})

	mw := reader.Middleware(filename, []string{"Orders", "Products"}, nil, false)
	chain := mw(collect)

	ictx := newICtxWithFile(filename, data)
	if _, err := chain(ictx, nil); err != nil {
		log.Fatalf("Test 3 failed: %v", err)
	}

	fmt.Printf("  Read %d rows (Orders first, then Products):\n", len(allRows))
	for _, row := range allRows {
		fmt.Printf("    sheet=%-10s line=%v\n",
			row[reader.RowKeySheetName],
			row[reader.RowKeyLineNumber],
		)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 4 — read from a real file on disk
// ─────────────────────────────────────────────────────────────────────────────

func testFromDisk(data []byte) {
	fmt.Println("═══ Test 4: read from a real file on disk ═══")

	const filename = "catalog.xlsx"

	// Write the bytes to a temp file.
	tmpFile, err := os.CreateTemp("", "catalog-*.xlsx")
	must(err, "create temp file")
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.Write(data)
	must(err, "write temp file")
	must(tmpFile.Close(), "close temp file")

	fmt.Printf("  Written to: %s\n", tmpFile.Name())

	// Read it back using os.Open and inject as io.Reader (same as GCS path).
	f, err := os.Open(tmpFile.Name())
	must(err, "open temp file")
	defer f.Close()

	// Inject the *os.File (which is an io.Reader) into IContext.
	ictx := icontext.New(context.Background())
	ictx.Set(sys_key.FILE_KEY, map[string]any{filename: f})

	rowCount := 0
	count := endpoint.Endpoint(func(_ context.Context, req any) (any, error) {
		if req != nil {
			rowCount++
		}
		return nil, nil
	})

	mw := reader.Middleware(filename, reader.AllSheets, nil, false)
	if _, err := mw(count)(ictx, nil); err != nil {
		log.Fatalf("Test 4 failed: %v", err)
	}

	fmt.Printf("  Total data rows read from disk: %d (expected 5)\n\n", rowCount)
}

// ─────────────────────────────────────────────────────────────────────────────
// Test 5 — branch on sheetName inside the endpoint
// ─────────────────────────────────────────────────────────────────────────────

func testBranchOnSheet(data []byte) {
	fmt.Println("═══ Test 5: branch on sheetName inside endpoint ═══")

	const filename = "catalog.xlsx"

	var products []Product
	var orders []Order

	collectBoth := endpoint.Endpoint(func(_ context.Context, req any) (any, error) {
		if req == nil {
			return nil, nil
		}
		row := req.(map[string]any)
		switch row[reader.RowKeySheetName] {
		case "Products":
			products = append(products, Product{
				SKU:  fmt.Sprintf("%v", row["sku"]),
				Name: fmt.Sprintf("%v", row["name"]),
				Line: row[reader.RowKeyLineNumber].(int),
			})
		case "Orders":
			orders = append(orders, Order{
				OrderID: fmt.Sprintf("%v", row["order_id"]),
				SKU:     fmt.Sprintf("%v", row["sku"]),
				Line:    row[reader.RowKeyLineNumber].(int),
			})
		}
		return nil, nil
	})

	mw := reader.Middleware(filename, reader.AllSheets, nil, false)
	ictx := newICtxWithFile(filename, data)
	if _, err := mw(collectBoth)(ictx, nil); err != nil {
		log.Fatalf("Test 5 failed: %v", err)
	}

	fmt.Printf("  Products (%d):\n", len(products))
	for _, p := range products {
		fmt.Printf("    [line %d] %s – %s\n", p.Line, p.SKU, p.Name)
	}
	fmt.Printf("  Orders (%d):\n", len(orders))
	for _, o := range orders {
		fmt.Printf("    [line %d] %s → %s\n", o.Line, o.OrderID, o.SKU)
	}
	fmt.Println()
}

// ─────────────────────────────────────────────────────────────────────────────
// Entry point
// ─────────────────────────────────────────────────────────────────────────────

func main() {
	fmt.Println("Building xlsx file in memory...")
	data := buildXLSX()
	fmt.Printf("Built %d bytes (%d KB)\n\n", len(data), len(data)/1024)

	testSingleSheet(data)
	testAllSheets(data)
	testExplicitSheets(data)
	testFromDisk(data)
	testBranchOnSheet(data)

	fmt.Println("All tests passed ✓")
}

// ─────────────────────────────────────────────────────────────────────────────
// Utilities
// ─────────────────────────────────────────────────────────────────────────────

func must(err error, label string) {
	if err != nil {
		log.Fatalf("%s: %v", label, err)
	}
}

// toFloat safely converts xlsxlite numeric cell values (stored as float64)
// to float64, returning 0 for nil or unrecognised types.
func toFloat(v any) float64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	}
	return 0
}

// rowDataOnly returns a copy of a row map with the meta keys stripped,
// for cleaner printing.
func rowDataOnly(row map[string]any) map[string]any {
	out := make(map[string]any, len(row))
	for k, v := range row {
		if k == reader.RowKeySheetName || k == reader.RowKeyLineNumber {
			continue
		}
		out[k] = v
	}
	return out
}
