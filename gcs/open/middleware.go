package open

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/gcs"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(bucket, name, credential string, kind gcs.FileType) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			eof := ctx.Value(sys_key.EOF)
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
				var file map[string]interface{}
				if file, ok = ctx.Value(sys_key.FILE_KEY).(map[string]interface{}); !ok {
					file = make(map[string]interface{})
				}
				switch kind {
				case gcs.READER:
					file[name] = con.Reader()
					ctx = context.WithValue(ctx, sys_key.FILE_KEY, file)
				case gcs.WRITER:
					file[name] = con.Writer()
					ctx = context.WithValue(ctx, sys_key.FILE_KEY, file)
				}
				var fileObject map[string]interface{}
				if fileObject, ok = ctx.Value(sys_key.FILE_OBJECT_KEY).(map[string]interface{}); !ok {
					fileObject = make(map[string]interface{})
				}
				fileObject[name] = con
				ctx = context.WithValue(ctx, sys_key.FILE_OBJECT_KEY, fileObject)
				return next(ctx, req)
			}
		}

	}
}
