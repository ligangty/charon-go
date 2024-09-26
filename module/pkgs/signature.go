package pkgs

import "org.commonjava/charon/module/storage"

func generateSign(s3Client storage.S3Client, artifactPath []string,
	packageType, topLevel, prefix, bucket, key, command string,
) ([]string, []string) {
	// TODO: not implemented yet!
	return nil, nil
}
