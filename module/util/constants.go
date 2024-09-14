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
	ARCHETYPE_CATALOG_FILENAME = "archetype-catalog.xml"
	//TODO: need to change to use go template
	ARCHETYPE_CATALOG_TEMPLATE = `
	<archetype-catalog>
		<archetypes>
		{% for arch in archetypes %}
			<archetype>
				<groupId>{{ arch.group_id }}</groupId>
				<artifactId>{{ arch.artifact_id }}</artifactId>
				<version>{{ arch.version }}</version>
				<description>{{ arch.description }}</description>
			</archetype>{% endfor %}
		</archetypes>
	</archetype-catalog>
	`
	// Logging format used
	CHARON_LOGGING_FMT = "%(asctime)s - %(levelname)s - %(message)s"

	DESCRIPTION = `charon is a tool to synchronize several types of artifacts 
	repository data to RedHat Ronda service (maven.repository.redhat.com).`
	PROG               = "charon"
	META_FILE_GEN_KEY  = "Generate"
	META_FILE_DEL_KEY  = "Delete"
	META_FILE_FAILED   = "Fail"
	PACKAGE_TYPE_MAVEN = "maven"
	PACKAGE_TYPE_NPM   = "npm"

	//TODO: need to change to use go template
	INDEX_HTML_TEMPLATE = `
	<!DOCTYPE html>
	<html>
	<head>
		<title>{{ index.title }}</title>
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<style>
	body {
		background: #fff;
	}
		</style>
	</head>
	<body>
		<header>
			<h1>{{ index.header }}</h1>
		</header>
		<hr/>
		<main>
			<ul style="list-style: none outside;" id="contents">{% for item in index.items %}
					<li><a href="{{ item }}" title="{{ item }}">{{ item }}</a></li>{% endfor%}
			</ul>
		</main>
		<hr/>
	</body>
	</html>
	`
	//TODO: need to change to use go template
	NPM_INDEX_HTML_TEMPLATE = `
	<!DOCTYPE html>
	<html>
	<head>
		<title>{{ index.title }}</title>
		<meta name="viewport" content="width=device-width, initial-scale=1.0">
		<style>
	body {
		background: #fff;
	}
		</style>
	</head>
	<body>
		<header>
			<h1>{{ index.header }}</h1>
		</header>
		<hr/>
		<main>
			<ul style="list-style: none outside;" id="contents">
					{% for item in index.items %}{% if item.startswith("@") or item.startswith("..") %}
					<li><a href="{{ item }}index.html" title="{{ item }}">{{ item }}</a></li>{% else %}
					<li><a href="{{ item }}" title="{{ item }}">{{ item }}</a></li>{% endif %}{% endfor%}
			</ul>
		</main>
		<hr/>
	</body>
	</html>
	`

	PROD_INFO_SUFFIX   = ".prodinfo"
	MANIFEST_SUFFIX    = ".txt"
	DEFAULT_ERRORS_LOG = "errors.log"

	DEFAULT_REGISTRY = "localhost"

	CONFIG_FILE = "charon.yaml"
)
