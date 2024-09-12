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

package httpc

import (
	"fmt"
	"strings"

	humanize "github.com/dustin/go-humanize"
)

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer
// interface and we can pass this into io.TeeReader() which will report progress on each
// write cycle.
type writeCounter struct {
	Total uint64
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.printProgress()
	return n, nil
}

func (wc writeCounter) printProgress() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 35))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}
