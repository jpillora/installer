package handler

import (
	"strings"
)

func getOS(s string) string {
	s = strings.ToLower(s)
	switch {
	case osReDarwin.MatchString(s):
		return "darwin"
	case osReDragonfly.MatchString(s):
		return "dragonfly"
	case osReWindows.MatchString(s):
		return "windows"
	case osReMisc.MatchString(s):
		// return the first capturing group (contains only the alphanumeric characters)
		return osReMisc.FindStringSubmatch(s)[1]
	default:
		return ""
	}
}

func getArch(s string) string {
	s = strings.ToLower(s)
	switch {
	case archReLoong64.MatchString(s):
		return "loong64"
	case archRePPC64.MatchString(s):
		return "ppc64"
	case archRePPC64LE.MatchString(s):
		return "ppc64le"
	case archReRiscv64.MatchString(s):
		return "riscv64"
	case archReArm64.MatchString(s):
		return "arm64"
	case archReAmd64.MatchString(s):
		return "amd64"
	case archReArm.MatchString(s):
		return "arm"
	case archRe386.MatchString(s):
		return "386"
	case archReMisc.MatchString(s):
		return archReMisc.FindStringSubmatch(s)[1]

	// fuzz match 'x?64(bit)?'
	case fuzzArchAmd64.MatchString(s):
		return "amd64"
	// fuzz match 'x?32(bit)?'
	case fuzzArch386.MatchString(s):
		return "386"
	default:
		return ""
	}
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
