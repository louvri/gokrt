package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/louvri/gokrt/connection"
	"google.golang.org/api/option"
)

type FileType rune

const READER FileType = 1
const WRITER FileType = 2

type File struct {
	client *storage.Client
	reader *storage.Reader
	writer *storage.Writer
	cancel context.CancelFunc
	name   string
	bucket string
	kind   FileType
}

func New(bucket, name string, credential string, kind FileType) (connection.Connection, error) {
	if credential == "" {
		return nil, errors.New("driver_gcs_middleware: credential is required")
	}
	var opts []option.ClientOption
	credential = strings.Replace(credential, "\n", "\\n", -1)
	_, err := os.Stat(credential)
	if err != nil {
		opts = append(opts, option.WithCredentialsJSON([]byte(credential)))
	} else {
		opts = append(opts, option.WithCredentialsFile(credential))
	}
	client, err := storage.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("driver_gcs_middleware: %s", err.Error())
	}
	return &File{
		bucket: bucket,
		name:   name,
		client: client,
	}, nil
}
func (f *File) Connect(ctx context.Context) error {
	var err error
	gcs := f.client.Bucket(f.bucket).Object(f.name)
	switch f.kind {
	case READER:
		{
			f.reader, err = gcs.NewReader(ctx)
			if err != nil {
				return fmt.Errorf("driver_gcs_middleware_connect: %s", err.Error())
			}
		}
	case WRITER:
		{
			ctx, cancel := context.WithCancel(ctx)
			f.cancel = cancel
			f.writer = gcs.NewWriter(ctx)
			f.writer.ContentType = "text/csv"
			f.writer.ChunkSize = 1024
		}
	}
	return nil
}
func (f *File) Cancel() {
	f.cancel()
}

func (f *File) Close() {
	defer f.client.Close()
	switch f.kind {
	case READER:
		{
			f.reader.Close()
		}
	case WRITER:
		{
			f.writer.Close()
		}
	}
}
func (f *File) Handler() interface{} {
	return f.client
}
func (f *File) Reader() io.Reader {
	return f.reader
}

func (f *File) Writer() io.Writer {
	return f.writer
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Driver() string {
	return "gcs"
}
