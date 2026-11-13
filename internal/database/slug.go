package database

import (
	"regexp"
	"strings"
)

var slugSeparator = regexp.MustCompile(`[^\p{L}\p{N}]+`)

func makeSlug(parts ...string) string {
	value := strings.ToLower(strings.Join(parts, "-"))
	value = slugSeparator.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}
