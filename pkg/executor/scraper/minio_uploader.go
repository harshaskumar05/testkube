package scraper

import (
	"context"

	coreminio "github.com/minio/minio-go/v7"

	"github.com/kubeshop/testkube/pkg/log"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/pkg/storage/minio"
)

type MinIOUploader struct {
	Endpoint, AccessKeyID, SecretAccessKey, Region, Token, Bucket string
	Ssl                                                           bool
	client                                                        *minio.Client
}

func NewMinIOUploader(endpoint, accessKeyID, secretAccessKey, region, token, bucket string, ssl bool) (*MinIOUploader, error) {
	l := &MinIOUploader{
		Endpoint:        endpoint,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		Region:          region,
		Token:           token,
		Bucket:          bucket,
		Ssl:             ssl,
	}

	client := minio.NewClient(l.Endpoint, l.AccessKeyID, l.SecretAccessKey, l.Region, l.Token, l.Bucket, l.Ssl)
	err := client.Connect()
	if err != nil {
		return nil, errors.Errorf("error occured creating minio client: %v", err)
	}
	l.client = client

	return l, nil
}

func (l *MinIOUploader) Upload(ctx context.Context, object *Object, execution testkube.Execution) error {
	folder := execution.Id

	log.DefaultLogger.Infow("MinIO loader is uploading file", "file", object.Name, "folder", folder, "size", object.Size)
	opts := coreminio.PutObjectOptions{}
	switch object.DataType {
	case DataTypeRaw:
		opts.ContentType = "application/octet-stream"
	case DataTypeTarball:
		opts.DisableMultipart = true
		opts.ContentEncoding = "gzip"
		opts.ContentType = "application/gzip"
		opts.UserMetadata = map[string]string{
			"X-Amz-Meta-Snowball-Auto-Extract": "true",
			"X-Amz-Meta-Minio-Snowball-Prefix": execution.Id,
		}
	}

	if err := l.client.SaveFileDirect(ctx, folder, object.Name, object.Data, object.Size, opts); err != nil {
		return errors.Wrapf(err, "error saving file %s", object.Name)
	}

	return nil
}

func (l *MinIOUploader) Close() error {
	return nil
}

var _ Uploader = (*MinIOUploader)(nil)
