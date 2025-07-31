package close

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/connection"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req any) (any, error) {
			eof := ctx.Value(sys_key.EOF)
			if eof != nil && eof == "eof" {
				if fileObjects, ok := ctx.Value(sys_key.FILE_OBJECT_KEY).(map[string]any); ok {
					for _, fileObject := range fileObjects {
						if con, ok := fileObject.(connection.Connection); ok && con.Driver() == "gcs" {
							con.Close()
						}
					}
				}
			}
			return next(ctx, req)
		}
	}
}
