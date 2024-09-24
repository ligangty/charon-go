package util

/*
Copyright (C) 2022 Red Hat, Inc. (https://github.com/Commonjava/charon)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
const (
	// Logging format used
	CHARON_LOGGING_FMT = "%(asctime)s - %(levelname)s - %(message)s"

	DESCRIPTION = `charon is a tool to synchronize several types of artifacts 
	repository data to RedHat Ronda service (maven.repository.redhat.com).`
	PROG = "charon"

	PROD_INFO_SUFFIX   = ".prodinfo"
	MANIFEST_SUFFIX    = ".txt"
	DEFAULT_ERRORS_LOG = "errors.log"

	DEFAULT_REGISTRY = "localhost"

	CONFIG_FILE = "charon.yaml"
)
