package open

import (
	"context"
	"mime/multipart"

	"github.com/go-kit/kit/endpoint"
	_multipart "github.com/louvri/gokrt/multipart"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(multipart *multipart.FileHeader) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			eof := ctx.Value(sys_key.EOF)
			if eof != nil && eof == "eof" {
				return next(ctx, req)
			} else {
				con, err := _multipart.New(multipart)
				if err != nil {
					return nil, err
				}
				var ok bool
				var file map[string]interface{}
				if file, ok = ctx.Value(sys_key.FILE_KEY).(map[string]interface{}); !ok {
					file = make(map[string]interface{})
				}
				if err := con.Connect(ctx); err != nil {
					return nil, err
				}
				file[con.Name()] = con.Reader()
				ctx = context.WithValue(ctx, sys_key.FILE_KEY, file)

				var fileObject map[string]interface{}
				if fileObject, ok = ctx.Value(sys_key.FILE_OBJECT_KEY).(map[string]interface{}); !ok {
					fileObject = make(map[string]interface{})
				}
				fileObject[con.Name()] = con
				ctx = context.WithValue(ctx, sys_key.FILE_OBJECT_KEY, fileObject)
				return next(ctx, req)
			}
		}

	}
}
