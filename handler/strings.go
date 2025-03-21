package handler

import (
	"regexp"
	"strings"
)

func getOS(s string) string {
	var (
		// '\b' ([^a-zA-Z0-9_]) is not ideal for matching the boundary
		// for example: gitleaks_8.24.0_darwin_x64.tar.gz
		// Since the RE2 does not support lookaheads & lookbehinds, we use the following workaround:
		// (?:[^a-zA-Z0-9]|^) to match the beginning of the substring,
		// and (?:[^a-zA-Z0-9]|$) to match the end of the substring.
		// Then we use the regexp.FindStringSubmatch to extract the first capturing group.

		// for OS detection, it is prefered to do a prefix match,
		// so that example_macos_x64.tar.gz can also be matched.
		oSReDarwin  = regexp.MustCompile(`(?:[^a-zA-Z0-9]|^)(darwin|mac|osx)`)
		osReWindows = regexp.MustCompile(`(?:[^a-zA-Z0-9]|^)(win|windows)`)
		// It is only necessary to match both the beginning and end of the substring,
		// if the regexp is meant to match the whole string.
		unixOSRe = regexp.MustCompile(`(?:[^a-zA-Z0-9]|^)(linux|(net|free|open)bsd)(?:[^a-zA-Z0-9]|$)`)
	)

	s = strings.ToLower(s)
	switch {
	case oSReDarwin.MatchString(s):
		return "darwin"
	case osReWindows.MatchString(s):
		return "windows"
	case unixOSRe.MatchString(s):
		// return the first capturing group (contains only the alphanumeric characters)
		return unixOSRe.FindStringSubmatch(s)[1]
	// in case of no match, default to linux
	default:
		return "linux"
	}
}

func getArch(s string) string {
	var (
		// for architecture detection, it is prefered to do a suffix match,
		// so that example_i686.tar.gz can also be matched.
		archReAmd64 = regexp.MustCompile(`(amd64|x86_64)(?:[^a-zA-Z0-9]|$)`)
		archRe386   = regexp.MustCompile(`(386|686)(?:[^a-zA-Z0-9]|$)`)
		archReArm64 = regexp.MustCompile(`(arm64|aarch64)(?:[^a-zA-Z0-9]|$)`)
		archReArm   = regexp.MustCompile(`(arm(v[567])?[eh]?[fl]?)(?:[^a-zA-Z0-9]|$)`)
		archReMisc  = regexp.MustCompile(`(?:[^a-zA-Z0-9]|^)(mips|mips64|mips64le|mipsle|ppc64|ppc64le|riscv64|s390x)(?:[^a-zA-Z0-9]|$)`)
	)

	s = strings.ToLower(s)
	switch {
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

	// fuzz match 'x?32(bit)?'
	case regexp.MustCompile(`(x?32(bit)?)\b`).
		MatchString(s):
		return "386"
	// in case of no match, default to amd64
	// fuzz match 'x?64(bit)?'
	default:
		return "amd64"
	}
}

func getFileExt(s string) string {
	fileExtRe := regexp.MustCompile(`(\.tar)?(\.[a-z][a-z0-9]+)$`)
	return fileExtRe.FindString(s)
}

func splitHalf(s, by string) (string, string) {
	i := strings.Index(s, by)
	if i == -1 {
		return s, ""
	}
	return s[:i], s[i+len(by):]
}
