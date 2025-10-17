package writer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/connection"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(filename string, columns []string, cancelOnError bool, splitter ...string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			splitterRune := ';'
			if len(splitter) > 0 && splitter[0] != "" {
				r := []rune(splitter[0])
				splitterRune = r[0]
			}
			var ok bool
			var ictx *icontext.Context
			if ictx, ok = ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context); !ok {
				ictx = icontext.New(ctx).(*icontext.Context)
			}
			response, responseError := next(ictx, req)
			if responseError != nil && cancelOnError {
				if tmp, ok := ictx.Get(sys_key.FILE_OBJECT_KEY).(map[string]any); ok {
					if con, ok := tmp[filename].(connection.Connection); ok {
						con.Cancel()
					}
				}
				return response, responseError
			}
			eof := ictx.Get(sys_key.EOF)
			if eof != nil && eof != "" {
				if eof != "eof" {
					if tmp, ok := ictx.Get(sys_key.FILE_OBJECT_KEY).(map[string]any); ok {
						if con, ok := tmp[filename].(connection.Connection); ok {
							con.Cancel()
						}
					}
				}
				return response, responseError
			}
			var writer *bufio.Writer
			if tmp := ictx.Get(sys_key.FILE_KEY).(map[string]any); tmp != nil {
				writer = bufio.NewWriter(tmp[filename].(io.Writer))
			} else {
				return nil, errors.New("csv_writer_middleware: connection not initialized")
			}
			var tobeRendered []map[string]any
			if tmp, ok := response.(map[string]any); ok {
				tobeRendered = make([]map[string]any, 0)
				tobeRendered = append(tobeRendered, tmp)
			} else if tmp, ok := response.([]map[string]any); ok {
				tobeRendered = tmp
			}
			if first, ok := ictx.Get(sys_key.SOF).(bool); ok {
				if first {
					var str strings.Builder
					for i, key := range columns {
						if i > 0 {
							str.WriteRune(splitterRune)
						}
						str.WriteString(key)
					}
					if _, err := writer.WriteString(str.String()); err != nil {
						return nil, err
					}
					writer.WriteRune('\n')
				}
			}
			for _, data := range tobeRendered {
				var str strings.Builder
				for i, key := range columns {
					item := data[key]
					if i > 0 {
						str.WriteRune(splitterRune)
					}
					_, err := str.WriteString(fmt.Sprintf("%v", item))
					if err != nil {
						return nil, err
					}
				}
				if _, err := writer.WriteString(str.String()); err != nil {
					return nil, err
				}
				if _, err := writer.WriteRune('\n'); err != nil {
					return nil, err
				}
				if err := writer.Flush(); err != nil {
					return nil, err
				}
			}
			return response, responseError
		}
	}
}
