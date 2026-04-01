// Package writer provides a middleware-style Excel (XLSX) writer that integrates
// with the gokrt go-kit endpoint chain.
//
// Design mirrors gokrt/csv/writer exactly:
//   - Calls next(ctx, req) first to obtain the data to write
//   - Reads the io.Writer from IContext (placed there by gcs/open.Middleware)
//   - Writes header row on the first call (SOF = true), then data rows
//   - Cancels the GCS connection on error when cancelOnError is true
//   - Finalises the XLSX workbook on EOF signal
//
// Typical chain (writing to GCS):
//
//	endpoint.Chain(
//	    gcsopen.Middleware(bucket, object, cred, gcs.WRITER),
//	    writer.Middleware(object, "Sheet1", columns, true),
//	    gcsclose.Middleware(),
//	)(yourEndpoint)
package writer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/connection"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
	"github.com/louvri/xlsxlite"
)

// sheetState holds the xlsxlite writer and sheet writer for one file+sheet
// combination. It is kept alive across calls because xlsxlite writes
// sequentially: the sheet must stay open until all rows are flushed.
type sheetState struct {
	mu         sync.Mutex
	xlsxW      *xlsxlite.Writer
	sheetW     *xlsxlite.SheetWriter
	headerDone bool
}

// stateKey is used as the context key for the sheetState so it survives across
// repeated calls within the same chain invocation (stored on IContext via Set).
type stateKey struct {
	filename string
	sheet    string
}

