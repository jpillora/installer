package handler

import "testing"

func TestFilExt(t *testing.T) {
	tests := []struct {
		file, ext string
	}{
		{"my.file.tar.gz", ".tar.gz"},
		{"my.file.tar.bz2", ".tar.bz2"},
		{"my.file.tar.bz", ".tar.bz"},
		{"my.file.bz2", ".bz2"},
		{"my.file.gz", ".gz"},
		{"my.file.tar.zip", ".tar.zip"}, // :(
	}
	for _, tc := range tests {
		ext := getFileExt(tc.file)
		if ext != tc.ext {
			t.Fatalf("getFileExt(%s) = %s, want %s", tc.file, ext, tc.ext)
		}
	}
}

func TestOSArch(t *testing.T) {
	for _, tc := range []struct {
		name, os, arch string
	}{
		// architecture implicated by OS name
		{"piknik-win32-0.10.2.zip", "windows", "386"},
		{"piknik-win64-0.10.2.zip", "windows", "amd64"},
		// armv7l
		{"yt-dlp_linux_armv7l", "linux", "arm"},
		// armv7
		{"ouch-armv7-unknown-linux-musleabihf.tar.gz", "linux", "arm"},
		// armv6
		{"gitleaks_8.24.0_linux_armv6.tar.gz", "linux", "arm"},
		{"gitleaks_8.24.0_windows_armv6.zip", "windows", "arm"},
		// armv5
		{"gg-linux-armv5", "linux", "arm"},
		// arm
		{"gitui-linux-arm.tar.gz", "linux", "arm"},
		{"croc_v10.2.1_Linux-ARM.tar.gz", "linux", "arm"},
		{"croc_v10.2.1_Windows-ARM.zip", "windows", "arm"},
		{"piknik-linux_arm-0.10.2.tar.gz", "linux", "arm"},
		// x64
		{"gitleaks_8.24.0_linux_x64.tar.gz", "linux", "amd64"},
		{"gitleaks_8.24.0_darwin_x64.tar.gz", "darwin", "amd64"},
		{"gitleaks_8.24.0_windows_x64.zip", "windows", "amd64"},
		// x32
		{"gitleaks_8.24.0_linux_x32.tar.gz", "linux", "386"},
		{"gitleaks_8.24.0_windows_x32.zip", "windows", "386"},
		// 64bit
		{"croc_v10.2.1_Linux-64bit.tar.gz", "linux", "amd64"},
		{"croc_v10.2.1_macOS-64bit.tar.gz", "darwin", "amd64"},
		{"croc_v10.2.1_Windows-64bit.zip", "windows", "amd64"},
		{"uv-x86_64-unknown-linux-musl.tar.gz", "linux", "amd64"},
		{"uv-x86_64-apple-darwin.tar.gz", "darwin", "amd64"},
		{"uv-x86_64-pc-windows-msvc.zip", "windows", "amd64"},
		// 32bit
		{"croc_v10.2.1_Linux-32bit.tar.gz", "linux", "386"},
		{"croc_v10.2.1_Windows-32bit.zip", "windows", "386"},
		// x86
		{"gitui-mac-x86.tar.gz", "darwin", "386"},
		// Archs besides x86, x64, arm and amd64
		{"crun-1.20-linux-ppc64le", "linux", "ppc64le"},
		{"crun-1.20-linux-riscv64", "linux", "riscv64"},
		{"crun-1.20-linux-s390x", "linux", "s390x"},
		{"uv-loongarch64-unknown-linux-gnu.tar.gz", "linux", "loong64"},
		{"uv-powerpc64-unknown-linux-gnu.tar.gz", "linux", "ppc64"},
		{"uv-powerpc64le-unknown-linux-gnu.tar.gz", "linux", "ppc64le"},
		{"uv-riscv64gc-unknown-linux-gnu.tar.gz", "linux", "riscv64"},
		{"uv-s390x-unknown-linux-gnu.tar.gz", "linux", "s390x"},

		// OSes besides linux, windows and macos
		{"croc_v10.2.1_DragonFlyBSD-64bit.tar.gz", "dragonfly", "amd64"},
		{"croc_v10.2.1_FreeBSD-64bit.tar.gz", "freebsd", "amd64"},
		{"croc_v10.2.1_FreeBSD-ARM64.tar.gz", "freebsd", "arm64"},
		{"croc_v10.2.1_NetBSD-32bit.tar.gz", "netbsd", "386"},
		{"croc_v10.2.1_NetBSD-64bit.tar.gz", "netbsd", "amd64"},
		{"croc_v10.2.1_NetBSD-ARM64.tar.gz", "netbsd", "arm64"},
		{"croc_v10.2.1_OpenBSD-64bit.tar.gz", "openbsd", "amd64"},
		{"croc_v10.2.1_OpenBSD-ARM64.tar.gz", "openbsd", "arm64"},
		{"piknik-dragonflybsd_amd64-0.10.2.tar.gz", "dragonfly", "amd64"},
		{"piknik-freebsd_amd64-0.10.2.tar.gz", "freebsd", "amd64"},
		{"piknik-freebsd_i386-0.10.2.tar.gz", "freebsd", "386"},
		{"piknik-netbsd_amd64-0.10.2.tar.gz", "netbsd", "amd64"},
		{"piknik-netbsd_i386-0.10.2.tar.gz", "netbsd", "386"},
		{"piknik-openbsd_amd64-0.10.2.tar.gz", "openbsd", "amd64"},
		{"piknik-openbsd_i386-0.10.2.tar.gz", "openbsd", "386"},

		// misc
		{"marmite-0.2.5-x86_64-unknown-linux-musl.tar.gz", "linux", "amd64"},
		{"uv-armv7-unknown-linux-musleabihf.tar.gz", "linux", "arm"},
		{"uv-i686-unknown-linux-musl.tar.gz", "linux", "386"},
		{"uv-aarch64-unknown-linux-musl.tar.gz", "linux", "arm64"},
		{"uv-aarch64-apple-darwin.tar.gz", "darwin", "arm64"},
		{"uv-aarch64-pc-windows-msvc.zip", "windows", "arm64"},
		{"uv-i686-pc-windows-msvc.zip", "windows", "386"},
		{"yt-dlp_linux_aarch64", "linux", "arm64"},
		{"gitleaks_8.24.0_linux_arm64.tar.gz", "linux", "arm64"},
		{"gitleaks_8.24.0_darwin_arm64.tar.gz", "darwin", "arm64"},
		{"gitui-linux-x86_64.tar.gz", "linux", "amd64"},
		{"gitui-linux-aarch64.tar.gz", "linux", "arm64"},
		{"gg-linux-x86_64", "linux", "amd64"},
		{"gg-linux-arm64", "linux", "arm64"},
		{"croc_v10.2.1_Linux-ARM64.tar.gz", "linux", "arm64"},
		{"croc_v10.2.1_macOS-ARM64.tar.gz", "darwin", "arm64"},
		{"croc_v10.2.1_Windows-ARM64.zip", "windows", "arm64"},
		{"ouch-x86_64-unknown-linux-musl.tar.gz", "linux", "amd64"},
		{"ouch-aarch64-unknown-linux-musl.tar.gz", "linux", "arm64"},
		{"piknik-linux_i386-0.10.2.tar.gz", "linux", "386"},
		{"piknik-linux_x86_64-0.10.2.tar.gz", "linux", "amd64"},
		{"piknik-win64-arm64-0.10.2.zip", "windows", "arm64"},

		// no os
		{"libtree_aarch64", "", "arm64"},
		{"libtree_armv6l", "", "arm"},
		{"libtree_i686", "", "386"},
		{"libtree_x86_64", "", "amd64"},
		// archs could be misreaded as extensions
		{"runc.ppc64le", "", "ppc64le"},
		{"runc.riscv64", "", "riscv64"},
		{"runc.s390x", "", "s390x"},
		{"runc.armel", "", "arm"},
		{"runc.armhf", "", "arm"},

		// no arch
		{"yt-dlp_linux", "linux", ""},
		// no arch, no os
		{"codex-npm-0.31.0.tgz", "", ""},

		// architecture indicator not supported
		// {"piknik-macos-0.10.2.tar.gz", "darwin", "arm64"},
		// {"piknik-macos-intel-0.10.2.tar.gz", "darwin", "amd64"},
	} {
		os := getOS(tc.name)
		arch := getArch(tc.name)
		if os != tc.os || arch != tc.arch {
			t.Fatalf("file '%s' results in %s/%s, expected %s/%s", tc.name, os, arch, tc.os, tc.arch)
		}
	}
}

