package signed

import (
	"context"
	"time"

	"cloud.google.com/go/storage"
	"github.com/go-kit/kit/endpoint"
	"github.com/louvri/gokrt/connection"
	"github.com/louvri/gokrt/sys_key"
)

func Middleware(bucket string, expiry time.Duration) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			eof := ctx.Value(sys_key.EOF)
			if eof != nil && eof != "" {
				if fileObjects, ok := ctx.Value(sys_key.FILE_OBJECT_KEY).(map[string]interface{}); ok {
					opts := &storage.SignedURLOptions{
						Scheme:  storage.SigningSchemeV4,
						Method:  "GET",
						Expires: time.Now().Add(expiry),
					}
					uris := make([]string, 0)
					for key, fileObject := range fileObjects {
						if con, ok := fileObject.(connection.Connection); ok && con.Driver() == "gcs" {
							uri, ierr := con.Handler().(*storage.Client).Bucket(bucket).SignedURL(key, opts)
							if ierr == nil {
								uris = append(uris, uri)
							}
						}
					}
					_, err := next(ctx, req)
					return uris, err
				}
			}
			return next(ctx, req)
		}
	}
}
