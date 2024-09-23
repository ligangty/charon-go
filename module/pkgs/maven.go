package pkgs

import (
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/template"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

const MAVEN_METADATA_TEMPLATE = `<metadata>
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

const (
	MAVEN_METADATA_FILE = "maven-metadata.xml"
	MAVEN_ARCH_FILE     = "archetype-catalog.xml"
)

var (
	STANDARD_GENERATED_IGNORES = []string{MAVEN_METADATA_FILE, MAVEN_ARCH_FILE}
)

// This MavenMetadata will represent a maven-metadata.xml data content
// which will be used in jinja2 or other places
type MavenMetadata struct {
	GroupId        string
	ArtifactId     string
	LastUpdateTime string
	versions       []string
	latestVersion  string
	releaseVersion string
}

func (m *MavenMetadata) LatestVersion() string {
	if strings.TrimSpace(m.latestVersion) != "" {
		return m.latestVersion
	}
	versions := m.Versions()
	m.latestVersion = versions[len(versions)-1]
	return m.latestVersion
}
func (m *MavenMetadata) ReleaseVersion() string {
	if strings.TrimSpace(m.releaseVersion) != "" {
		return m.releaseVersion
	}
	versions := m.Versions()
	m.releaseVersion = versions[len(versions)-1]
	return m.releaseVersion
}
func (m *MavenMetadata) Versions() []string {
	vers := m.versions
	slices.SortFunc(vers, versionCompare)
	m.versions = vers
	return m.versions
}
func (m *MavenMetadata) String() string {
	return fmt.Sprintf("%s:%s:\n%s\n\n", m.GroupId, m.ArtifactId, m.Versions())
}
func (m *MavenMetadata) GenerateMetaFileContent() (string, error) {
	t := template.Must(template.New("settings").Parse(MAVEN_METADATA_TEMPLATE))
	var buf bytes.Buffer
	err := t.Execute(&buf, m)
	if err != nil {
		logger.Error(fmt.Sprintf("executing template: %s", err))
		return "", err
	}
	return buf.String(), nil
}

// This ArchetypeRef will represent an entry in archetype-catalog.xml content
// which will be used in jinja2 or other places
type ArchetypeRef struct {
	GroupId     string
	ArtifactId  string
	Version     string
	Description string
}

func (m ArchetypeRef) String() string {
	return fmt.Sprintf("%s:%s\n%s\n%s\n\n",
		m.GroupId, m.ArtifactId, m.Version, m.Description)
}

// This MavenArchetypeCatalog represents an archetype-catalog.xml which will be
// used in jinja2 to regenerate the file with merged contents
type MavenArchetypeCatalog struct {
	Archetypes []ArchetypeRef
}

func isInt(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func versionCompare(ver1, ver2 string) int {
	xitems := strings.Split(ver1, ".")
	if strings.Contains(xitems[len(xitems)-1], "-") {
		xitems = append(xitems[0:len(xitems)-1], strings.Split(xitems[len(xitems)-1], "-")...)
	}
	yitems := strings.Split(ver2, ".")
	if strings.Contains(yitems[len(yitems)-1], "-") {
		yitems = append(yitems[0:len(yitems)-1], strings.Split(yitems[len(yitems)-1], "-")...)
	}
	big := max(len(xitems), len(yitems))
	for i := 0; i < big; i++ {
		if i >= len(xitems) {
			return -1
		}
		if i >= len(yitems) {
			return 1
		}
		xitem := xitems[i]
		yitem := yitems[i]
		if isInt(xitem) && !isInt(yitem) {
			return 1
		} else if !isInt(xitem) && isInt(yitem) {
			return -1
		} else if isInt(xitem) && isInt(yitem) {
			xitemInt, _ := strconv.Atoi(xitem)
			yitemInt, _ := strconv.Atoi(yitem)
			if xitemInt > yitemInt {
				return 1
			} else if xitemInt < yitemInt {
				return -1
			}
		} else {
			if xitem > yitem {
				return 1
			} else if xitem < yitem {
				return -1
			} else {
				continue
			}
		}
	}

	return 0
}

// Scan a file path and finds all pom files absolute paths
func scanForPoms(fullPath string) []string {
	allPomPaths := []string{}
	filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && strings.HasSuffix(path, ".pom") {
			allPomPaths = append(allPomPaths, path)
		}
		return nil
	})
	return allPomPaths
}

// Parse maven groupId and artifactId from a standard path in a local maven repo.
//
// e.g: org/apache/maven/plugin/maven-plugin-plugin -> (org.apache.maven.plugin,
// maven-plugin-plugin)
//
// root is like a prefix of the path which is not part of the maven GAV
func parseGA(fullGAPath, root string) [2]string {
	gaPath := trimRoot(fullGAPath, root)

	items := strings.Split(gaPath, "/")
	artifact := items[len(items)-1]
	group := strings.Join(items[:len(items)-1], ".")

	return [2]string{group, artifact}
}

// Parse maven groupId, artifactId and version from a standard path in a local maven repo.
//
// e.g: org/apache/maven/plugin/maven-plugin-plugin/1.0.0/maven-plugin-plugin-1.0.0.pom
// -> (org.apache.maven.plugin, maven-plugin-plugin, 1.0.0)
//
// root is like a prefix of the path which is not part of the maven GAV
func parseGAV(fullArtifactPath, root string) [3]string {
	verPath := trimRoot(fullArtifactPath, root)

	items := strings.Split(verPath, "/")
	version := items[len(items)-2]
	artifact := items[len(items)-3]
	group := strings.Join(items[:len(items)-3], ".")

	return [3]string{group, artifact, version}
}

// Give a list of paths with pom files and parse the maven groupId, artifactId and version
// from them. The result will be a dict like {groupId: {artifactId: [versions list]}}.
// Root is like a prefix of the path which is not part of the maven GAV
func parseGAVs(pomPaths []string, root string) map[string]map[string][]string {
	gavs := make(map[string]map[string][]string)
	for _, pom := range pomPaths {
		gav := parseGAV(pom, root)
		g := gav[0]
		a := gav[1]
		v := gav[2]
		avs := make(map[string][]string)
		if item, ok := gavs[g]; ok {
			avs = item
		}
		vers := []string{}
		if item, ok := avs[a]; ok {
			vers = item
		}
		vers = append(vers, v)
		avs[a] = vers
		gavs[g] = avs
	}
	return gavs
}

func trimRoot(fullPath, root string) string {
	slashRoot := strings.TrimSpace(root)
	if slashRoot == "" {
		slashRoot = "/"
	}
	if !strings.HasSuffix(slashRoot, "/") {
		slashRoot += "/"
	}

	verPath := strings.TrimPrefix(fullPath, slashRoot)
	verPath = strings.TrimSuffix(verPath, "/")
	return verPath
}
