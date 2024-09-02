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

const DEFAULT_MIME_TYPE = "application/octet-stream"
const CHECKSUM_META_KEY = "checksum"

type s3ClientIface interface {
	s3.ListObjectsV2APIClient
	s3.HeadObjectAPIClient
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
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
func (c *S3Client) GetFiles(bucket string, prefix string, suffix string) ([]string, bool) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}
	if strings.TrimSpace(prefix) != "" {
		input.Prefix = aws.String(prefix)
	}
	result, err := c.client.ListObjectsV2(context.TODO(), input)
	var contents []types.Object
	if err != nil {
		logger.Error(fmt.Sprintf("[S3] ERROR: Can not get files under %s in bucket %s due to error: %s ", prefix,
			bucket, err))
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

func (c *S3Client) getObject(bucket, key string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	output, err := c.client.GetObject(context.TODO(), input)
	if err != nil {
		logger.Error(fmt.Sprintf("[S3] ERROR: Can not read file %s in bucket %s due to error: %s ", key,
			bucket, err))
		return nil, err
	}
	defer output.Body.Close()

	return io.ReadAll(output.Body)
}

func (c *S3Client) ReadFileContent(bucket, key string) (string, error) {
	contentBytes, err := c.getObject(bucket, key)
	if err != nil {
		logger.Error(fmt.Sprintf("[S3] ERROR: Can not read file %s in bucket %s due to error: %s ", key,
			bucket, err))
		return "", err
	}
	return string(contentBytes[:]), nil
}

func (c *S3Client) DownloadFile(bucket, key, filePath string) error {
	contentBytes, err := c.getObject(bucket, key)
	if err != nil {
		logger.Error(fmt.Sprintf("[S3] ERROR: Can not download file %s in bucket %s due to error: %s ", key,
			bucket, err))
		return err
	}
	realFilePath := path.Join(filePath, key)
	util.StoreFile(realFilePath, string(contentBytes), true)
	return nil
}

// List the content in folder in an s3 bucket. Note it's not recursive,
// which means the content only contains the items in that folder, but
// not in its subfolders.
func (c *S3Client) ListFolderContent(bucket, folder string) []string {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
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
				bucket, err.Error()))
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

func (c *S3Client) FileExistsInBucket(bucket, path string) (bool, error) {
	_, err := c.client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
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

// Deletes file in s3 bucket, regardless of any extra
// information like product and version info.
//
// * Warning: this will directly delete the files even if
// it has lots of product info, so please be careful to use.
// If you want to delete product artifact files, please use
// delete_files
func (c *S3Client) SimpleDeleteFile(filePath string, target util.Target) bool {
	bucket := target.Bucket
	prefix := target.Prefix
	pathKey := path.Join(prefix, filePath)
	// try:
	existed, _ := c.FileExistsInBucket(bucket, pathKey)
	if existed {
		_, err := c.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(pathKey),
		})
		if err != nil {
			logger.Error(fmt.Sprintf("Error: Can not delete file due to error: %s", err.Error()))
			return false
		}
		return true
	} else {
		logger.Warn(
			fmt.Sprintf("Warning: File %s does not exist in S3 bucket %s, will ignore its deleting",
				filePath, bucket))
		return true
	}
}

// Uploads file to s3 bucket, regardless of any extra
// information like product and version info.
//
// * Warning: If force is set True, it will directly overwrite
// the files even if it has lots of product info, so please be
// careful to use. If you want to upload product artifact files,
// please use upload_files()
func (c *S3Client) SimpleUploadFile(filePath, fileContent string,
	target [2]string, mimeType string, checksumSHA1 string, force bool) error {
	bucket := target[0]
	prefix := target[1]
	pathKey := path.Join(prefix, filePath)
	logger.Debug(fmt.Sprintf("Uploading %s to bucket %s", pathKey, bucket))
	existed, err := c.FileExistsInBucket(bucket, pathKey)
	if err != nil {
		logger.Error(
			fmt.Sprintf("Error: file existence check failed due to error: %s", err))
		return err
	}

	contentType := mimeType
	if strings.TrimSpace(contentType) == "" {
		contentType = DEFAULT_MIME_TYPE
	}
	if !existed || force {
		fMeta := map[string]string{}
		if strings.TrimSpace(checksumSHA1) != "" {
			fMeta[CHECKSUM_META_KEY] = checksumSHA1
		}
		if !c.dry_run {
			_, err := c.client.PutObject(context.TODO(), &s3.PutObjectInput{
				Bucket:      aws.String(bucket),
				Key:         aws.String(pathKey),
				Body:        strings.NewReader(fileContent),
				ContentType: aws.String(contentType),
				Metadata:    fMeta,
			})
			if err != nil {
				logger.Error(fmt.Sprintf(
					"ERROR: file %s not uploaded to bucket %s due to error: %s ",
					filePath, bucket, err))
				return err
			}
			logger.Debug(fmt.Sprintf("Uploaded %s to bucket %s", pathKey, bucket))
		}
	} else {
		return fmt.Errorf("error: file %s already exists, upload is forbiden", pathKey)
	}
	return nil
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
func (c *S3Client) UploadFiles(file_paths []string, targets []util.Target,
	product string, root string) {
	realRoot := root
	if strings.TrimSpace(realRoot) == "" {
		realRoot = "/"
	}

	// mainTarget := targets[0]
	// mainBucket := mainTarget.Bucket
	// keyPrefix := mainTarget.Prefix
	// var extraTargets []util.Target
	// if len(targets) > 1 {
	// 	for i, t := range targets {
	// 		if i >= 1 {
	// 			extraTargets[i] = t
	// 		}
	// 	}
	// }
	// var extraPrefixedBuckets [][]string
}
