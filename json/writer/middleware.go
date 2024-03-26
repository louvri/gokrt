package writer

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/connection"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(filename string, columns []string, cancelOnError bool) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			response, responseError := next(ctx, req)
			if responseError != nil && cancelOnError {
				if tmp, ok := ctx.Value(sys_key.FILE_OBJECT_KEY).(map[string]interface{}); ok {
					if con, ok := tmp[filename].(connection.Connection); ok {
						con.Cancel()
					}
				}
				return response, responseError
			}
			var writer *bufio.Writer
			if tmp := ctx.Value(sys_key.FILE_KEY).(map[string]interface{}); tmp != nil {
				writer = bufio.NewWriter(tmp[filename].(io.Writer))
			} else {
				return nil, errors.New("json_writer_middleware: connection not initialized")
			}
			first := ctx.Value(sys_key.SOF)
			if tmp, ok := first.(bool); ok && tmp {
				writer.WriteRune('[')
				writer.WriteRune('\n')
			}
			eof := ctx.Value(sys_key.EOF)
			if eof != nil && eof != "" {
				writer.WriteRune(']')
				writer.Flush()
				return response, responseError
			}
			var tobeRendered []map[string]interface{}
			if tmp, ok := response.(map[string]interface{}); ok {
				tobeRendered = make([]map[string]interface{}, 0)
				tobeRendered = append(tobeRendered, tmp)
			} else if tmp, ok := response.([]map[string]interface{}); ok {
				tobeRendered = tmp
			}
			for _, data := range tobeRendered {
				filtered := make(map[string]interface{})
				for _, key := range columns {
					filtered[key] = data[key]
				}
				if tmp, err := json.Marshal(filtered); err != nil {
					return nil, err
				} else {
					if _, err = writer.WriteString(string(tmp)); err != nil {
						return nil, err
					}
					if _, err = writer.WriteRune('\n'); err != nil {
						return nil, err
					}
					if err = writer.Flush(); err != nil {
						return nil, err
					}
				}
			}
			return response, responseError
		}
	}
}
