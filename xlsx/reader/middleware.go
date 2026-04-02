// Package reader provides a middleware-style Excel (XLSX) reader that integrates
// with the gokrt go-kit endpoint chain.
// Single-sheet usage:
//
//	endpoint.Chain(
//	    gcsopen.Middleware(bucket, object, cred, gcs.READER),
//	    reader.Middleware(object, "Invoices", decoder, false),
//	    gcsclose.Middleware(),
//	)(myEndpoint)
//
// Multi-sheet usage — pass the sheet names you want, or none for all sheets:
//
//	endpoint.Chain(
//	    gcsopen.Middleware(bucket, object, cred, gcs.READER),
//	    reader.Middleware(object, "Sheet1", "Sheet2", decoder, false),
//	    gcsclose.Middleware(),
//	)(myEndpoint)
//
// Each row map gains a "sheetName" key so the next endpoint can branch on it.
package reader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
	"github.com/louvri/xlsxlite"
)

// RowKeySheetName is injected into every row map so the next endpoint can
// identify which sheet the row came from when reading multiple sheets.
const RowKeySheetName = "sheetName"

// RowKeyLineNumber is injected into every row map (1-based, header excluded),
// matching the csv/reader convention.
const RowKeyLineNumber = "lineNumber"

// Middleware returns a go-kit endpoint.Middleware that reads one or more
// Excel worksheets row by row and calls the next endpoint once per
// non-empty data row.
//
// Parameters:
//   - filename:    GCS object name; must match the name given to gcs/open.Middleware.
//   - sheets:      worksheet names to read, in order. Pass no names (or a single
//     empty string "") to read only the first sheet. Pass multiple
//     names to iterate them sequentially. Pass AllSheets to read
//     every sheet in workbook order.
//   - decoder:     optional transform from raw map[string]any → domain type.
//     Receives the row map (including "sheetName" and "lineNumber").
//     Pass nil to forward the raw map unchanged.
//   - ignoreError: when true, per-row errors are collected and returned as a
//     single combined error at the end (csv/reader behaviour).
//     When false, the first error stops iteration immediately.
func Middleware(
	filename string,
	sheets []string,
	decoder func(data any) any,
	ignoreError bool,
) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {

			// ── 1. Ensure we have an IContext ─────────────────────────────────
			var ictx icontext.IContext
			if tmp, ok := ctx.(icontext.IContext); ok {
				ictx = tmp
			} else {
				ictx = icontext.New(ctx)
			}

			// ── 2. Retrieve the io.Reader injected by gcs/open ────────────────
			fileMap, ok := ictx.Get(sys_key.FILE_KEY).(map[string]any)
			if !ok || fileMap == nil {
				return nil, errors.New("xlsx_reader_middleware: FILE_KEY not found in context; ensure gcs/open.Middleware runs first")
			}
			rawReader, ok := fileMap[filename].(io.Reader)
			if !ok || rawReader == nil {
				return nil, fmt.Errorf("xlsx_reader_middleware: no reader found for file %q in FILE_KEY", filename)
			}

			// ── 3. Buffer into memory so xlsxlite can random-access the ZIP ──
			// GCS provides a plain io.Reader; xlsxlite.OpenReader needs io.ReaderAt.
			// Reading once into a bytes.Reader is the standard pattern (also what
			// xlsxlite/gcs does internally) and is bounded by MaxGCSDownloadSize.
			data, err := io.ReadAll(rawReader)
			if err != nil {
				return nil, fmt.Errorf("xlsx_reader_middleware: failed to read %q from GCS: %w", filename, err)
			}
			if len(data) == 0 {
				return nil, fmt.Errorf("xlsx_reader_middleware: file %q is empty", filename)
			}

			// ── 4. Open the XLSX workbook ──────────────────────────────────────
			xlsxReader, err := xlsxlite.OpenReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				return nil, fmt.Errorf("xlsx_reader_middleware: failed to parse xlsx %q: %w", filename, err)
			}
			defer xlsxReader.Close()

			// ── 5. Resolve the sheet list to iterate ──────────────────────────
			targets, err := resolveSheets(xlsxReader, sheets)
			if err != nil {
				return nil, fmt.Errorf("xlsx_reader_middleware: %w", err)
			}

			// ── 6. exec: call the next endpoint for one decoded row ───────────
			exec := func(row map[string]any) (any, error) {
				if len(row) == 0 {
					return nil, nil
				}
				var payload any = row
				if decoder != nil {
					payload = decoder(row)
				}
				return next(ictx, payload)
			}

			// ── 7. Iterate every target sheet sequentially ────────────────────
			var (
				response   any
				nextErrors []error
			)

			for sheetIdx, sheetName := range targets {
				isFirstSheet := sheetIdx == 0

				iter, openErr := xlsxReader.OpenSheet(sheetName)
				if openErr != nil {
					return nil, fmt.Errorf("xlsx_reader_middleware: failed to open sheet %q in %q: %w", sheetName, filename, openErr)
				}

				resp, errs, streamErr := streamSheet(ictx, iter, sheetName, isFirstSheet, exec, ignoreError)
				iter.Close() //nolint:errcheck // Close is always called; error is non-actionable here.

				if streamErr != nil {
					return nil, streamErr
				}
				nextErrors = append(nextErrors, errs...)
				if resp != nil {
					response = resp
				}
			}

			// ── 8. Flush / finalise call with empty row (mirrors csv/reader) ──
			if tmp, flushErr := exec(nil); flushErr != nil && !ignoreError {
				return nil, fmt.Errorf("xlsx_reader_middleware: flush: %w", flushErr)
			} else {
				if flushErr != nil {
					nextErrors = append(nextErrors, flushErr)
				}
				if tmp != nil {
					response = tmp
				}
			}

			// ── 9. Signal EOF; give next one final chance to act ──────────────
			ictx.Set(sys_key.EOF, "eof")
			if tmp, eofErr := next(ctx, nil); eofErr != nil {
				return nil, fmt.Errorf("xlsx_reader_middleware: eof signal: %w", eofErr)
			} else if tmp != nil {
				response = tmp
			}

			// ── 10. Surface collected per-row errors (deduplicated) ───────────
			if len(nextErrors) > 0 {
				return response, deduplicateErrors(nextErrors)
			}
			return response, nil
		}
	}
}

