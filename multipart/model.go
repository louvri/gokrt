package multipart

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/louvri/gokrt/connection"
)

type File struct {
	handler *multipart.FileHeader
	reader  multipart.File
	name    string
}

func New(multipart *multipart.FileHeader) (connection.Connection, error) {
	return &File{
		handler: multipart,
		name:    multipart.Filename,
	}, nil
}

func (f *File) Connect(ctx context.Context) error {
	var err error
	f.reader, err = f.handler.Open()
	if err != nil {
		return fmt.Errorf("driver_multipart_middleware_connect: %s", err.Error())
	}
	return nil
}
func (f *File) Cancel() {
	//not implemented
}

func (f *File) Close() {
	f.reader.Close()
}
func (f *File) Handler() any {
	return f.handler
}
func (f *File) Reader() io.Reader {
	return f.reader
}

func (f *File) Writer() io.Writer {
	return nil
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Driver() string {
	return "multipart"
}
