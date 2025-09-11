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
		osReDarwin    = regexp.MustCompile(`(?:[^a-zA-Z0-9]|^)(darwin|mac|osx)`)
		osReDragonfly = regexp.MustCompile(`(?:[^a-zA-Z0-9]|^)(dragonfly)`)
		osReWindows   = regexp.MustCompile(`(?:[^a-zA-Z0-9]|^)(win)`)
		// It is only necessary to match both the beginning and end of the substring,
		// if the regexp is meant to match the whole string.
		osReMisc = regexp.MustCompile(`(?:[^a-zA-Z0-9]|^)` +
			`(aix|android|illumos|ios|linux|(free|net|open)bsd|plan9|solaris)` +
			`(?:[^a-zA-Z0-9]|$)`)
	)

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
	var (
		// for architecture detection, it is prefered to do a suffix match,
		// so that example_i686.tar.gz can also be matched.
		archReAmd64   = regexp.MustCompile(`(amd64|x86_64)(?:[^a-zA-Z0-9]|$)`)
		archRe386     = regexp.MustCompile(`(386|686)(?:[^a-zA-Z0-9]|$)`)
		archReArm64   = regexp.MustCompile(`(arm64|aarch64)(?:[^a-zA-Z0-9]|$)`)
		archReArm     = regexp.MustCompile(`(arm(v[567])?[eh]?[fl]?)(?:[^a-zA-Z0-9]|$)`)
		archReLoong64 = regexp.MustCompile(`(loong64|loongarch64)(?:[^a-zA-Z0-9]|$)`)
		archRePPC64   = regexp.MustCompile(`(ppc64|powerpc64)(?:[^a-zA-Z0-9]|$)`)
		archRePPC64LE = regexp.MustCompile(`(ppc64le|powerpc64le)(?:[^a-zA-Z0-9]|$)`)
		archReRiscv64 = regexp.MustCompile(`(riscv64)`) // also match riscv64gc
		archReMisc    = regexp.MustCompile(`(?:[^a-zA-Z0-9]|^)` +
			`(mips|mips64|mips64le|mipsle|s390x|wasm)` +
			`(?:[^a-zA-Z0-9]|$)`)
	)

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
	case regexp.MustCompile(`(x?64(bit)?)\b`).
		MatchString(s):
		return "amd64"
	// fuzz match 'x?32(bit)?'
	case regexp.MustCompile(`(x?32(bit)?|x86)\b`).
		MatchString(s):
		return "386"
	default:
		return ""
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