func TestQueryCacheKey(t *testing.T) {
	q := Query{
		User:    "testuser",
		Program: "testrepo",
		Release: "v1.0.0",
		OS:      "linux",
		Arch:    "amd64",
	}

	key1 := q.cacheKey()
	key2 := q.cacheKey()

	if key1 != key2 {
		t.Errorf("cacheKey should be deterministic, got %s and %s", key1, key2)
	}

	// Different query should produce different key
	q2 := q
	q2.Arch = "arm64"
	key3 := q2.cacheKey()
	if key1 == key3 {
		t.Error("different queries should produce different cache keys")
	}
}

func TestAssetMethods(t *testing.T) {
	tests := []struct {
		asset   Asset
		key     string
		is32Bit bool
		isMac   bool
		isMacM1 bool
	}{
		{
			asset:   Asset{OS: "linux", Arch: "amd64"},
			key:     "linux/amd64",
			is32Bit: false,
			isMac:   false,
			isMacM1: false,
		},
		{
			asset:   Asset{OS: "linux", Arch: "386"},
			key:     "linux/386",
			is32Bit: true,
			isMac:   false,
			isMacM1: false,
		},
		{
			asset:   Asset{OS: "darwin", Arch: "arm64"},
			key:     "darwin/arm64",
			is32Bit: false,
			isMac:   true,
			isMacM1: true,
		},
		{
			asset:   Asset{OS: "darwin", Arch: "amd64"},
			key:     "darwin/amd64",
			is32Bit: false,
			isMac:   true,
			isMacM1: false,
		},
	}

	for _, tt := range tests {
		if got := tt.asset.Key(); got != tt.key {
			t.Errorf("Asset.Key() = %v, want %v", got, tt.key)
		}
		if got := tt.asset.Is32Bit(); got != tt.is32Bit {
			t.Errorf("Asset.Is32Bit() = %v, want %v", got, tt.is32Bit)
		}
		if got := tt.asset.IsMac(); got != tt.isMac {
			t.Errorf("Asset.IsMac() = %v, want %v", got, tt.isMac)
		}
		if got := tt.asset.IsMacM1(); got != tt.isMacM1 {
			t.Errorf("Asset.IsMacM1() = %v, want %v", got, tt.isMacM1)
		}
	}
}

