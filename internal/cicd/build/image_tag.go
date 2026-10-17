package build

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var validImageTag = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)

func validateCustomImageTag(tag string) error {
	if !validImageTag.MatchString(tag) {
		return fmt.Errorf("image tag must match %s", validImageTag.String())
	}
	return nil
}

func automaticImageTag(branch string, buildNumber int, now time.Time) string {
	var component strings.Builder
	for _, char := range branch {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z', char >= '0' && char <= '9', char == '_', char == '.', char == '-':
			component.WriteRune(char)
		default:
			component.WriteByte('-')
		}
	}
	prefix := strings.Trim(component.String(), ".-")
	if prefix == "" {
		prefix = "build"
	}
	suffix := fmt.Sprintf("-%d-%s", buildNumber, now.Format("20060102150405"))
	maxPrefix := 128 - len(suffix)
	if len(prefix) > maxPrefix {
		prefix = strings.TrimRight(prefix[:maxPrefix], ".-")
	}
	return prefix + suffix
}
