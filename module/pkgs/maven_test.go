package pkgs

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"org.commonjava/charon/module/storage"
	"org.commonjava/charon/module/util/archive"
	"org.commonjava/charon/module/util/files"
)

const TEST_REPO = "../../tests/input/commons-lang3.zip"

func TestMavenMetadata(t *testing.T) {
	meta := MavenMetadata{
		GroupId:        "foo.bar",
		ArtifactId:     "foobar",
		versions:       []string{"1.0", "4.0-beta", "2.0", "3.0", "5.0", "4.0", "4.0-alpha"},
		LastUpdateTime: time.Now().Format("2006-01-02 15:04:01"),
	}
	content, err := meta.GenerateMetaFileContent()
	assert.Nil(t, err)
	assert.Contains(t, content, "<groupId>foo.bar</groupId")
	assert.Contains(t, content, "<artifactId>foobar</artifactId")
	assert.Contains(t, content, "<versions>")
	assert.Contains(t, content, "<versioning>")
	assert.Contains(t, content, "<latest>5.0</latest>")
	assert.Contains(t, content, "<release>5.0</release>")
	for _, v := range meta.Versions() {
		assert.Contains(t, content, "<version>"+v+"</version>")
	}
	assert.Contains(t, content, "<lastUpdated>"+meta.LastUpdateTime+"</lastUpdated>")
}

func TestMavenArchetypeCatalog(t *testing.T) {
	archs := []ArchetypeRef{
		{GroupId: "io.quarkus", ArtifactId: "quarkus-core", Version: "1.0", Description: "quarkus-core 1.0"},
		{GroupId: "foo.bar", ArtifactId: "foobar", Version: "2.0", Description: "foobar 2.0"},
		{GroupId: "foo.bar", ArtifactId: "foobar", Version: "1.0", Description: "foobar 1.0"},
	}
	arch := NewMavenArchetypeCatalog(archs)
	content, err := arch.GenerateMetaFileContent()
	assert.Nil(t, err)
	for _, a := range archs {
		assert.Contains(t, content, "<groupId>"+a.GroupId+"</groupId>")
		assert.Contains(t, content, "<artifactId>"+a.ArtifactId+"</artifactId>")
		assert.Contains(t, content, "<version>"+a.Version+"</version>")
		assert.Contains(t, content, "<description>"+a.Description+"</description>")
	}
	fmt.Println(content)
}

func TestParseArchetypes(t *testing.T) {
	content := `<MavenArchetypeCatalog>
  <archetypes>
    <archetype>
      <groupId>foo.bar</groupId>
      <artifactId>foobar</artifactId>
      <version>1.0</version>
      <repository></repository>
      <description>foobar 1.0</description>
    </archetype>
    <archetype>
      <groupId>foo.bar</groupId>
      <artifactId>foobar</artifactId>
      <version>2.0</version>
      <repository></repository>
      <description>foobar 2.0</description>
    </archetype>
    <archetype>
      <groupId>io.quarkus</groupId>
      <artifactId>quarkus-core</artifactId>
      <version>1.0</version>
      <repository></repository>
      <description>quarkus-core 1.0</description>
    </archetype>
  </archetypes>
</MavenArchetypeCatalog>
`
	archs, err := parseArchetypes(content)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(archs))
	assert.Equal(t, ArchetypeRef{
		GroupId:     "foo.bar",
		ArtifactId:  "foobar",
		Version:     "1.0",
		Description: "foobar 1.0",
	}, archs[0])
	assert.Equal(t, ArchetypeRef{
		GroupId:     "foo.bar",
		ArtifactId:  "foobar",
		Version:     "2.0",
		Description: "foobar 2.0",
	}, archs[1])
	assert.Equal(t, ArchetypeRef{
		GroupId:     "io.quarkus",
		ArtifactId:  "quarkus-core",
		Version:     "1.0",
		Description: "quarkus-core 1.0",
	}, archs[2])

}

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
	archive.ExtractZipAll(TEST_REPO, dir)
	allPoms := scanForPoms(dir)
	assert.True(t, len(allPoms) > 0)
	for _, pom := range allPoms {
		if filepath.Ext(pom) != ".pom" {
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

func TestParseGAVsWithRoot(t *testing.T) {
	poms := []string{
		"/tmp/maven-repository/org/apache/maven/plugins/maven-plugin-plugin/1.0.0/maven-plugin-plugin-1.0.0.pom",
		"/tmp/maven-repository/org/apache/maven/plugins/maven-plugin-plugin/1.0.1/maven-plugin-plugin-1.0.1.pom",
		"/tmp/maven-repository/org/apache/maven/plugins/maven-plugin-plugin/1.2.0/maven-plugin-plugin-1.2.0.pom",
	}
	gavs := parseGAVs(poms, "/tmp/maven-repository")
	assert.Equal(t, 1, len(gavs))
	artifacts, ok := gavs["org.apache.maven.plugins"]
	assert.True(t, ok)
	assert.Equal(t, 1, len(artifacts))
	vers, ok := artifacts["maven-plugin-plugin"]
	assert.True(t, ok)
	assert.Equal(t, 3, len(vers))
	assert.Contains(t, vers, "1.0.0")
	assert.Contains(t, vers, "1.0.1")
	assert.Contains(t, vers, "1.2.0")
}

func TestGenerateMetadata(t *testing.T) {
	root, _ := os.MkdirTemp("", "charon-test-*")
	defer os.RemoveAll(root)
	existedPoms := []string{
		"org/apache/maven/plugin/maven-plugin-plugin/1.0.0/maven-plugin-plugin-1.0.0.pom",
		"org/apache/maven/plugin/maven-plugin-plugin/1.0.1/maven-plugin-plugin-1.0.1.pom",
		"org/apache/maven/plugin/maven-plugin-plugin/1.0.2/maven-plugin-plugin-1.0.2.pom",
		"io/quarkus/quarkus-core/2.0.0/quarkus-core-2.0.0.pom",
		"io/quarkus/quarkus-core/2.0.1/quarkus-core-2.0.1.pom",
	}
	prefix := "maven-repository"
	poms := []string{
		"org/apache/maven/plugin/maven-plugin-plugin/1.0.0/maven-plugin-plugin-1.0.0.pom",
		"io/quarkus/quarkus-core/2.0.0/quarkus-core-2.0.0.pom",
	}
	s3client, err := storage.S3ClientWithMock(storage.MockAWSS3Client{
		LsObjV2: func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
			contents := []types.Object{}
			for _, pom := range existedPoms {
				contents = append(contents, types.Object{Key: aws.String(pom)})
			}
			return &s3.ListObjectsV2Output{
				Contents: contents,
			}, nil
		},
	})
	assert.Nil(t, err)

	result := generateMetadatas(*s3client, poms, storage.TEST_BUCKET, prefix, root)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, 8, len(result[META_FILE_GEN_KEY]))

	metaFile := path.Join(root, "org/apache/maven/plugin/maven-plugin-plugin/maven-metadata.xml")
	assert.True(t, files.FileOrDirExists(metaFile))
	content, _ := files.ReadFile(metaFile)
	assert.Contains(t, content, "<version>1.0.0</version>")
	assert.Contains(t, content, "<version>1.0.1</version>")
	assert.Contains(t, content, "<version>1.0.2</version>")

	metaFile = path.Join(root, "io/quarkus/quarkus-core/maven-metadata.xml")
	assert.True(t, files.FileOrDirExists(metaFile))
	content, _ = files.ReadFile(metaFile)
	assert.Contains(t, content, "<version>2.0.0</version>")
	assert.Contains(t, content, "<version>2.0.1</version>")
}

