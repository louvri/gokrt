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
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(filename, splitter string, columns []string, cancelOnError bool) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			splitterRune := ';'
			if splitter != "" {
				r := []rune(splitter)
				splitterRune = r[0]
			}
			response, responseError := next(ctx, req)
			if responseError != nil && cancelOnError {
				if tmp, ok := ctx.Value(sys_key.FILE_OBJECT_KEY).(map[string]interface{}); ok {
					if con, ok := tmp[filename].(connection.Connection); ok {
						con.Cancel()
					}
				}
				return response, responseError
			}
			eof := ctx.Value(sys_key.EOF)
			if eof != nil && eof != "" {
				if eof != "eof" {
					if tmp, ok := ctx.Value(sys_key.FILE_OBJECT_KEY).(map[string]interface{}); ok {
						if con, ok := tmp[filename].(connection.Connection); ok {
							con.Cancel()
						}
					}
				}
				return response, responseError
			}
			var writer *bufio.Writer
			if tmp := ctx.Value(sys_key.FILE_KEY).(map[string]interface{}); tmp != nil {
				writer = bufio.NewWriter(tmp[filename].(io.Writer))
			} else {
				return nil, errors.New("csv_writer_middleware: connection not initialized")
			}
			var tobeRendered []map[string]interface{}
			if tmp, ok := response.(map[string]interface{}); ok {
				tobeRendered = make([]map[string]interface{}, 0)
				tobeRendered = append(tobeRendered, tmp)
			} else if tmp, ok := response.([]map[string]interface{}); ok {
				tobeRendered = tmp
			}
			if first, ok := ctx.Value(sys_key.SOF).(bool); ok {
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
