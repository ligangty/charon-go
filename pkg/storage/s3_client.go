package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"org.commonjava/charon/pkg/util"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

type s3ClientIface interface {
	s3.ListObjectsV2APIClient
	s3.HeadObjectAPIClient
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type S3Client struct {
	aws_profile string
	con_limit   int
	dry_run     bool
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
func (c *S3Client) GetFiles(bucketName string, prefix string, suffix string) ([]string, bool) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	}
	if strings.TrimSpace(prefix) != "" {
		input.Prefix = aws.String(prefix)
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

func (c *S3Client) getObject(bucketName, key string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	}
	output, err := c.client.GetObject(context.TODO(), input)
	if err != nil {
		logger.Error(fmt.Sprintf("[S3] ERROR: Can not read file %s in bucket %s due to error: %s ", key,
			bucketName, err))
		return nil, err
	}
	defer output.Body.Close()

	return io.ReadAll(output.Body)
}

func (c *S3Client) ReadFileContent(bucketName, key string) (string, error) {
	contentBytes, err := c.getObject(bucketName, key)
	if err != nil {
		logger.Error(fmt.Sprintf("[S3] ERROR: Can not read file %s in bucket %s due to error: %s ", key,
			bucketName, err))
		return "", err
	}
	return string(contentBytes[:]), nil
}

func (c *S3Client) DownloadFile(bucketName, key, filePath string) error {
	contentBytes, err := c.getObject(bucketName, key)
	if err != nil {
		logger.Error(fmt.Sprintf("[S3] ERROR: Can not download file %s in bucket %s due to error: %s ", key,
			bucketName, err))
		return err
	}
	realFilePath := path.Join(filePath, key)
	util.StoreFile(realFilePath, string(contentBytes), true)
	return nil
}

// List the content in folder in an s3 bucket. Note it's not recursive,
// which means the content only contains the items in that folder, but
// not in its subfolders.
func (c *S3Client) ListFolderContent(bucketName, folder string) []string {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	}
	if strings.HasSuffix(folder, "/") {
		input.Prefix = aws.String(folder)
	} else {
		input.Prefix = aws.String(folder + "/")
	}
	input.Delimiter = aws.String("/")
	paginator := s3.NewListObjectsV2Paginator(c.client, input)

	contents := []string{}
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			logger.Error(fmt.Sprintf("[S3] ERROR: Can not get contents of %s from bucket %s due to error: %s", folder,
				bucketName, err.Error()))
			return []string{}
		}

		folders := page.CommonPrefixes
		if len(folders) > 0 {
			for _, f := range folders {
				contents = append(contents, *f.Prefix)
			}
		}
		files := page.Contents
		if len(files) > 0 {
			for _, f := range files {
				contents = append(contents, *f.Key)
			}
		}
	}
	return contents
}

func (c *S3Client) FileExistsInBucket(bucketName, path string) (bool, error) {
	_, err := c.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		var ae *types.NotFound
		if errors.As(err, &ae) {
			return false, nil
		}
		return false, err
	}
	return true, nil
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
