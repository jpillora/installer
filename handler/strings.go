package handler

import (
	"regexp"
	"strings"
)

func getOS(s string) string {
	var (
		// '_' in 'linux_x32' will not match '\b', so the '\b' can only match the start of string
		oSReDarwin  = regexp.MustCompile(`\b(darwin|mac|osx)`)
		osReWindows = regexp.MustCompile(`\b(win|windows)`)
		unixOSRe    = regexp.MustCompile(`\b(linux|(net|free|open)bsd)`)
	)

	s = strings.ToLower(s)
	switch {
	case oSReDarwin.MatchString(s):
		return "darwin"
	case osReWindows.MatchString(s):
		return "windows"
	case unixOSRe.MatchString(s):
		return unixOSRe.FindString(s)
	// in case of no match, default to linux
	default:
		return "linux"
	}
}

func getArch(s string) string {
	var (
		// '_' in 'linux_x32' will not match '\b', so the '\b' can only match the end of string
		archReAmd64    = regexp.MustCompile(`(amd64|x86_64)\b`)
		archRe386      = regexp.MustCompile(`(386|686)\b`)
		archReArm64    = regexp.MustCompile(`(arm64|aarch64)\b`)
		archReArm      = regexp.MustCompile(`(arm(v[567])?[eh]?[fl]?)\b`)
		archReMips     = regexp.MustCompile(`(mips)\b`)
		archReMips64   = regexp.MustCompile(`(mips64)\b`)
		archReMips64Le = regexp.MustCompile(`(mips64le)\b`)
		archReMipsLe   = regexp.MustCompile(`(mipsle)\b`)
		archRePpc64    = regexp.MustCompile(`(ppc64)\b`)
		archRePpc64Le  = regexp.MustCompile(`(ppc64le)\b`)
		archReRiscv64  = regexp.MustCompile(`(riscv64)\b`)
		archReS390x    = regexp.MustCompile(`(s390x)\b`)
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
	case archReMips.MatchString(s):
		return "mips"
	case archReMips64.MatchString(s):
		return "mips64"
	case archReMips64Le.MatchString(s):
		return "mips64le"
	case archReMipsLe.MatchString(s):
		return "mipsle"
	case archRePpc64.MatchString(s):
		return "ppc64"
	case archRePpc64Le.MatchString(s):
		return "ppc64le"
	case archReRiscv64.MatchString(s):
		return "riscv64"
	case archReS390x.MatchString(s):
		return "s390x"

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
