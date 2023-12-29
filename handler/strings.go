package handler

import (
	"regexp"
	"strings"
)

var (
	archRe     = regexp.MustCompile(`(arm64|arm|386|amd64|x86_64|aarch64|i686)`)
	fileExtRe  = regexp.MustCompile(`(\.[a-z][a-z0-9]+)+$`)
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
	//arch modifications
	if a == "x86_64" || a == "" {
		a = "amd64"
	} else if a == "i686" {
		a = "386"
	} else if a == "aarch64" {
		a = "arm64"
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
