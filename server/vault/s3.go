package vault

import (
	"bytes"
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client provides put/get for the MulaMail encrypted-mail vault.
type S3Client struct {
	client *s3.Client
	bucket string
}

func NewS3Client(region, bucket string) (*S3Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(region))
	if err != nil {
		return nil, err
	}
	return &S3Client{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
	}, nil
}

// Put stores raw bytes at the given key.
func (v *S3Client) Put(ctx context.Context, key string, data []byte) error {
	_, err := v.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(v.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	return err
}

// Get retrieves the object at the given key.
func (v *S3Client) Get(ctx context.Context, key string) ([]byte, error) {
	out, err := v.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(v.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

// Delete removes the object at the given key.
func (v *S3Client) Delete(ctx context.Context, key string) error {
	_, err := v.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(v.bucket),
		Key:    aws.String(key),
	})
	return err
}

// List returns all keys with the given prefix.
func (v *S3Client) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	paginator := s3.NewListObjectsV2Paginator(v.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(v.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, obj := range page.Contents {
			if obj.Key != nil {
				keys = append(keys, *obj.Key)
			}
		}
	}

	return keys, nil
}
