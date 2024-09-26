package pkgs

import "org.commonjava/charon/module/storage"

func generateIndexes(s3Client storage.S3Client, changedDirs []string,
	packageType, topLevel, bucket, prefix string) []string {
	// TODO: not implemented yet
	return []string{}
}
