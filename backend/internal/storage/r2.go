package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const immutableCacheControl = "public,max-age=31536000,immutable"

// R2 Cloudflare R2 存储实现，走 S3 兼容 API。
type R2 struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
	baseURL string
}

func NewR2(accountID, accessKey, secretKey, bucket, publicBaseURL string) (*R2, error) {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	client := s3.New(s3.Options{
		Region:       "auto",
		Credentials:  credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		BaseEndpoint: aws.String(endpoint),
		UsePathStyle: true,
	})
	return &R2{
		client:  client,
		presign: s3.NewPresignClient(client),
		bucket:  bucket,
		baseURL: strings.TrimRight(publicBaseURL, "/"),
	}, nil
}

func (r *R2) Driver() string { return "r2" }

func (r *R2) Put(ctx context.Context, obj Object) error {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(r.bucket),
		Key:          aws.String(obj.Key),
		Body:         bytes.NewReader(obj.Body),
		ContentType:  aws.String(obj.ContentType),
		CacheControl: aws.String(immutableCacheControl),
	})
	return err
}

func (r *R2) Get(ctx context.Context, key string) (Object, error) {
	out, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return Object{}, err
	}
	defer out.Body.Close()
	body, err := io.ReadAll(out.Body)
	if err != nil {
		return Object{}, err
	}
	ct := ""
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	return Object{Key: key, ContentType: ct, Body: body}, nil
}

func (r *R2) Delete(ctx context.Context, keys []string) error {
	for _, k := range keys {
		if _, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(r.bucket),
			Key:    aws.String(k),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *R2) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error) {
	req, err := r.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(r.bucket),
		Key:          aws.String(key),
		ContentType:  aws.String(contentType),
		CacheControl: aws.String(immutableCacheControl),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}

func (r *R2) PublicURL(key string) string {
	return r.baseURL + "/" + key
}
