package config

import (
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"org.commonjava/charon/module/util/files"
	"org.commonjava/charon/module/util/test"
)

var bt *test.BaseTest = &test.BaseTest{}

func TestConfig(t *testing.T) {
	bt.SetUp()
	defer bt.TearDown()
	conf, err := GetConfig("")
	assert.Nil(t, err)
	assert.Equal(t, "test", conf.AwsProfile)
	assert.False(t, conf.AwsCFEnable)
	assert.Equal(t, []string{".*^(redhat).*", ".*snapshot.*"}, conf.IgnorePatterns)
	assert.Equal(t, 1, len(conf.GetTarget("ga")))
	assert.Equal(t, 1, len(conf.GetTarget("ea")))
	assert.Equal(t, 1, len(conf.GetTarget("npm")))
	assertTarget(t, &Target{Bucket: "charon-test", Prefix: "ga", Registry: "localhost"}, conf.GetTarget("ga")[0])
	assertTarget(t, &Target{Bucket: "charon-test-ea", Prefix: "earlyaccess/all", Registry: "localhost"}, conf.GetTarget("ea")[0])
	assertTarget(t, &Target{Bucket: "charon-test-npm", Prefix: "", Registry: "npm1.registry.redhat.com", Domain: "npm.registry.redhat.com"}, conf.GetTarget("npm")[0])
	assert.Equal(t, "touch {{ file }}.asc", conf.SignatureCommand)
	assert.Equal(t, []string{".sha1", ".sha256", ".md5", "maven-metadata.xml", "archtype-catalog.xml"}, conf.GetIgnoreSignatureSuffix("maven"))
	assert.Equal(t, []string{"package.json"}, conf.GetIgnoreSignatureSuffix("npm"))
	assert.Equal(t, "manifest", conf.ManifestBucket)
}

func assertTarget(t *testing.T, expected, actual *Target) {
	assert.Equal(t, expected.Bucket, actual.Bucket)
	assert.Equal(t, expected.Prefix, actual.Prefix)
	assert.Equal(t, expected.Registry, actual.Registry)
	assert.Equal(t, expected.Domain, actual.Domain)
}

func TestNoConfig(t *testing.T) {
	bt.ChangeHome()
	resetGlobal()
	defer bt.TearDown()
	_, err := GetConfig("")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestNonDefaultConfig(t *testing.T) {
	configContent := `
ignore_patterns:
- ".*^(redhat).*"

targets:
  changed:
  - bucket: changed-bucket
    prefix: changed-prefix
`
	tmpConfigFile, _ := os.CreateTemp("", "charon-test-config-*.yaml")
	files.StoreFile(tmpConfigFile.Name(), configContent, true)
	conf, err := GetConfig(tmpConfigFile.Name())
	assert.Nil(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, 1, len(conf.GetTarget("changed")))
	assert.Equal(t, "changed-bucket", conf.GetTarget("changed")[0].Bucket)
	assert.Equal(t, "changed-prefix", conf.GetTarget("changed")[0].Prefix)
	os.Remove(tmpConfigFile.Name())
}

func TestConfigMissingTargets(t *testing.T) {
	contentMissingTargets := `ignore_patterns:
- ".*^(redhat).*"
- ".*snapshot.*"
`
	resetGlobal()
	defer bt.TearDown()
	bt.ChangeConfigContent(contentMissingTargets)
	msg := "'targets' is a required property"
	_, err := GetConfig("")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), msg)
}

func TestConfigMissingBucket(t *testing.T) {
	contentMissingTargets := `ignore_patterns:
- ".*^(redhat).*"
- ".*snapshot.*"

targets:
  ga:
  - prefix: ga
`
	resetGlobal()
	defer bt.TearDown()
	bt.ChangeConfigContent(contentMissingTargets)
	msg := "'bucket' is a required property"
	_, err := GetConfig("")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), msg)
}

func TestConfigMissingPrefix(t *testing.T) {
	contentMissingTargets := `ignore_patterns:
- ".*^(redhat).*"
- ".*snapshot.*"

targets:
  ga:
  - bucket: charon-test
`
	resetGlobal()
	defer bt.TearDown()
	bt.ChangeConfigContent(contentMissingTargets)
	conf, err := GetConfig("")
	assert.Nil(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, "charon-test", conf.GetTarget("ga")[0].Bucket)
	assert.Equal(t, "", conf.GetTarget("ga")[0].Prefix)
}

func TestConfigMissingRegistry(t *testing.T) {
	contentMissingTargets := `ignore_patterns:
- ".*^(redhat).*"
- ".*snapshot.*"

targets:
  npm:
  - bucket: charon-npm-test
`
	resetGlobal()
	defer bt.TearDown()
	bt.ChangeConfigContent(contentMissingTargets)
	conf, err := GetConfig("")
	assert.Nil(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, "charon-npm-test", conf.GetTarget("npm")[0].Bucket)
	assert.Equal(t, "localhost", conf.GetTarget("npm")[0].Registry)
}

func TestIgnorePatterns(t *testing.T) {
	contentMissingTargets := `ignore_patterns:
  - '\.nexus.*' # noqa: W605
  - '\.index.*' # noqa: W605
  - '\.meta.*' # noqa: W605
  - '^\..+'  # path with a filename that starts with a dot # noqa: W605
  - 'index\.html.*' # noqa: W605

targets:
  ga:
  - bucket: charon-test
`
	resetGlobal()
	defer bt.TearDown()
	bt.ChangeConfigContent(contentMissingTargets)
	conf, err := GetConfig("")
	assert.Nil(t, err)
	assert.NotNil(t, conf)
	assert.Equal(t, 5, len(conf.IgnorePatterns))
	assert.True(t, isIgnored(".index.html", conf.IgnorePatterns))
	assert.True(t, isIgnored(".abcxyz.jar", conf.IgnorePatterns))
	assert.True(t, isIgnored("index.html", conf.IgnorePatterns))
	assert.True(t, isIgnored(".nexuxabc", conf.IgnorePatterns))
	assert.False(t, isIgnored("abcxyz.jar", conf.IgnorePatterns))
	assert.False(t, isIgnored("abcxyz.pom", conf.IgnorePatterns))
	assert.False(t, isIgnored("abcxyz.jar.md5", conf.IgnorePatterns))
}

func isIgnored(filename string, ignorePatterns []string) bool {
	if len(ignorePatterns) > 0 {
		for _, dirs := range ignorePatterns {
			if match, _ := regexp.MatchString(dirs, filename); match {
				return true
			}
		}
	}
	return false
}
