// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

- package deps_test
+ package deps

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// -----------------------------
// normalizeArch
// -----------------------------

func TestNormalizeArch(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"amd64", archAMD64},
		{"x64", archAMD64},
		{"x86_64", archAMD64},
		{"X64", archAMD64},
		{"ia32", "386"},
		{"x86", "386"},
		{"386", "386"},
		{"arm64", archARM64},
		{"aarch64", archARM64},
		{"arm", archARM},
		{"ARMV7", archARM},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		if got := normalizeArch(tt.in); got != tt.want {
			t.Fatalf("normalizeArch(%q) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

// -----------------------------
// analyzePackagePlatform
// -----------------------------

func TestAnalyzePackagePlatform(t *testing.T) {
	resolver := &NpmResolver{}

	tests := []struct {
		name string
		pkg  string
		os   string
		arch string
	}{
		{
			name: "linux arm64 with libc",
			pkg:  "@parcel/watcher-linux-arm64-glibc",
			os:   "linux",
			arch: archARM64,
		},
		{
			name: "darwin arm64",
			pkg:  "pkg-darwin-arm64",
			os:   "darwin",
			arch: archARM64,
		},
		{
			name: "incomplete platform info",
			pkg:  "foo-linux",
		},
		{
			name: "abnormal platform format",
			pkg:  "foo-linux-unknown-extra",
		},
		{
			name: "non platform package",
			pkg:  "lodash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osName, arch := resolver.analyzePackagePlatform(tt.pkg)
			if osName != tt.os || arch != tt.arch {
				t.Fatalf(
					"analyzePackagePlatform(%q) = (%q,%q); want (%q,%q)",
					tt.pkg, osName, arch, tt.os, tt.arch,
				)
			}
		})
	}
}

// -----------------------------
// isForCurrentPlatform
// -----------------------------

func platformPkgForRuntime() string {
	goos := runtime.GOOS
	goarch := normalizeArch(runtime.GOARCH)

	switch goos {
	case "linux":
		if goarch == archARM64 {
			return "pkg-linux-arm64"
		}
		return "pkg-linux-x64"
	case "darwin":
		if goarch == archARM64 {
			return "pkg-darwin-arm64"
		}
		return "pkg-darwin-x64"
	case "windows":
		return "pkg-win32-x64"
	default:
		return "pkg-neutral"
	}
}

func TestIsForCurrentPlatform(t *testing.T) {
	resolver := &NpmResolver{}

	match := platformPkgForRuntime()
	if !resolver.isForCurrentPlatform(match) {
		t.Fatalf("expected %q to match current platform", match)
	}

	var other string
	switch runtime.GOOS {
	case "linux":
		other = "pkg-darwin-arm64"
	case "darwin":
		other = "pkg-win32-x64"
	default:
		other = "pkg-linux-x64"
	}

	if resolver.isForCurrentPlatform(other) {
		t.Fatalf("expected %q NOT to match current platform", other)
	}
}

// -----------------------------
// GetInstalledPkgs
// -----------------------------

type testResolver struct {
	NpmResolver
	buffer io.Reader
}

func (r *testResolver) ListPkgPaths() (io.Reader, error) {
	return r.buffer, nil
}

func TestGetInstalledPkgs_NonCrossPlatform(t *testing.T) {
	tmp := t.TempDir()
	pkgDir := filepath.Join(tmp, "node_modules")
	_ = os.MkdirAll(pkgDir, 0o755)

	paths := []string{
		filepath.Join(pkgDir, "lodash"),
		filepath.Join(pkgDir, "express"),
	}

	buf := &bytes.Buffer{}
	for _, p := range paths {
		_ = os.MkdirAll(p, 0o755)
		buf.WriteString(p + "\n")
	}

	resolver := &testResolver{buffer: buf}
	pkgs := resolver.GetInstalledPkgs(pkgDir)

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 pkgs, got %d", len(pkgs))
	}
}

func TestGetInstalledPkgs_EmptyOutput(t *testing.T) {
	tmp := t.TempDir()
	pkgDir := filepath.Join(tmp, "node_modules")
	_ = os.MkdirAll(pkgDir, 0o755)

	resolver := &testResolver{buffer: bytes.NewBuffer(nil)}
	pkgs := resolver.GetInstalledPkgs(pkgDir)

	if len(pkgs) != 0 {
		t.Fatalf("expected 0 pkgs, got %d", len(pkgs))
	}
}

// -----------------------------
// ResolvePackageLicense
// -----------------------------

func TestResolvePackageLicense_CrossPlatformSkip(t *testing.T) {
	resolver := &NpmResolver{}
	cfg := &ConfigDeps{}

	var other string
	switch runtime.GOOS {
	case "linux":
		other = "pkg-darwin-arm64"
	case "darwin":
		other = "pkg-win32-x64"
	default:
		other = "pkg-linux-x64"
	}

	result := resolver.ResolvePackageLicense(other, "/fake/path", cfg)
	if !result.IsCrossPlatform {
		t.Fatalf("expected IsCrossPlatform=true for %q", other)
	}
}
