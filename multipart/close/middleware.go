package close

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/connection"
	icontext "github.com/louvri/gokrt/context"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			var ictx icontext.IContext

			if tmp, ok := ctx.(icontext.IContext); ok {
				ictx = tmp
			} else {
				ictx = icontext.New(ctx)
			}
			eof := ictx.Get(sys_key.EOF)
			if eof != nil && eof != "" {
				if fileObjects, ok := ictx.Get(sys_key.FILE_OBJECT_KEY).(map[string]any); ok {
					for _, fileObject := range fileObjects {
						if con, ok := fileObject.(connection.Connection); ok && con.Driver() == "multipart" {
							con.Close()
						}
					}
				}
			}
			return next(ictx, req)
		}
	}
}
