package storage

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

type s3ClientIface interface {
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

type S3Client struct {
	aws_profile string
	con_limit   int
	dry_run     bool
	buckets     map[string]types.Bucket
	client      s3ClientIface
}

func NewS3Client(aws_profile string, con_limit int, dry_run bool) (*S3Client, error) {
	s3Client := &S3Client{
		aws_profile: aws_profile,
		con_limit:   con_limit,
		dry_run:     dry_run,
	}

	var cfg aws.Config
	var err error
	if strings.TrimSpace(s3Client.aws_profile) != "" {
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(aws_profile))
	} else {
		cfg, err = config.LoadDefaultConfig(context.TODO())
	}

	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	// Create an Amazon S3 service client
	s3Client.client = s3.NewFromConfig(cfg)
	return s3Client, nil
}

// Get the file names from s3 bucket. Can use prefix and suffix to filter the
// files wanted. If some error happend, will return an empty file list and false result
func (c *S3Client) GetFiles(bucketName string,
	prefix string, suffix string) ([]string, bool) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	}
	if strings.TrimSpace(prefix) != "" {
		input.Prefix = &prefix
	}
	result, err := c.client.ListObjectsV2(context.TODO(), input)
	var contents []types.Object
	if err != nil {
		logger.Error(fmt.Sprintf("[S3] ERROR: Can not get files under %s in bucket %s due to error: %s ", prefix,
			bucketName, err))
		return []string{}, false
	} else {
		contents = result.Contents
	}
	var files []string

	if strings.TrimSpace(suffix) != "" {
		for _, v := range contents {
			fileName := *v.Key
			if strings.HasSuffix(fileName, suffix) {
				files = append(files, fileName)
			}
		}
	} else {
		for _, v := range contents {
			files = append(files, *v.Key)
		}
	}
	return files, true
}

// Upload a list of files to s3 bucket.
//
// * Use the cut down file path as s3 key. The cut down way is move root from the file path if it starts with root. Example: if file_path is
// /tmp/maven-repo/org/apache/.... and root is /tmp/maven-repo Then the key will be
// org/apache/.....
//
// * The product will be added as the extra metadata with key "rh-products". For
// example, if the product for a file is "apache-commons", the metadata of that file
// will contain "rh-products":"apache-commons"
//
// * For existed files, the upload will not override them, as the metadata of
// "rh-products" will be updated to add the new product. For example, if an exited file
// with new product "commons-lang3" is uploaded based on existed metadata
// "apache-commons", the file will not be overridden, but the metadata will be changed to
// "rh-products": "apache-commons,commons-lang3"
//
// * Every file has sha1 checksum in "checksum" metadata. When uploading existed files,
// if the checksum does not match the existed one, will not upload it and report error.
// Note that if file name match
//
// * Return all failed to upload files due to any exceptions.
func (c *S3Client) UploadFiles(file_paths []string, targets [][]string,
	product string, root string) {
	realRoot := root
	if strings.TrimSpace(realRoot) == "" {
		realRoot = "/"
	}
	//TODO: not implemented yet
}

func (c *S3Client) getBucket(bucket_name string) types.Bucket {
	// self.__lock.acquire()
	// defer self.__lock.release()
	bucket, existed := c.buckets[bucket_name]
	if existed {
		return bucket
	}

	logger.Debug(fmt.Sprintf("[S3] Cache aws bucket %s", bucket_name))
	bucket = types.Bucket{Name: &bucket_name}
	c.buckets[bucket_name] = bucket
	return bucket
}
