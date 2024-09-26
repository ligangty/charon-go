package test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"org.commonjava/charon/module/util"
	"org.commonjava/charon/module/util/files"
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
	if !util.IsBlankString(b.oldHome) && os.Getenv("HOME") != b.oldHome {
		os.Setenv("HOME", b.oldHome)
	}
	if !util.IsBlankString(b.tmpDir) && files.FileOrDirExists(b.tmpDir) {
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
    domain: "npm.registry.redhat.com"

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
	configPath := path.Join(configBase, "charon.yaml")
	files.StoreFile(configPath, fileContent, true)
	if !files.FileOrDirExists(configPath) {
		return fmt.Errorf("configuration initilization failed")
	}
	return nil
}

func MockHttpGetOkAnd(webRoot, expect string, todo func(port int)) {
	MockHttpServerAnd(webRoot, expect, http.StatusOK, todo)
}

func MockHttpServerAnd(webRoot, expect string, statusCode int, todo func(port int)) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	srv := &http.Server{}
	http.HandleFunc(webRoot, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		io.WriteString(w, expect)
	})
	httpServerExitDone := &sync.WaitGroup{}
	httpServerExitDone.Add(1)
	go func() {
		defer httpServerExitDone.Done() // let main know we are done cleaning up

		// always returns error. ErrServerClosed on graceful close
		if err := srv.Serve(listener); err != http.ErrServerClosed {
			// unexpected error. port in use?
			log.Fatalf("Serve(): %v", err)
		}
	}()
	time.Sleep(500 * time.Millisecond) // wait to start

	port := listener.Addr().(*net.TCPAddr).Port
	fmt.Printf("Using port:%d\n", port)
	todo(port)

	if err := srv.Shutdown(context.TODO()); err != nil {
		panic(err) // failure/timeout shutting down the server gracefully
	}
	httpServerExitDone.Wait()
}
