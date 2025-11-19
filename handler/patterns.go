package handler

import (
	"regexp"
)

// os patterns
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

// architecture patterns
var (
	// for architecture detection, it is prefered to do a suffix match,
	// so that example_i686.tar.gz can also be matched.

	archReAmd64   = regexp.MustCompile(`(amd64|x86_64)(?:[^a-zA-Z0-9]|$)`)
	archRe386     = regexp.MustCompile(`(386|686|x86_32)(?:[^a-zA-Z0-9]|$)`)
	archReArm64   = regexp.MustCompile(`(arm64|aarch64|aarch_64)(?:[^a-zA-Z0-9]|$)`)
	archReArm     = regexp.MustCompile(`(arm(v[567]|32)?[eh]?[fl]?)(?:[^a-zA-Z0-9]|$)`)
	archReLoong64 = regexp.MustCompile(`(loong64|loongarch64)(?:[^a-zA-Z0-9]|$)`)
	archRePPC64   = regexp.MustCompile(`(ppc64|powerpc64)(?:[^a-zA-Z0-9]|$)`)
	archRePPC64LE = regexp.MustCompile(`(ppc64le|powerpc64le|ppcle_64)(?:[^a-zA-Z0-9]|$)`)
	archReRiscv64 = regexp.MustCompile(`(riscv64)`) // also match riscv64gc
	archReMisc    = regexp.MustCompile(`(?:[^a-zA-Z0-9]|^)` +
		`(mips|mips64|mips64le|mipsle|s390x|s390_64|wasm)` +
		`(?:[^a-zA-Z0-9]|$)`)

	fuzzArchAmd64 = regexp.MustCompile(`(x?64(bit)?)\b`)
	fuzzArch386   = regexp.MustCompile(`(x?32(bit)?|x86)\b`)
)

var (
	checksumRe     = regexp.MustCompile(`(checksums|sha256sums)`)
	fileExtRe      = regexp.MustCompile(`(\.tar)?(\.[a-z][a-z0-9]+)$`)
	searchGithubRe = regexp.MustCompile(`https:\/\/github\.com\/(\w+)\/(\w+)`)
)