func TestGenMetaFile(t *testing.T) {
	tmpRoot := extractTarball(TEST_REPO, "test", "")
	defer os.RemoveAll(tmpRoot)
	root := path.Join(tmpRoot, "apache-commons-maven-repository/maven-repository")
	poms := scanForPoms(root)
	gavMap := parseGAVs(poms, root)
	for g, avs := range gavMap {
		for a, vers := range avs {
			genMetaFile(g, a, vers, root, true)
		}
	}
	mavenMetaFile := path.Join(
		root, "org/apache/commons/commons-lang3/maven-metadata.xml")
	if !files.FileOrDirExists(mavenMetaFile) {
		assert.Fail(t, "maven-metadata is not generated correctly!")
	}
	metaContent, _ := files.ReadFile(mavenMetaFile)
	assert.Contains(t, metaContent, "<groupId>org.apache.commons</groupId>")
	assert.Contains(t, metaContent, "<artifactId>commons-lang3</artifactId>")
	lines := strings.Split(metaContent, "\n")
	count := 0
	pattern := regexp.MustCompile(".*<version>.*</version>.*")
	for _, l := range lines {
		if ok := pattern.MatchString(l); ok {
			count++
		}
	}
	assert.Equal(t, 13, count)
}

func TestScanPaths(t *testing.T) {
	fRoot := extractTarball(TEST_REPO, "test", "")
	defer os.RemoveAll(fRoot)
	assertPom := func(poms []string) {
		for _, p := range poms {
			if filepath.Ext(p) != ".pom" {
				assert.Fail(t, fmt.Sprintf("%s is not a pom file", p))
			}
		}
	}
	scannedPaths := scanPaths([]string{}, fRoot, "maven-repository")
	assert.Equal(t, "maven-repository", path.Base(scannedPaths.topLevel))
	assert.Equal(t, 30, len(scannedPaths.mvnPaths))
	assert.Equal(t, 13, len(scannedPaths.poms))
	assertPom(scannedPaths.poms)
	assert.Equal(t, 18, len(scannedPaths.dirs))

	scannedPaths = scanPaths([]string{"license.*", "README.*", ".*settings.xml.*"}, fRoot, "maven-repository")
	assert.Equal(t, "maven-repository", path.Base(scannedPaths.topLevel))
	assert.Equal(t, 27, len(scannedPaths.mvnPaths))
	assert.Equal(t, 13, len(scannedPaths.poms))
	assertPom(scannedPaths.poms)
	assert.Equal(t, 18, len(scannedPaths.dirs))
	fmt.Println(scannedPaths)
}
