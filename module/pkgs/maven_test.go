package pkgs

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"org.commonjava/charon/module/util/archive"
)

// func TestMavenMetadata(t *testing.T) {
// 	meta := MavenMetadata{
// 		GroupId:        "foo.bar",
// 		ArtifactId:     "foobar",
// 		versions:       []string{"1.0", "4.0-beta", "2.0", "3.0", "5.0", "4.0", "4.0-alpha"},
// 		LastUpdateTime: time.Now().Format("2006-01-02 15:04:01"),
// 	}
// 	content, err := meta.GenerateMetaFileContent()
// 	assert.Nil(t, err)
// 	fmt.Println(content)
// }

func TestVersionsCompare(t *testing.T) {
	// Normal versions comparasion
	assert.Equal(t, -1, versionCompare("1.0.0", "1.0.1"))
	assert.Equal(t, 1, versionCompare("1.10.0", "1.9.1"))
	assert.Equal(t, 0, versionCompare("1.0.1", "1.0.1"))
	assert.Equal(t, 1, versionCompare("2.0.1", "1.0.1"))

	// # Special versions comparasion
	assert.Equal(t, 1, versionCompare("1.0.1-alpha", "1.0.1"))
	assert.Equal(t, 1, versionCompare("1.0.1-beta", "1.0.1-alpha"))
	assert.Equal(t, 1, versionCompare("1.0.2", "1.0.1-alpha"))
	assert.Equal(t, 1, versionCompare("1.0.1", "1.0-m2"))
	assert.Equal(t, 1, versionCompare("1.0.2-alpha", "1.0.1-m2"))
	assert.Equal(t, 1, versionCompare("1.0.2-alpha", "1.0.1-alpha"))
}

func TestScanForPoms(t *testing.T) {
	dir, _ := os.MkdirTemp("", "charon-test-*")
	archive.ExtractZipAll("../../tests/input/commons-lang3.zip", dir)
	allPoms := scanForPoms(dir)
	assert.True(t, len(allPoms) > 0)
	for _, pom := range allPoms {
		if !strings.HasSuffix(pom, ".pom") {
			assert.Fail(t, "%s is not a pom", pom)
		}
	}
	os.RemoveAll(dir)
}

func TestParseGA(t *testing.T) {
	assert.Equal(t, [2]string{"org.apache.maven.plugin", "maven-plugin-plugin"},
		parseGA("org/apache/maven/plugin/maven-plugin-plugin", ""))
	assert.Equal(t, [2]string{"org.apache.maven.plugin", "maven-plugin-plugin"},
		parseGA("org/apache/maven/plugin/maven-plugin-plugin/", ""))
	assert.Equal(t, [2]string{"org.apache.maven.plugin", "maven-plugin-plugin"},
		parseGA("/org/apache/maven/plugin/maven-plugin-plugin/", ""))
}

func TestParseGAV(t *testing.T) {
	assert.Equal(t, [3]string{"org.apache.maven.plugin", "maven-plugin-plugin", "1.0.0"},
		parseGAV("org/apache/maven/plugin/maven-plugin-plugin/1.0.0/maven-plugin-plugin-1.0.0.pom", ""))
	assert.Equal(t, [3]string{"org.apache.maven.plugin", "maven-plugin-plugin", "1.0.0"},
		parseGAV("org/apache/maven/plugin/maven-plugin-plugin/1.0.0/maven-plugin-plugin-1.0.0.pom/", ""))
	assert.Equal(t, [3]string{"org.apache.maven.plugin", "maven-plugin-plugin", "1.0.0"},
		parseGAV("/org/apache/maven/plugin/maven-plugin-plugin/1.0.0/maven-plugin-plugin-1.0.0.pom/", ""))
}

func TestParseGAVs(t *testing.T) {
	poms := []string{
		"org/apache/maven/plugin/maven-plugin-plugin/1.0.0/maven-plugin-plugin-1.0.0.pom",
		"org/apache/maven/plugin/maven-plugin-plugin/1.0.1/maven-plugin-plugin-1.0.1.pom",
		"org/apache/maven/plugin/maven-compiler-plugin/1.0.3/maven-compiler-plugin-1.0.3.pom",
		"org/apache/maven/plugin/maven-compiler-plugin/1.0.4/maven-compiler-plugin-1.0.4.pom",
		"io/quarkus/quarkus-bom/1.0/quarkus-bom-1.0.pom",
		"io/quarkus/quarkus-bom/1.1/quarkus-bom-1.1.pom",
	}
	gavs := parseGAVs(poms, "")
	assert.Equal(t, 2, len(gavs))
	artifacts, ok := gavs["org.apache.maven.plugin"]
	assert.True(t, ok)
	assert.Equal(t, 2, len(artifacts))
	vers, ok := artifacts["maven-plugin-plugin"]
	assert.True(t, ok)
	assert.Equal(t, 2, len(vers))
	assert.Contains(t, vers, "1.0.0")
	assert.Contains(t, vers, "1.0.1")
	vers, ok = artifacts["maven-compiler-plugin"]
	assert.True(t, ok)
	assert.Equal(t, 2, len(vers))
	assert.Contains(t, vers, "1.0.3")
	assert.Contains(t, vers, "1.0.4")
	artifacts, ok = gavs["io.quarkus"]
	assert.True(t, ok)
	assert.Equal(t, 1, len(artifacts))
	vers, ok = artifacts["quarkus-bom"]
	assert.True(t, ok)
	assert.Equal(t, 2, len(vers))
	assert.Contains(t, vers, "1.0")
	assert.Contains(t, vers, "1.1")
}
