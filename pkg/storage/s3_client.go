package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"slices"
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
	CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
}

type S3Client struct {
	awsProfile string
	conLimit   int
	dryRun     bool
	client     s3ClientIface
}

func NewS3Client(awsProfile string, conLimit int, dryRun bool) (*S3Client, error) {
	s3Client := &S3Client{
		awsProfile: awsProfile,
		conLimit:   conLimit,
		dryRun:     dryRun,
	}

	var cfg aws.Config
	var err error
	if strings.TrimSpace(s3Client.awsProfile) != "" {
		cfg, err = config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(awsProfile))
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

func (c *S3Client) getObject(bucket, key string) ([]byte, map[string]string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	output, err := c.client.GetObject(context.TODO(), input)
	if err != nil {
		logger.Error(fmt.Sprintf("[S3] ERROR: Can not read file %s in bucket %s due to error: %s ", key,
			bucket, err))
		return nil, nil, err
	}
	defer output.Body.Close()
	content, err := io.ReadAll(output.Body)
	return content, output.Metadata, err
}

func (c *S3Client) ReadFileContent(bucket, key string) (string, error) {
	contentBytes, _, err := c.getObject(bucket, key)
	if err != nil {
		logger.Error(fmt.Sprintf("[S3] ERROR: Can not read file %s in bucket %s due to error: %s ", key,
			bucket, err))
		return "", err
	}
	return string(contentBytes[:]), nil
}

func (c *S3Client) DownloadFile(bucket, key, filePath string) error {
	contentBytes, _, err := c.getObject(bucket, key)
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
		if !c.dryRun {
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
func (c *S3Client) UploadFiles(filePaths []string, targets []util.Target,
	product string, root string) []string {
	realRoot := root
	if strings.TrimSpace(realRoot) == "" {
		realRoot = "/"
	}

	mainTarget := targets[0]
	mainBucket := mainTarget.Bucket
	keyPrefix := mainTarget.Prefix
	var extraPrefixedBuckets []util.Target
	if len(targets) > 1 {
		extraPrefixedBuckets = make([]util.Target, len(targets))
		for i, t := range targets {
			if i >= 1 {
				extraPrefixedBuckets[i] = t
			}
		}
	}
	return doPathCutAnd(product, mainBucket, keyPrefix, filePaths, extraPrefixedBuckets, c.pathUploadHandler, root)
}

func (c *S3Client) pathUploadHandler(product, mainBucket, keyPrefix, fullFilePath, fPath string, index,
	total int, extraPrefixedBuckets []util.Target) bool {
	if !util.IsFile(fullFilePath) {
		logger.Warn(fmt.Sprintf("[S3] Warning: file %s does not exist during uploading. Product: %s",
			fullFilePath, product))
		return false
	}
	logger.Debug(fmt.Sprintf("[S3] (%d/%d) Uploading %s to bucket %s",
		index, total, fullFilePath, mainBucket))
	mainPathKey := fPath
	if strings.TrimSpace(keyPrefix) != "" {
		mainPathKey = path.Join(keyPrefix, fPath)
	}
	existed, err := c.FileExistsInBucket(mainBucket, mainPathKey)
	if err != nil {
		logger.Error(fmt.Sprintf("[S3] Error: file existence check failed due to error: %s", err))
		return false
	}
	sha1 := util.ReadSHA1(fullFilePath)
	contentType := util.GuessMimetype(fullFilePath)
	if contentType == "" {
		contentType = DEFAULT_MIME_TYPE
	}
	if !existed {
		fMeta := map[string]string{}
		if sha1 != "" {
			fMeta[CHECKSUM_META_KEY] = sha1
		}
		if !c.dryRun {
			var err error
			if len(fMeta) > 0 {
				_, err = c.client.PutObject(context.TODO(), &s3.PutObjectInput{
					Bucket:      aws.String(mainBucket),
					Key:         aws.String(mainPathKey),
					Body:        strings.NewReader(fullFilePath),
					ContentType: aws.String(contentType),
					Metadata:    fMeta,
				})
			} else {
				_, err = c.client.PutObject(context.TODO(), &s3.PutObjectInput{
					Bucket:      aws.String(mainBucket),
					Key:         aws.String(mainPathKey),
					Body:        strings.NewReader(fullFilePath),
					ContentType: aws.String(contentType),
				})
			}
			if err != nil {
				logger.Error(fmt.Sprintf("[S3] ERROR: file %s not uploaded to bucket %s due to error: %s ", fullFilePath,
					mainBucket, err))
				return false
			}
			if strings.TrimSpace(product) != "" {
				c.updateProductInfo(mainPathKey, mainBucket, []string{product})
			}
		}
		logger.Debug(fmt.Sprintf("[S3] Uploaded %s to bucket %s", fPath, mainBucket))
	} else {
		c.handleExisted(fullFilePath, sha1, mainPathKey, mainBucket, product)
	}

	for _, target := range extraPrefixedBuckets {
		extraBucket := target.Bucket
		extraPrefix := target.Prefix
		extraPathKey := fPath
		if strings.TrimSpace(extraPrefix) != "" {
			extraPathKey = path.Join(extraPrefix, fPath)
		}
		logger.Debug(fmt.Sprintf("Copyinging %s from bucket %s to bucket %s",
			fullFilePath, mainBucket, extraBucket))
		existed, _ := c.FileExistsInBucket(extraBucket, extraPathKey)
		if !existed {
			if !c.dryRun {
				ok := c.copyBetweenBucket(mainBucket, mainPathKey, extraBucket, extraPathKey)
				if !ok {
					logger.Error(fmt.Sprintf("[S3] ERROR: copying failure happend for file %s to bucket %s due to error: %s ",
						fullFilePath, extraBucket, err))
					return false
				}
				if strings.TrimSpace(product) != "" {
					c.updateProductInfo(extraPathKey, extraBucket, []string{product})
				}
			}
		} else {
			c.handleExisted(fullFilePath, sha1, extraPathKey, extraBucket, product)
		}
	}
	return true
}

func (c *S3Client) handleExisted(filePath, fileSHA1, pathKey, bucketName, product string) bool {
	logger.Debug(fmt.Sprintf("File %s already exists in bucket %s, check if need to update product.",
		pathKey, bucketName))
	_, fMeta, err := c.getObject(bucketName, pathKey)
	if err != nil {
		logger.Error(fmt.Sprintf("[S3] Can not get object for %s due to error: %s", pathKey, err))
		return false
	}
	checksum := ""
	if value, ok := fMeta[CHECKSUM_META_KEY]; ok {
		checksum = value
	}
	if checksum != "" && strings.TrimSpace(checksum) != fileSHA1 {
		logger.Warn(fmt.Sprintf("Warning: checksum check failed. The file %s is different from the one in S3 bucket %s. Product: %s",
			pathKey, bucketName, product))
		return false
	}

	prods, ok := c.getProductInfo(pathKey, bucketName)
	if !c.dryRun && ok && !slices.Contains(prods, product) {
		logger.Debug(
			fmt.Sprintf("File %s has new product, updating the product %s",
				filePath,
				product,
			))
		prods = append(prods, product)
		ok := c.updateProductInfo(pathKey, bucketName, prods)
		return ok
	}
	return true
}

func (c *S3Client) getProductInfo(file, bucketName string) ([]string, bool) {
	logger.Debug(fmt.Sprintf("[S3] Getting product infomation for file %s", file))
	prodInfoFile := file + util.PROD_INFO_SUFFIX
	infoFileContent, err := c.ReadFileContent(bucketName, prodInfoFile)
	if err != nil {
		logger.Warn(fmt.Sprintf("[S3] WARN: Can not get product info for file %s due to error: %s", file, err))
		return []string{}, false
	}
	var prods []string
	for _, p := range strings.Split(infoFileContent, ",") {
		prods = append(prods, strings.TrimSpace(p))
	}
	logger.Debug(fmt.Sprintf("[S3] Got product information as below %s", prods))
	return prods, true
}

func (c *S3Client) updateProductInfo(file, bucketName string, prods []string) bool {
	//TODO: not implemented
	return false
}

func (c *S3Client) copyBetweenBucket(source, sourceKey, target, targetKey string) bool {
	logger.Debug(fmt.Sprintf("Copying file %s from bucket %s to target %s as %s",
		sourceKey, source, target, targetKey))
	_, err := c.client.CopyObject(context.TODO(), &s3.CopyObjectInput{
		Bucket:     aws.String(target),
		CopySource: aws.String(fmt.Sprintf("%v/%v", source, sourceKey)),
		Key:        aws.String(targetKey),
	})
	if err != nil {
		logger.Error(fmt.Sprintf("ERROR: Can not copy file %s to bucket %s due to error: %s",
			sourceKey, target, err))
		return false
	}
	return true
}

func doPathCutAnd(product, mainBucket, keyPrefix string,
	filePaths []string, extraPrefixedBuckets []util.Target,
	pathHandler func(a, b, c, d, e string, f, g int, h []util.Target) bool,
	root string) []string {
	slashRoot := root
	if strings.TrimSpace(root) == "" {
		slashRoot = "/"
	}
	if !strings.HasSuffix(root, "/") {
		slashRoot = slashRoot + "/"
	}
	var failedPaths []string
	index := 1
	filePathsCount := len(filePaths)
	for _, fullPath := range filePaths {
		fPath := strings.TrimPrefix(fullPath, slashRoot)
		if pathHandler(product, mainBucket,
			keyPrefix, fullPath, fPath, index,
			filePathsCount, extraPrefixedBuckets) {
			failedPaths = append(failedPaths, fullPath)
		}
		index += 1
	}
	return failedPaths
}
