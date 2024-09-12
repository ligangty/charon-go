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
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"org.commonjava/charon/module/util/files"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

// ContentType for RFC http content type (parts)
const (
	ContentTypePlain = "text/plain"
	ContentTypeHTML  = "text/html"

	ContentTypeJSON = "application/json"
	ContentTypeXML  = "application/xml"

	ContentTypeZip    = "application/zip"
	ContentTypeStream = "application/octet-stream"
	CottentTypeJar    = "application/java-archive"
)

// Status code for RFC http response status code (parts)
const (
	StatusOK        = http.StatusOK
	StatusCreated   = http.StatusCreated
	StatusAccepted  = http.StatusAccepted
	StatusNoContent = http.StatusNoContent

	StatusMultipleChoices  = http.StatusMultipleChoices
	StatusMovedPermanently = http.StatusMovedPermanently
	StatusFound            = http.StatusFound
	StatusSeeOther         = http.StatusSeeOther
	StatusNotModified      = http.StatusNotModified
	StatusUseProxy         = http.StatusUseProxy

	StatusBadRequest        = http.StatusBadRequest
	StatusUnauthorized      = http.StatusUnauthorized
	StatusForbidden         = http.StatusForbidden
	StatusNotFound          = http.StatusNotFound
	StatusMethodNotAllowed  = http.StatusMethodNotAllowed
	StatusNotAcceptable     = http.StatusNotAcceptable
	StatusProxyAuthRequired = http.StatusProxyAuthRequired
	StatusRequestTimeout    = http.StatusRequestTimeout
	StatusConflict          = http.StatusConflict

	StatusInternalServerError = http.StatusInternalServerError
	StatusNotImplemented      = http.StatusNotImplemented
	StatusBadGateway          = http.StatusBadGateway
	StatusServiceUnavailable  = http.StatusServiceUnavailable
	StatusGatewayTimeout      = http.StatusGatewayTimeout

	StatusUnknown = -1
)

// Methods for RFC http methods
const (
	MethodGet     = http.MethodGet
	MethodHead    = http.MethodHead
	MethodPost    = http.MethodPost
	MethodPut     = http.MethodPut
	MethodPatch   = http.MethodPatch
	MethodDelete  = http.MethodDelete
	MethodOptions = http.MethodOptions
)

const NotStoreFile = ""

type errorHandler func()

type Authenticate func(request *http.Request) error

// GetHost gets the hostname from a url string
func GetHost(URLString string) string {
	u, err := url.Parse(URLString)
	if err != nil {
		return ""
	}

	return u.Hostname()
}

// GetPort gets the port from a url string
func GetPort(URLString string) string {
	u, err := url.Parse(URLString)
	if err != nil {
		return "-1"
	}

	return u.Port()
}

// HTTPRequest do raw http request with method, input data and headers. If url is trying to access bin content(like file), can use filename parameter to specify where to store this file as.
// Parameters: request url; request method; authentication method; if need response content; data payload to send(POST or PUT); headers to send; the file location to store if response is a binary download; if print verbose log message
// Returned: content as string, response status code as int, if succeeded as bool
func HTTPRequest(url, method string, auth Authenticate, needResult bool, dataPayload io.Reader, headers map[string]string, filename string) (string, int, bool) {
	resp, err := doHttp(url, method, auth, dataPayload, headers)
	if err != nil {
		logger.Error(fmt.Sprintf("Cannot do http request for %s, error: %s", url, err))
		return "", StatusUnknown, false
	}
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(fmt.Sprintf("Cannot read http request body for %s, error: %s", url, err))
		return "", StatusUnknown, false
	}

	if resp.StatusCode >= 400 {
		return string(content), resp.StatusCode, false
	}

	if !isBinContent(resp.Header) && needResult {
		return string(content), resp.StatusCode, true
	}

	return "", resp.StatusCode, true
}

func DownloadFile(url, filename string, auth Authenticate) (string, error) {
	resp, err := doHttp(url, MethodGet, auth, nil, nil)
	if err != nil {
		logger.Error(
			fmt.Sprintf("Can not download file %s, error: %s", url, err))
		return "", err
	}
	defer resp.Body.Close()
	logger.Debug("The api is trying to download a file")
	conDispo := resp.Header.Get("Content-Disposition")
	filePath := ""
	if strings.TrimSpace(filename) != "" {
		filePath = strings.TrimSpace(filename)
	} else {
		if strings.TrimSpace(conDispo) != "" {
			start := strings.Index(conDispo, "filename")
			filePath = conDispo[start:]
			splitted := strings.Split(filePath, "=")
			filePath = splitted[1]
		} else {
			filePath = path.Base(url)
		}
		filePath = "./" + filePath
	}
	folder := path.Dir(filePath)
	if !files.FileOrDirExists(folder) {
		os.MkdirAll(folder, 0755)
	}
	// Check and create the file
	for files.FileOrDirExists(filePath) {
		filePath = filePath + ".1"
	}
	out, err := os.Create(filePath)
	if err != nil {
		logger.Error(
			fmt.Sprintf("Cannot download file due to io error! Error: %s", err))
		return "", err
	} else {
		defer out.Close()
		logger.Debug("Download started.\n")
		counter := &writeCounter{}
		_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
		if err != nil {
			logger.Error("Cannot download file due to io error!")
			return "", err
		} else {
			logger.Info(fmt.Sprintf("\nFile downloaded as %s\n", filePath))
		}
	}
	return filePath, nil
}

func doHttp(url, method string, auth Authenticate, dataPayload io.Reader, headers map[string]string) (*http.Response, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, dataPayload)
	if err != nil {
		logger.Error(
			fmt.Sprintf("Http request failed due to http error %s", err))
		return nil, err
	}
	if len(headers) > 0 {
		for key, val := range headers {
			req.Header.Add(key, val)
		}
	}
	if auth != nil {
		err := auth(req)
		if err != nil {
			logger.Error(
				fmt.Sprintf("Http request failed due to authentication error %s", err))
			return nil, err
		}
	}

	return client.Do(req)
}

// HTTPError represents a generic http problem
type HTTPError struct {
	Message    string
	StatusCode int
}

func (err HTTPError) Error() string {
	return err.Message
}

func isBinContent(headers http.Header) bool {
	contentType := headers.Get("Content-Type")
	// logger.Info(contentType)
	if strings.HasPrefix(contentType, "text") {
		return false
	}
	if contentType == ContentTypeJSON || contentType == ContentTypeXML {
		return false
	}

	return true
}
