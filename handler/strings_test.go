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
		// m"arm"ite
		{"marmite-0.2.5-x86_64-unknown-linux-musl.tar.gz", "linux", "amd64"},
	} {
		os := getOS(tc.name)
		arch := getArch(tc.name)
		if os != tc.os || arch != tc.arch {
			t.Fatalf("file '%s' results in %s/%s, expected %s/%s", tc.name, os, arch, tc.os, tc.arch)
		}
	}
}
