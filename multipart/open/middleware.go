package open

import (
	"context"
	"mime/multipart"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	_multipart "github.com/louvri/gokrt/multipart"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(multipart *multipart.FileHeader) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			if _, ok := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context); !ok {
				ctx = icontext.New(ctx)
			}
			ictx, _ := ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context)
			eof := ictx.Get(sys_key.EOF)
			if eof != nil && eof == "eof" {
				return next(ctx, req)
			} else {
				con, err := _multipart.New(multipart)
				if err != nil {
					return nil, err
				}
				var ok bool
				var file map[string]any
				if file, ok = ictx.Get(sys_key.FILE_KEY).(map[string]any); !ok {
					file = make(map[string]any)
				}
				if err := con.Connect(ctx); err != nil {
					return nil, err
				}
				file[con.Name()] = con.Reader()
				ctx = context.WithValue(ctx, sys_key.FILE_KEY, file)

				var fileObject map[string]any
				if fileObject, ok = ictx.Get(sys_key.FILE_OBJECT_KEY).(map[string]any); !ok {
					fileObject = make(map[string]any)
				}
				fileObject[con.Name()] = con
				ictx.Set(sys_key.FILE_OBJECT_KEY, fileObject)
				// ctx = context.WithValue(ctx, sys_key.FILE_OBJECT_KEY, fileObject)
				return next(ctx, req)
			}
		}

	}
}
