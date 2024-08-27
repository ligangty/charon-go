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

package util

import (
	"fmt"
	"io"
	"os"
	"path"
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

func ReadFile(file string) (string, error) {
	if !FileOrDirExists(file) {
		return "", fmt.Errorf("File not found, %s", file)
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
