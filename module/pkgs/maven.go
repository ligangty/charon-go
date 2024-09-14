package pkgs

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
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
