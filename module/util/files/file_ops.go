/*
 *  Copyright (C) 2011-2020 Red Hat, Inc.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *          http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package files

import (
	"crypto"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"org.commonjava/charon/module/util"
)

const (
	MD5    crypto.Hash = crypto.MD5
	SHA1   crypto.Hash = crypto.SHA1
	SHA256 crypto.Hash = crypto.SHA256
)

func StoreFile(fileName string, content string, overWrite bool) {
	exists := false
	if FileOrDirExists(fileName) {
		if overWrite {
			// Printlnf("File %s exists, will overwrite it.", fileName)
			os.Remove(fileName)
		} else {
			exists = true
		}
	}

	var f *os.File
	var err error
	if !exists {
		folder := path.Dir(fileName)
		if !FileOrDirExists(folder) {
			os.MkdirAll(folder, 0700)
		}
		f, err = os.Create(fileName)
		if err != nil {
			panic(err)
		}
	} else {
		f, err = os.Open(fileName)
		if err != nil {
			panic(err)
		}
	}

	_, err = f.Write([]byte(content))
	if err != nil {
		panic(err)
	}
	defer f.Close()
}

func FileOrDirExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func IsFile(name string) bool {
	fi, err := os.Stat(name)
	if err != nil {
		return false
	}
	if mode := fi.Mode(); mode.IsRegular() {
		return true
	}
	return false
}

func IsDir(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fileInfo.IsDir()
}

func GuessMimetype(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return ""
	}
	return mime.TypeByExtension(ext)
}

func ReadFile(file string) (string, error) {
	if !FileOrDirExists(file) {
		return "", fmt.Errorf("file not found, %s", file)
	}

	f, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// This function will read sha1 hash of a file from a ${file}.sha1 file first, which should
// contain the sha1 has of the file. This is a maven repository rule which contains .sha1 files
// for artifact files. We can use this to avoid the digestion of big files which will improve
// performance. BTW, for some files like .md5, .sha1 and .sha256, they don't have .sha1 files as
// they are used for hashing, so we will directly calculate its sha1 hash through digesting.
func ReadSHA1(file string) string {
	nonSearchSuffix := []string{".md5", ".sha1", ".sha256", ".sha512"}
	suffix := filepath.Ext(file)
	if !slices.Contains(nonSearchSuffix, suffix) {
		sha1File := file + ".sha1"
		if IsFile(sha1File) {
			content, _ := ReadFile(sha1File)
			return content
		}
	}
	return Digest(file, crypto.SHA1)
}

func Digest(file string, hash crypto.Hash) string {
	if !IsFile(file) {
		return ""
	}
	f, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := hash.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}

	return hex.EncodeToString(h.Sum(nil))
}

// This function will caculate the hash value for the string content with the specified hash type
func DigestContent(content string, hash crypto.Hash) string {
	h := hash.New()
	io.WriteString(h, content)
	return hex.EncodeToString(h.Sum(nil))
}

func WriteManifest(paths []string, root, productKey string) (string, string) {
	manifestName := productKey + util.MANIFEST_SUFFIX
	manifestPath := path.Join(root, manifestName)
	artifacts := []string{}
	for _, p := range paths {
		p = strings.TrimPrefix(p, root)
		p = strings.TrimPrefix(p, "/")
		artifacts = append(artifacts, p)
	}
	StoreFile(manifestPath, strings.Join(artifacts, "\n"), true)
	return manifestName, manifestPath
}
