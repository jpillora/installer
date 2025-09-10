package handler

import (
	"regexp"
	"strings"
)

var (
	// IMPORTANT: arch regex captures must be ordered from most specific to least specific
	archRe     = regexp.MustCompile(`(arm64|armv7|armv6|amd64|x86_64|aarch64|386|686|\barm\b|\b32\b|\b64\b)`)
	fileExtRe  = regexp.MustCompile(`(\.tar)?(\.[a-z][a-z0-9]+)$`)
	posixOSRe  = regexp.MustCompile(`(darwin|linux|(net|free|open)bsd|mac|osx|windows|win)`)
	checksumRe = regexp.MustCompile(`(checksums|sha256sums)`)
)

func getOS(s string) string {
	s = strings.ToLower(s)
	o := posixOSRe.FindString(s)
	if o == "mac" || o == "osx" {
		o = "darwin"
	}
	if o == "win" {
		o = "windows"
	}
	return o
}

func getArch(s string) string {
	s = strings.ToLower(s)
	a := archRe.FindString(s)
	// arch modifications
	switch a {
	case "64", "x86_64", "":
		a = "amd64" // default
	case "32", "686":
		a = "386"
	case "aarch64":
		a = "arm64"
	case "armv6", "armv7":
		a = "arm"
	}
	return a
}

func getFileExt(s string) string {
	return fileExtRe.FindString(s)
}

func splitHalf(s, by string) (string, string) {
	i := strings.Index(s, by)
	if i == -1 {
		return s, ""
	}
	return s[:i], s[i+len(by):]
}
