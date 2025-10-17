package open

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/gcs"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(bucket, name, credential string, kind gcs.FileType) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			var ok bool
			var ictx *icontext.Context
			if ictx, ok = ctx.Value(sys_key.GOKRT_CONTEXT).(*icontext.Context); !ok {
				ictx = icontext.New(ctx).(*icontext.Context)
			}
			eof := ictx.Get(sys_key.EOF)
			if eof != nil && eof == "eof" {
				return next(ctx, req)
			} else {
				con, err := gcs.New(bucket, name, credential, kind)
				if err != nil {
					return nil, err
				}
				err = con.Connect(context.Background())
				if err != nil {
					return nil, err
				}
				var ok bool
				var file map[string]any
				if file, ok = ictx.Get(sys_key.FILE_KEY).(map[string]any); !ok {
					file = make(map[string]any)
				}
				switch kind {
				case gcs.READER:
					file[name] = con.Reader()
					// ctx = context.WithValue(ctx, sys_key.FILE_KEY, file)
					ictx.Set(sys_key.FILE_KEY, file)
				case gcs.WRITER:
					file[name] = con.Writer()
					// ctx = context.WithValue(ctx, sys_key.FILE_KEY, file)
					ictx.Set(sys_key.FILE_KEY, file)
				}
				var fileObject map[string]any
				if fileObject, ok = ictx.Get(sys_key.FILE_OBJECT_KEY).(map[string]any); !ok {
					fileObject = make(map[string]any)
				}
				fileObject[name] = con
				// ctx = context.WithValue(ctx, sys_key.FILE_OBJECT_KEY, fileObject)
				ictx.Set(sys_key.FILE_OBJECT_KEY, fileObject)
				return next(ictx, req)
			}
		}

	}
}
