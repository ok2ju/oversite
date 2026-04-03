package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOClient wraps the MinIO SDK to implement testutil.S3Client.
type MinIOClient struct {
	client *minio.Client
}

// NewMinIOClient creates a new MinIO client connected to the given endpoint.
// The endpoint should be a host:port string (e.g. "localhost:9000").
func NewMinIOClient(endpoint, accessKey, secretKey string, useSSL bool) (*MinIOClient, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("minio endpoint is required")
	}
	if accessKey == "" {
		return nil, fmt.Errorf("minio access key is required")
	}
	if secretKey == "" {
		return nil, fmt.Errorf("minio secret key is required")
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("creating minio client: %w", err)
	}

	return &MinIOClient{client: client}, nil
}

// EnsureBucket creates the bucket if it does not already exist.
// This is idempotent — calling it multiple times for the same bucket is safe.
func (c *MinIOClient) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := c.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("checking bucket %q: %w", bucket, err)
	}
	if exists {
		slog.Info("bucket already exists", "bucket", bucket)
		return nil
	}

	if err := c.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("creating bucket %q: %w", bucket, err)
	}
	slog.Info("bucket created", "bucket", bucket)
	return nil
}

// PutObject uploads data from reader to the specified bucket and key.
func (c *MinIOClient) PutObject(ctx context.Context, bucket, key string, reader io.Reader, size int64) error {
	_, err := c.client.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("putting object %s/%s: %w", bucket, key, err)
	}
	return nil
}

// GetObject retrieves an object from the specified bucket and key.
// The caller is responsible for closing the returned ReadCloser.
func (c *MinIOClient) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	obj, err := c.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting object %s/%s: %w", bucket, key, err)
	}

	// Stat the object to trigger an actual request and detect errors
	// (e.g. key not found). GetObject itself is lazy and doesn't fail
	// until you read or stat.
	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()
		return nil, fmt.Errorf("stating object %s/%s: %w", bucket, key, err)
	}

	return obj, nil
}

// DeleteObject removes an object from the specified bucket and key.
func (c *MinIOClient) DeleteObject(ctx context.Context, bucket, key string) error {
	err := c.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("deleting object %s/%s: %w", bucket, key, err)
	}
	return nil
}

// ObjectExists checks whether an object exists at the specified bucket and key.
func (c *MinIOClient) ObjectExists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := c.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("checking object %s/%s: %w", bucket, key, err)
	}
	return true, nil
}

// PresignedGetURL generates a presigned URL for downloading an object.
// The URL is valid for the specified expiry duration.
func (c *MinIOClient) PresignedGetURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	u, err := c.client.PresignedGetObject(ctx, bucket, key, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("generating presigned URL for %s/%s: %w", bucket, key, err)
	}
	return u.String(), nil
}
