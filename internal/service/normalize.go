package service

import "strings"

// Normalize lowercases and trims whitespace from a raw ingredient name.
func Normalize(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}
