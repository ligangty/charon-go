package archive

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const TEST_INPUT_PATH = "../../../tests/input"

func TestDetectPackage(t *testing.T) {
	mvnZip := path.Join(TEST_INPUT_PATH, "commons-lang3.zip")
	assert.Equal(t, NOT_NPM, DetectNPMArchive(mvnZip))
	npmTarball := path.Join(TEST_INPUT_PATH, "code-frame-7.14.5.tgz")
	assert.Equal(t, TAR_FILE, DetectNPMArchive(npmTarball))
}

func TestExtractZipAll(t *testing.T) {
	mvnZip := path.Join(TEST_INPUT_PATH, "commons-lang3.zip")
	tempDir, _ := os.MkdirTemp("", "charon-test-*")
	defer os.RemoveAll(tempDir)

	ExtractZipAll(mvnZip, tempDir)

	count := 0
	containsJar := false
	containsPom := false
	filepath.WalkDir(tempDir, func(path string, d fs.DirEntry, err error) error {
		if d.Type().IsRegular() {
			count++
			if strings.HasSuffix(d.Name(), ".jar") {
				containsJar = true
			}
			if strings.HasSuffix(d.Name(), ".pom") {
				containsPom = true
			}
		}

		return nil
	})
	assert.True(t, count > 0)
	assert.True(t, containsJar)
	assert.True(t, containsPom)
}
