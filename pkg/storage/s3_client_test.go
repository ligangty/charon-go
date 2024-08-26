package storage

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
)

const TEST_BUCKET = "test_bucket"

type mockListObjectV2 func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)

func (m mockListObjectV2) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return m(ctx, params, optFns...)
}
func TestGetFiles(t *testing.T) {
	all_files := []string{
		"io/quarkus/quakus-bom/quarkus.bom",
		"org/apache/activemq/activemq.jar",
		"org/apache/activemq/activemq.pom",
	}
	s3client, err := NewS3Client("", 10, false)
	assert.Nil(t, err)
	s3client.client = mockListObjectV2(func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
		if params.Bucket == nil || strings.TrimSpace(*params.Bucket) != TEST_BUCKET {
			return nil, fmt.Errorf("expect bucket to not be %s", TEST_BUCKET)
		}

		contents := []types.Object{
			{Key: &all_files[0]}, {Key: &all_files[1]}, {Key: &all_files[2]},
		}
		if params.Prefix != nil && *params.Prefix == "io/quarkus" {
			contents = []types.Object{{Key: &all_files[0]}}
		}

		return &s3.ListObjectsV2Output{
			Contents: contents,
		}, nil
	})
	_, ok := s3client.GetFiles("", "", "")
	assert.False(t, ok)
	files, ok := s3client.GetFiles("test_bucket", "", "")
	assert.True(t, ok)
	assert.Equal(t, 3, len(files))

	files, ok = s3client.GetFiles("test_bucket", "io/quarkus", "")
	assert.True(t, ok)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, all_files[0], files[0])

	files, ok = s3client.GetFiles("test_bucket", "", "jar")
	assert.True(t, ok)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, all_files[1], files[0])

	files, ok = s3client.GetFiles("test_bucket", "org/apache", "pom")
	assert.True(t, ok)
	assert.Equal(t, 1, len(files))
	assert.Equal(t, all_files[2], files[0])
}