// Middleware returns a go-kit endpoint.Middleware that writes one or more
// Excel rows per invocation and finalises the workbook at EOF.
//
// Parameters:
//   - filename:      GCS object name; must match the name given to gcs/open.Middleware.
//   - sheet:         worksheet name (e.g. "Sheet1"). Cannot be empty.
//   - columns:       ordered list of map keys to extract from each row.
//     These also become the header row.
//   - cancelOnError: when true, calls Connection.Cancel() on the first error
//     from the inner endpoint, aborting the GCS upload.
//
// The inner endpoint (next) must return one of:
//   - map[string]any        – a single row
//   - []map[string]any      – multiple rows
//   - nil                   – nothing to write this call (no-op)
func Middleware(filename, sheet string, columns []string, cancelOnError bool) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {

			// ── 1. Ensure IContext ────────────────────────────────────────────
			var ictx icontext.IContext
			if tmp, ok := ctx.(icontext.IContext); ok {
				ictx = tmp
			} else {
				ictx = icontext.New(ctx)
			}

			// ── 2. Call the inner endpoint first (mirrors csv/writer) ─────────
			response, responseError := next(ictx, req)

			// ── 3. Cancel GCS connection on error when configured ─────────────
			if responseError != nil && cancelOnError {
				cancelConnection(ictx, filename)
				return response, responseError
			}

			// ── 4. On EOF: finalise the workbook and return ───────────────────
			eof := ictx.Get(sys_key.EOF)
			if eof != nil && eof != "" {
				if eof != "eof" {
					// Abnormal EOF — cancel rather than finalise.
					cancelConnection(ictx, filename)
				} else {
					// Normal EOF — close the sheet and workbook cleanly.
					if err := finalise(ictx, filename, sheet); err != nil {
						return nil, fmt.Errorf("xlsx_writer_middleware: finalise: %w", err)
					}
				}
				return response, responseError
			}

			// ── 5. Obtain the underlying io.Writer from IContext ──────────────
			w, err := resolveWriter(ictx, filename)
			if err != nil {
				return nil, err
			}

			// ── 6. Get-or-create the per-invocation sheet state ───────────────
			state, err := getOrCreateState(ictx, w, filename, sheet)
			if err != nil {
				return nil, err
			}

			// ── 7. Write header on the very first row (SOF) ───────────────────
			state.mu.Lock()
			defer state.mu.Unlock()

			if sof, ok := ictx.Get(sys_key.SOF).(bool); ok && sof && !state.headerDone {
				headerCells := makeHeaderCells(columns)
				if writeErr := state.sheetW.WriteRow(xlsxlite.Row{Cells: headerCells}); writeErr != nil {
					return nil, fmt.Errorf("xlsx_writer_middleware: write header: %w", writeErr)
				}
				state.headerDone = true
			}

			// ── 8. Normalise response to []map[string]any ─────────────────────
			rows := toRows(response)

			// ── 9. Write each data row ────────────────────────────────────────
			for _, data := range rows {
				cells := makeDataCells(columns, data)
				if writeErr := state.sheetW.WriteRow(xlsxlite.Row{Cells: cells}); writeErr != nil {
					return nil, fmt.Errorf("xlsx_writer_middleware: write row: %w", writeErr)
				}
			}

			return response, responseError
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

// cancelConnection calls Cancel on the GCS Connection stored in FILE_OBJECT_KEY,
// mirroring the csv/writer cancellation pattern.
func cancelConnection(ictx icontext.IContext, filename string) {
	if tmp, ok := ictx.Get(sys_key.FILE_OBJECT_KEY).(map[string]any); ok {
		if con, ok := tmp[filename].(connection.Connection); ok {
			con.Cancel()
		}
	}
}

// finalise closes the SheetWriter and the xlsxlite.Writer so all XML is flushed
// to the underlying io.Writer before gcs/close seals the GCS object.
func finalise(ictx icontext.IContext, filename, sheet string) error {
	key := stateKey{filename: filename, sheet: sheet}
	raw := ictx.Value(key)
	if raw == nil {
		// Nothing was ever written — still need to produce a valid workbook.
		w, err := resolveWriter(ictx, filename)
		if err != nil {
			return err
		}
		state, err := getOrCreateState(ictx, w, filename, sheet)
		if err != nil {
			return err
		}
		raw = state
	}
	state, ok := raw.(*sheetState)
	if !ok {
		return errors.New("xlsx_writer_middleware: invalid sheet state type")
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if closeErr := state.sheetW.Close(); closeErr != nil {
		return fmt.Errorf("close sheet: %w", closeErr)
	}
	if closeErr := state.xlsxW.Close(); closeErr != nil {
		return fmt.Errorf("close workbook: %w", closeErr)
	}
	return nil
}

// resolveWriter retrieves the io.Writer stored in FILE_KEY by gcs/open.
func resolveWriter(ictx icontext.IContext, filename string) (io.Writer, error) {
	fileMap, ok := ictx.Get(sys_key.FILE_KEY).(map[string]any)
	if !ok || fileMap == nil {
		return nil, errors.New("xlsx_writer_middleware: FILE_KEY not found in context; ensure gcs/open.Middleware runs first")
	}
	w, ok := fileMap[filename].(io.Writer)
	if !ok || w == nil {
		return nil, fmt.Errorf("xlsx_writer_middleware: no writer found for file %q in FILE_KEY", filename)
	}
	return w, nil
}

// getOrCreateState returns the sheetState for this filename+sheet, creating it
// (and the underlying xlsxlite.Writer + SheetWriter) on the first call.
// State is stored on the IContext keyed by stateKey so it survives across the
// repeated calls that the csv reader drives through the chain.
func getOrCreateState(ictx icontext.IContext, w io.Writer, filename, sheet string) (*sheetState, error) {
	key := stateKey{filename: filename, sheet: sheet}
	if raw := ictx.Value(key); raw != nil {
		if state, ok := raw.(*sheetState); ok {
			return state, nil
		}
	}

	xlsxW := xlsxlite.NewWriter(w)
	sheetW, err := xlsxW.NewSheet(xlsxlite.SheetConfig{Name: sheet})
	if err != nil {
		return nil, fmt.Errorf("xlsx_writer_middleware: open sheet %q: %w", sheet, err)
	}

	state := &sheetState{
		xlsxW:  xlsxW,
		sheetW: sheetW,
	}

	// Store the state on the context so subsequent calls in the same chain
	// invocation reuse it. IContext.Set with a non-SysKey delegates to
	// context.WithValue on the base context.
	ictx.Set(key, state)
	return state, nil
}

// makeHeaderCells converts the column names into xlsxlite string cells.
func makeHeaderCells(columns []string) []xlsxlite.Cell {
	cells := make([]xlsxlite.Cell, len(columns))
	for i, col := range columns {
		cells[i] = xlsxlite.StringCell(col)
	}
	return cells
}

// makeDataCells extracts values from a map in column order.
// Missing keys produce an empty cell; values are formatted with %v so
// any type is handled without panicking.
func makeDataCells(columns []string, data map[string]any) []xlsxlite.Cell {
	cells := make([]xlsxlite.Cell, len(columns))
	for i, col := range columns {
		v := data[col]
		cells[i] = toCell(v)
	}
	return cells
}

// toCell converts a Go value to the most appropriate xlsxlite Cell type,
// avoiding the string round-trip where possible.
func toCell(v any) xlsxlite.Cell {
	if v == nil {
		return xlsxlite.EmptyCell()
	}
	switch val := v.(type) {
	case string:
		return xlsxlite.StringCell(val)
	case float64:
		return xlsxlite.NumberCell(val)
	case float32:
		return xlsxlite.NumberCell(float64(val))
	case int:
		return xlsxlite.IntCell(val)
	case int64:
		return xlsxlite.IntCell(int(val))
	case int32:
		return xlsxlite.IntCell(int(val))
	case bool:
		return xlsxlite.BoolCell(val)
	default:
		// Fallback: stringify any remaining type (e.g. time.Time, custom types).
		return xlsxlite.StringCell(fmt.Sprintf("%v", val))
	}
}

// toRows normalises the response from the inner endpoint into a slice of
// map[string]any rows, matching csv/writer's tobeRendered logic exactly.
func toRows(response any) []map[string]any {
	switch v := response.(type) {
	case map[string]any:
		return []map[string]any{v}
	case []map[string]any:
		return v
	default:
		return nil
	}
}
