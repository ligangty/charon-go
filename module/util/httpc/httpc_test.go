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
	"testing"

	"github.com/stretchr/testify/assert"
	"org.commonjava/charon/module/util/test"
)

func TestGetHost(t *testing.T) {
	assert.Equal(t, "www.redhat.com", GetHost("https://www.redhat.com"))
	assert.Equal(t, "www.test.com", GetHost("http://www.test.com:8080"))
}

func TestGetPort(t *testing.T) {
	assert.Equal(t, "", GetPort("https://www.redhat.com"))
	assert.Equal(t, "8080", GetPort("http://www.test.com:8080"))
}

func TestHTTPRequstOK(t *testing.T) {
	test.MockHttpGetOkAnd("/", "Hello World", func(port int) {
		content, _, success := HTTPRequest(fmt.Sprintf("http://localhost:%d/", port), MethodGet, nil, true, nil, nil, "")
		assert.True(t, success)
		assert.Equal(t, "Hello World", content)
	})
}

func TestHTTPRequstNonExist(t *testing.T) {
	content, _, success := HTTPRequest("https://this.does.not.exist", MethodGet, nil, true, nil, nil, "")
	assert.Equal(t, "", content)
	assert.False(t, success)
}