// AllSheets is a sentinel value for the sheets parameter of Middleware that
// causes every sheet in the workbook to be read in workbook order.
//
//	reader.Middleware(object, AllSheets, decoder, false)
var AllSheets = []string{}

// resolveSheets returns the ordered list of sheet names to iterate.
//
//   - nil or empty slice, or a single ""  → first sheet only (index 0)
//   - AllSheets (empty slice literal)     → all sheets in workbook order
//   - any other slice                     → exactly those sheets, in order given
//
// Because AllSheets and "first sheet only" are both empty slices the distinction
// is made by the caller using the exported sentinel; internally we treat both
// the same way: when the slice is empty we use all sheet names from the workbook,
// and when there is a single empty string we use only index 0.
func resolveSheets(r *xlsxlite.Reader, sheets []string) ([]string, error) {
	// Single empty string → first sheet only.
	if len(sheets) == 1 && sheets[0] == "" {
		if r.SheetCount() == 0 {
			return nil, errors.New("workbook contains no sheets")
		}
		return r.SheetNames()[:1], nil
	}

	// Nil, empty slice (including AllSheets sentinel) → all sheets.
	if len(sheets) == 0 {
		if r.SheetCount() == 0 {
			return nil, errors.New("workbook contains no sheets")
		}
		return r.SheetNames(), nil
	}

	// Explicit list: validate every name exists before we start streaming.
	known := make(map[string]struct{}, r.SheetCount())
	for _, n := range r.SheetNames() {
		known[n] = struct{}{}
	}
	for _, s := range sheets {
		if _, ok := known[s]; !ok {
			return nil, fmt.Errorf("sheet %q not found in workbook (available: %v)", s, r.SheetNames())
		}
	}
	return sheets, nil
}

