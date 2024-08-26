package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
)

const TEST_BUCKET = "test_bucket"

type MockAWSS3Client struct {
	lsObjV2 func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	getObj  func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

func (m MockAWSS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return m.lsObjV2(ctx, params, optFns...)
}
func (m MockAWSS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m.getObj(ctx, params, optFns...)
}
func S3ClientWithMock(mockAWSS3Client MockAWSS3Client) (*S3Client, error) {
	s3client, err := NewS3Client("", 10, false)
	if err != nil {
		return nil, err
	}
	s3client.client = mockAWSS3Client
	return s3client, nil
}
func TestGetFiles(t *testing.T) {
	all_files := []string{
		"io/quarkus/quakus-bom/quarkus.bom",
		"org/apache/activemq/activemq.jar",
		"org/apache/activemq/activemq.pom",
	}
	s3client, err := S3ClientWithMock(MockAWSS3Client{
		lsObjV2: func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			if params.Bucket == nil || strings.TrimSpace(*params.Bucket) != TEST_BUCKET {
				return nil, fmt.Errorf("expect bucket to not be %s", TEST_BUCKET)
			}

			contents := []types.Object{
				{Key: aws.String(all_files[0])}, {Key: aws.String(all_files[1])}, {Key: aws.String(all_files[2])},
			}
			if params.Prefix != nil && *params.Prefix == "io/quarkus" {
				contents = []types.Object{{Key: aws.String(all_files[0])}}
			}

			return &s3.ListObjectsV2Output{
				Contents: contents,
			}, nil
		},
	})
	assert.Nil(t, err)

	_, ok := s3client.GetFiles("", "", "")
	assert.False(t, ok)

	files, ok := s3client.GetFiles(TEST_BUCKET, "", "")
	assert.True(t, ok)
	assert.Equal(t, 3, len(files))

	files, ok = s3client.GetFiles(TEST_BUCKET, "io/quarkus", "")
	assert.True(t, ok)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, all_files[0], files[0])

	files, ok = s3client.GetFiles(TEST_BUCKET, "", "jar")
	assert.True(t, ok)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, all_files[1], files[0])

	files, ok = s3client.GetFiles(TEST_BUCKET, "org/apache", "pom")
	assert.True(t, ok)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, all_files[2], files[0])
}

func TestReadFileContent(t *testing.T) {
	testKey := "io/quarkus/quakus-bom/maven-metadata.xml"
	testContet := `<?xml version="1.0" encoding="UTF-8"?>
<metadata>
  <groupId>io.quarkus</groupId>
  <artifactId>quarkus-bom</artifactId>
  <versioning>
    <latest>0.12.0</latest>
    <release>0.12.0</release>
    <versions>
      <version>0.11.0</version>
      <version>0.12.0</version>
    </versions>
  </versioning>
</metadata>
	`
	s3client, err := S3ClientWithMock(MockAWSS3Client{
		getObj: func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			if params.Bucket == nil || strings.TrimSpace(*params.Bucket) != TEST_BUCKET {
				return nil, fmt.Errorf("expect bucket to not be %s", TEST_BUCKET)
			}
			if params.Key == nil || strings.TrimSpace(*params.Key) != testKey {
				return nil, fmt.Errorf("404 Not Found: expect key to be %s", testKey)
			}
			return &s3.GetObjectOutput{
				Body: io.NopCloser(strings.NewReader(testContet)),
			}, nil
		},
	})
	assert.Nil(t, err)

	_, err = s3client.ReadFileContent("", testKey)
	assert.Contains(t, err.Error(), TEST_BUCKET)

	_, err = s3client.ReadFileContent(TEST_BUCKET, "not_exist_key")
	assert.Contains(t, err.Error(), "404")

	content, err := s3client.ReadFileContent(TEST_BUCKET, testKey)
	assert.Nil(t, err)
	assert.Equal(t, testContet, content)

}
