package pkgs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// func TestMavenMetadata(t *testing.T) {
// 	meta := MavenMetadata{
// 		GroupId:        "foo.bar",
// 		ArtifactId:     "foobar",
// 		versions:       []string{"1.0", "2.0", "3.0"},
// 		LastUpdateTime: time.Now().Format("2006-01-02 15:04:01"),
// 	}
// 	content, err := meta.GenerateMetaFileContent()
// 	assert.Nil(t, err)
// 	fmt.Println(content)
// }

func TestVersionsCompare(t *testing.T) {
	// Normal versions comparasion
	assert.Equal(t, -1, versionCompare("1.0.0", "1.0.1"))
	assert.Equal(t, 1, versionCompare("1.10.0", "1.9.1"))
	assert.Equal(t, 0, versionCompare("1.0.1", "1.0.1"))
	assert.Equal(t, 1, versionCompare("2.0.1", "1.0.1"))

	// # Special versions comparasion
	assert.Equal(t, 1, versionCompare("1.0.1-alpha", "1.0.1"))
	assert.Equal(t, 1, versionCompare("1.0.1-beta", "1.0.1-alpha"))
	assert.Equal(t, 1, versionCompare("1.0.2", "1.0.1-alpha"))
	assert.Equal(t, 1, versionCompare("1.0.1", "1.0-m2"))
	assert.Equal(t, 1, versionCompare("1.0.2-alpha", "1.0.1-m2"))
	assert.Equal(t, 1, versionCompare("1.0.2-alpha", "1.0.1-alpha"))
}
