package reader

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(filename string, size int, decoder func(data any) any, ignoreError bool, splitterSym ...string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			var ictx icontext.Context

			if tmp, ok := ctx.(icontext.Context); ok {
				ictx = tmp
			} else {
				ictx = icontext.New(ctx).(icontext.Context)
			}
			splitter := ";"
			if len(splitterSym) > 0 && splitterSym[0] != "" {
				splitter = splitterSym[0]
			}
			var reader io.Reader
			if tmp, ok := ictx.Get(sys_key.FILE_KEY).(map[string]any); tmp != nil && ok {
				reader = tmp[filename].(io.Reader)
			} else {
				return nil, nil
			}
			scanner := bufio.NewScanner(reader)
			first := true
			var columns []string
			exec := func(ctx context.Context, data map[string]any) (any, error) {
				isEmpty := len(data) == 0
				if !isEmpty {
					var err error
					var response any
					if decoder != nil {
						response, err = next(ctx, decoder(data))
					} else {
						response, err = next(ctx, data)
					}
					if err != nil {
						return response, err
					}
					return response, nil
				}
				return nil, nil
			}
			var err error
			var response any
			nextErr := make([]error, 0)
			lineNumber := 1
			for scanner.Scan() {
				text := scanner.Text()
				text = strings.ReplaceAll(text, "\ufeff", "")
				text = strings.ReplaceAll(text, "\xa0", " ")
				text = strings.ReplaceAll(text, "\"", " ")
				text = strings.TrimSpace(text)
				if first {
					columns = strings.Split(text, splitter)
					first = false
					ictx.Set(sys_key.SOF, true)
				} else {
					values := strings.Split(text, splitter)
					//check values
					isempty := true
					for _, item := range values {
						isempty = isempty && (item == "" || item == " ")
					}
					if !isempty {
						data := make(map[string]any)
						for i, column := range columns {
							data[column] = values[i]
						}
						data["lineNumber"] = lineNumber
						response, err = exec(ictx, data)
						if err != nil && !ignoreError {
							return nil, fmt.Errorf("%s:%s", "csv_reader_middleware:", err.Error())
						} else if err != nil {
							nextErr = append(nextErr, err)
						}
					}
					ictx.Set(sys_key.SOF, false)
				}
				lineNumber++
				time.Sleep(0)
			}
			if tmp, err := exec(ictx, nil); err != nil && !ignoreError {
				return nil, fmt.Errorf("%s:%s", "csv_reader_middleware:", err.Error())
			} else {
				if err != nil {
					nextErr = append(nextErr, err)
				}
				if tmp != nil {
					response = tmp
				}
			}
			if err := scanner.Err(); err != nil && !ignoreError {
				return nil, fmt.Errorf("%s:%s", "csv_reader_middleware:", err.Error())
			}
			ictx.Set(sys_key.EOF, "eof")
			if tmp, err := next(ctx, nil); err != nil {
				return nil, fmt.Errorf("%s:%s", "csv_reader_middleware:", err.Error())
			} else if tmp != nil {
				response = tmp
			}
			if len(nextErr) > 0 {
				duplicate := make(map[string]string)
				allErrors := ""
				for _, err := range nextErr {
					errString := err.Error()
					if duplicate[errString] == "" && errString != "" {
						duplicate[errString] = errString
						allErrors = allErrors + errString
					}

				}
				return response, errors.New(allErrors)
			}
			return response, nil
		}
	}
}
