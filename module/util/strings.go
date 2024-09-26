package util

import "strings"

// A similar method to StringUtils.isBlank(str) in java
// apache commons-lang3. Means
//   - IsBlankString("")        = true
//   - IsBlankString(" ")       = true
//   - IsBlankString("bob")     = false
//   - IsBlankString("  bob  ") = false
func IsBlankString(str string) bool {
	return len(strings.TrimSpace(str)) == 0
}
