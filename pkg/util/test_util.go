package util

import (
	"fmt"
	"os"
	"path"
	"strings"
)

type BaseTest struct {
	tmpDir  string
	oldHome string
}

func (b *BaseTest) SetUp() {
	b.ChangeHome()
	configBase := b.getConfigBase()
	configContent := getConfigContent()
	prepareConfig(configBase, configContent)
}

func (b *BaseTest) TearDown() {
	if strings.TrimSpace(b.oldHome) != "" && os.Getenv("HOME") != b.oldHome {
		os.Setenv("HOME", b.oldHome)
	}
	if strings.TrimSpace(b.tmpDir) != "" && FileOrDirExists(b.tmpDir) {
		os.RemoveAll(b.tmpDir)
	}
}

func (b *BaseTest) getConfigBase() string {
	return path.Join(b.tmpDir, ".charon")
}
func getConfigContent() string {
	return `aws_profile: "test"
aws_cf_enable: false
ignore_patterns:
- ".*^(redhat).*"
- ".*snapshot.*"

ignore_signature_suffix:
  maven:
    - ".sha1"
    - ".sha256"
    - ".md5"
    - "maven-metadata.xml"
    - "archtype-catalog.xml"
  npm:
    - "package.json"

detach_signature_command: "touch {{ file }}.asc"

targets:
  ga:
  - bucket: "charon-test"
    prefix: ga
  ea:
  - bucket: "charon-test-ea"
    prefix: earlyaccess/all
  npm:
  - bucket: "charon-test-npm"
    registry: "npm1.registry.redhat.com"

manifest_bucket: "manifest"
`
}

func (b *BaseTest) ChangeHome() {
	old := os.Getenv("HOME")
	tp, err := os.MkdirTemp("", "charon-test-*")
	if err != nil {
		panic(err)
	}
	b.tmpDir = tp
	os.Setenv("HOME", b.tmpDir)
	b.oldHome = old
}

func (b *BaseTest) ChangeConfigContent(content string) {
	b.ChangeHome()
	configBase := b.getConfigBase()
	os.Mkdir(configBase, 0755)
	prepareConfig(configBase, content)
}

func prepareConfig(configBase, fileContent string) error {
	configPath := path.Join(configBase, CONFIG_FILE)
	StoreFile(configPath, fileContent, true)
	if !FileOrDirExists(configPath) {
		return fmt.Errorf("configuration initilization failed")
	}
	return nil
}
