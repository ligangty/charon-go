package pkgs

const (
	MAVEN_METADATA_TEMPLATE = `<metadata>
  {{if .GroupId -}}
  <groupId>{{.GroupId}}</groupId>
  {{- end}}
  {{if .ArtifactId -}}
  <artifactId>{{.ArtifactId}}</artifactId>
  {{- end}}
  {{if .Versions -}}
  <versioning>
    {{if .LatestVersion -}}
    <latest>{{.LatestVersion}}</latest>
    {{- end}}
    {{if .ReleaseVersion -}}
    <release>{{.ReleaseVersion}}</release>
    {{- end}}
    {{if .Versions -}}
    <versions>
      {{range $ver := .Versions -}}
      <version>{{$ver}}</version>
      {{end}}
    </versions>
    {{- end}}
    {{if .LastUpdateTime -}}
    <lastUpdated>{{.LastUpdateTime}}</lastUpdated>
    {{- end}}
  </versioning>
  {{- end}}
</metadata>
`

	// TODO: need to change to use golang html template
	ARCHETYPE_CATALOG_TEMPLATE = `<archetype-catalog>
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
	//TODO: need to change to use go template
	INDEX_HTML_TEMPLATE = `<!DOCTYPE html>
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
	// TODO: need to change to use go template
	NPM_INDEX_HTML_TEMPLATE = `<!DOCTYPE html>
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
)
const (
	MAVEN_METADATA_FILE = "maven-metadata.xml"
	MAVEN_ARCH_FILE     = "archetype-catalog.xml"
	META_FILE_GEN_KEY   = "Generate"
	META_FILE_DEL_KEY   = "Delete"
	META_FILE_FAILED    = "Fail"
	PACKAGE_TYPE_MAVEN  = "maven"
	PACKAGE_TYPE_NPM    = "npm"
)

var (
	STANDARD_GENERATED_IGNORES = []string{MAVEN_METADATA_FILE, MAVEN_ARCH_FILE}
)
