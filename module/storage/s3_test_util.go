package storage

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const TEST_BUCKET = "test_bucket"

type MockAWSS3Client struct {
	LsObjV2 func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	HeadObj func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	GetObj  func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObj  func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DelObj  func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	CpObj   func(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
}

func (m MockAWSS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return m.LsObjV2(ctx, params, optFns...)
}
func (m MockAWSS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return m.HeadObj(ctx, params, optFns...)
}
func (m MockAWSS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m.GetObj(ctx, params, optFns...)
}
func (m MockAWSS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return m.PutObj(ctx, params, optFns...)
}
func (m MockAWSS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return m.DelObj(ctx, params, optFns...)
}
func (m MockAWSS3Client) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	return m.CpObj(ctx, params, optFns...)
}
func S3ClientWithMock(mockAWSS3Client MockAWSS3Client) (*S3Client, error) {
	s3client, err := NewS3Client("", 10, false)
	if err != nil {
		return nil, err
	}
	s3client.client = mockAWSS3Client
	return s3client, nil
}
