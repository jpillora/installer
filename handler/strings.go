package handler

import (
	"regexp"
	"strings"
)

func getOS(s string) string {
	s = strings.ToLower(s)
	posixOSRe := regexp.MustCompile(`(darwin|linux|(net|free|open)bsd|mac|osx|windows|win)`)
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
	var (
		// '_' in 'linux_x32' will not match '\b', so the '\b' can only match the end of string
		archReAmd64    = regexp.MustCompile(`(amd64|x86_64)\b`)
		archRe386      = regexp.MustCompile(`(386|686)\b`)
		archReArm64    = regexp.MustCompile(`(arm64|aarch64)\b`)
		archReArm      = regexp.MustCompile(`(arm(v[567])?)\b`)
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