// streamSheet streams all rows from a single open RowIterator.
// It returns the last non-nil response from the next endpoint, a slice of
// per-row errors (when ignoreError is true), and a fatal error that should
// stop the whole middleware (when ignoreError is false).
//
// SOF is set to true on the very first data row of the first sheet, and
// reset to false thereafter, preserving the csv/reader contract.
func streamSheet(
	ictx icontext.IContext,
	iter *xlsxlite.RowIterator,
	sheetName string,
	isFirstSheet bool,
	exec func(map[string]any) (any, error),
	ignoreError bool,
) (response any, errs []error, fatal error) {
	var (
		columns    []string
		first      = true
		lineNumber = 1
	)

	for iter.Next() {
		row := iter.Row()

		if first {
			// First row of this sheet → header
			columns = make([]string, len(row.Cells))
			for i, cell := range row.Cells {
				columns[i] = fmt.Sprintf("%v", cell.Value)
			}
			first = false
			if isFirstSheet {
				ictx.Set(sys_key.SOF, true)
			}
			lineNumber++
			time.Sleep(0) // cooperative yield, matches csv/reader
			continue
		}

		if isRowEmpty(row.Cells) {
			lineNumber++
			time.Sleep(0)
			continue
		}

		rowData := buildRowMap(columns, row.Cells, lineNumber, sheetName)

		resp, execErr := exec(rowData)
		if execErr != nil && !ignoreError {
			return nil, nil, fmt.Errorf("xlsx_reader_middleware: sheet %q row %d: %w", sheetName, lineNumber, execErr)
		} else if execErr != nil {
			errs = append(errs, execErr)
		}
		if resp != nil {
			response = resp
		}

		ictx.Set(sys_key.SOF, false)
		lineNumber++
		time.Sleep(0)
	}

	if iterErr := iter.Err(); iterErr != nil && !ignoreError {
		return nil, nil, fmt.Errorf("xlsx_reader_middleware: sheet %q iterator error: %w", sheetName, iterErr)
	} else if iterErr != nil {
		errs = append(errs, iterErr)
	}

	return response, errs, nil
}

// isRowEmpty returns true when every cell value is nil or an empty string.
func isRowEmpty(cells []xlsxlite.Cell) bool {
	for _, c := range cells {
		if c.Value != nil && fmt.Sprintf("%v", c.Value) != "" {
			return false
		}
	}
	return true
}

// buildRowMap builds the header-keyed map forwarded to the next endpoint.
// It always includes RowKeyLineNumber (1-based, header excluded) and
// RowKeySheetName so downstream code can identify the source sheet.
// When a data row is shorter than the header, missing columns are set to nil.
func buildRowMap(columns []string, cells []xlsxlite.Cell, lineNumber int, sheetName string) map[string]any {
	row := make(map[string]any, len(columns)+2)
	for i, col := range columns {
		if i < len(cells) {
			row[col] = cells[i].Value
		} else {
			row[col] = nil
		}
	}
	row[RowKeyLineNumber] = lineNumber
	row[RowKeySheetName] = sheetName
	return row
}

// deduplicateErrors joins a slice of errors into one, dropping exact-string
// duplicates — matching the csv/reader approach.
func deduplicateErrors(errs []error) error {
	seen := make(map[string]struct{}, len(errs))
	var combined string
	for _, e := range errs {
		s := e.Error()
		if _, dup := seen[s]; dup || s == "" {
			continue
		}
		seen[s] = struct{}{}
		combined += s
	}
	if combined == "" {
		return nil
	}
	return errors.New(combined)
}
