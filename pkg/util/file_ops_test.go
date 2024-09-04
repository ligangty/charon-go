package util

import (
	"crypto"
	"fmt"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileExists(t *testing.T) {
	assert.True(t, FileOrDirExists("/usr/bin/bash"))
	assert.False(t, FileOrDirExists("/kljsdflksdjf"))
}

func TestStoreFile(t *testing.T) {
	fileName := fmt.Sprintf("/tmp/%d", nowInMillis())
	fmt.Println(fileName)
	fileContent := "This is a test."
	StoreFile(fileName, fileContent, true)
	assert.True(t, FileOrDirExists(fileName), "Stored file should exist")

	f, _ := os.Open(fileName)
	defer f.Close()
	actual, _ := io.ReadAll(f)
	assert.Equal(t, fileContent, string(actual), "Stored file content should be correct")

}

func TestIsFile(t *testing.T) {
	StoreFile("/tmp/testDir/testFile", "test content", false)
	assert.True(t, IsFile("/tmp/testDir/testFile"))
	assert.False(t, IsFile("/tmp/testDir/notExist"))
	assert.False(t, IsFile("/tmp/testDir"))
	os.RemoveAll("/tmp/testDir")
}

func TestGuessMimetype(t *testing.T) {
	assert.Equal(t, "", GuessMimetype("/tmp/abc"))
	assert.Equal(t, "text/html; charset=utf-8", GuessMimetype("/tmp/abc.html"))
	assert.Equal(t, "text/xml; charset=utf-8", GuessMimetype("/tmp/abc.xml"))
	assert.Equal(t, "text/plain; charset=utf-8", GuessMimetype("/tmp/abc.txt"))
	assert.Equal(t, "application/json", GuessMimetype("/tmp/abc.json"))
	assert.Equal(t, "application/zip", GuessMimetype("/tmp/abc.zip"))
	assert.Equal(t, "application/x-tar", GuessMimetype("/tmp/abc.tar"))
	assert.Equal(t, "application/gzip", GuessMimetype("/tmp/abc.tar.gz"))
}

func TestDigest(t *testing.T) {
	testFile := path.Join("../../tests/input", "commons-lang3.zip")
	fmt.Println(testFile)
	assert.Equal(t, "bd4fe0a8111df64430b6b419a91e4218ddf44734", Digest(testFile, crypto.SHA1))
	assert.Equal(t,
		"61ff1d38cfeb281b05fcd6b9a2318ed47cd62c7f99b8a9d3e819591c03fe6804",
		Digest(testFile, crypto.SHA256))
}

func TestDigestContent(t *testing.T) {
	testContent := "test common content"
	assert.Equal(t, "8c7b70f25fb88bc6a0372f70f6805132e90e2029", DigestContent(testContent, crypto.SHA1))
	assert.Equal(t,
		"1a1c26da1f6830614ed0388bb30d9e849e05bba5de4031e2a2fa6b48032f5354",
		DigestContent(testContent, crypto.SHA256),
	)
}

func TestReadSHA1(t *testing.T) {
	testFile := path.Join("../../tests/input", "commons-lang3.zip")
	// read the real sha1 hash
	assert.Equal(t, "bd4fe0a8111df64430b6b419a91e4218ddf44734", Digest(testFile, crypto.SHA1))
	// read hash from .sha1 file
	assert.Equal(t, "bd4fe0a8111df64430b6b419a91e4218ddf44734", ReadSHA1(testFile))

	// For .sha1 file itself, will use digest directly
	testFile = path.Join("../../tests/input", "commons-lang3.zip.sha1")
	assert.Equal(t, Digest(testFile, crypto.SHA1), ReadSHA1(testFile))
}

func nowInMillis() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}