func TestAssetsHasM1(t *testing.T) {
	tests := []struct {
		name   string
		assets Assets
		want   bool
	}{
		{
			name:   "no assets",
			assets: Assets{},
			want:   false,
		},
		{
			name: "no M1 assets",
			assets: Assets{
				{OS: "linux", Arch: "amd64"},
				{OS: "darwin", Arch: "amd64"},
			},
			want: false,
		},
		{
			name: "has M1 asset",
			assets: Assets{
				{OS: "linux", Arch: "amd64"},
				{OS: "darwin", Arch: "arm64"},
				{OS: "darwin", Arch: "amd64"},
			},
			want: true,
		},
		{
			name:   "only M1 asset",
			assets: Assets{{OS: "darwin", Arch: "arm64"}},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.assets.HasM1(); got != tt.want {
				t.Errorf("Assets.HasM1() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitHalf(t *testing.T) {
	tests := []struct {
		input  string
		sep    string
		first  string
		second string
	}{
		{"user/repo", "/", "user", "repo"},
		{"only", "/", "only", ""},
		{"", "/", "", ""},
		{"a/b/c", "/", "a", "b/c"},
		{"user@repo", "@", "user", "repo"},
		{"repo@v1.0", "@", "repo", "v1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			first, second := splitHalf(tt.input, tt.sep)
			if first != tt.first {
				t.Errorf("first = %v, want %v", first, tt.first)
			}
			if second != tt.second {
				t.Errorf("second = %v, want %v", second, tt.second)
			}
		})
	}
}
