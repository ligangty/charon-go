package util

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileExists(t *testing.T) {
	assert.True(t, FileOrDirExists("/usr/bin/bash"))
	assert.False(t, FileOrDirExists("/kljsdflksdjf"))
}

func TestStoreFile(t *testing.T) {
	fileName := fmt.Sprintf("/tmp/%d", NowInMillis())
	fmt.Println(fileName)
	fileContent := "This is a test."
	StoreFile(fileName, fileContent, true)
	assert.True(t, FileOrDirExists(fileName), "Stored file should exist")

	f, _ := os.Open(fileName)
	defer f.Close()
	actual, _ := io.ReadAll(f)
	assert.Equal(t, fileContent, string(actual), "Stored file content should be correct")

}

func NowInMillis() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}
