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

package deps


import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

//
// TC-001 / TC-002
// Unit test: architecture normalization
//
func TestNormalizeArch(t *testing.T) {
	cases := map[string]string{
		"amd64":   archAMD64,
		"x64":     archAMD64,
		"x86_64":  archAMD64,
		"X64":     archAMD64,
		"ia32":    "386",
		"x86":     "386",
		"386":     "386",
		"arm64":   archARM64,
		"aarch64": archARM64,
		"arm":     archARM,
		"ARMV7":   archARM,
		"unknown": "unknown",
	}

	for in, want := range cases {
		if got := normalizeArch(in); got != want {
			t.Fatalf("normalizeArch(%q) = %q; want %q", in, got, want)
		}
	}
}

//
// TC-006 / TC-007 / TC-008
// Unit + boundary tests for package platform parsing
//
func TestAnalyzePackagePlatform(t *testing.T) {
	resolver := &NpmResolver{}

	tests := []struct {
		name     string
		pkgName  string
		wantOS   string
		wantArch string
	}{
		{
			name:     "full linux arm64 package",
			pkgName:  "@parcel/watcher-linux-arm64-glibc",
			wantOS:   "linux",
			wantArch: archARM64,
		},
		{
			name:     "full darwin arm64 package",
			pkgName:  "pkg-darwin-arm64",
			wantOS:   "darwin",
			wantArch: archARM64,
		},
		{
			name:     "TC-007 incomplete platform info",
			pkgName:  "foo-linux",
			wantOS:   "",
			wantArch: "",
		},
		{
			name:     "TC-008 abnormal platform format",
			pkgName:  "foo-linux-unknown-extra",
			wantOS:   "",
			wantArch: "",
		},
		{
			name:     "regular non-platform package",
			pkgName:  "lodash",
			wantOS:   "",
			wantArch: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			osName, arch := resolver.analyzePackagePlatform(tc.pkgName)
			if osName != tc.wantOS || arch != tc.wantArch {
				t.Fatalf(
					"analyzePackagePlatform(%q) = (%q,%q); want (%q,%q)",
					tc.pkgName, osName, arch, tc.wantOS, tc.wantArch,
				)
			}
		})
	}
}

//
// helper: build a package name matching current runtime
//
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

//
// TC-009
// Unit test: isForCurrentPlatform logic
//
func TestIsForCurrentPlatform(t *testing.T) {
	resolver := &NpmResolver{}

	matchPkg := platformPkgForRuntime()
	if !resolver.isForCurrentPlatform(matchPkg) {
		t.Fatalf("expected %q to match current platform", matchPkg)
	}

	var otherPkg string
	switch runtime.GOOS {
	case "linux":
		otherPkg = "pkg-darwin-arm64"
	case "darwin":
		otherPkg = "pkg-win32-x64"
	default:
		otherPkg = "pkg-linux-x64"
	}

	if resolver.isForCurrentPlatform(otherPkg) {
		t.Fatalf("expected %q NOT to match current platform", otherPkg)
	}
}

//
// testResolver mocks npm ls output
//
type testResolver struct {
	NpmResolver
	buffer io.Reader
}

func (r *testResolver) ListPkgPaths() (io.Reader, error) {
	return r.buffer, nil
}

//
// TC-004
// Regression test: non-cross-platform packages remain parsed
//
func TestGetInstalledPkgs_NonCrossPlatform(t *testing.T) {
	tmp := t.TempDir()
	nodeModules := filepath.Join(tmp, "node_modules")
	_ = os.MkdirAll(nodeModules, 0o755)

	paths := []string{
		filepath.Join(nodeModules, "lodash"),
		filepath.Join(nodeModules, "express"),
	}

	var b bytes.Buffer
	for _, p := range paths {
		_ = os.MkdirAll(p, 0o755)
		b.WriteString(p + "\n")
	}

	tr := &testResolver{buffer: &b}
	pkgs := tr.GetInstalledPkgs(nodeModules)

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 pkgs, got %d", len(pkgs))
	}
}

//
// TC-011
// Negative test: empty npm ls output
//
func TestGetInstalledPkgs_EmptyOutput(t *testing.T) {
	tmp := t.TempDir()
	nodeModules := filepath.Join(tmp, "node_modules")
	_ = os.MkdirAll(nodeModules, 0o755)

	tr := &testResolver{buffer: bytes.NewBuffer(nil)}
	pkgs := tr.GetInstalledPkgs(nodeModules)

	if len(pkgs) != 0 {
		t.Fatalf("expected 0 pkgs, got %d", len(pkgs))
	}
}

//
// TC-005 / TC-010
// Regression + negative test for ResolvePackageLicense
//
func TestResolvePackageLicense_CrossPlatformSkip(t *testing.T) {
	resolver := &NpmResolver{}
	cfg := &ConfigDeps{}

	var otherPkg string
	switch runtime.GOOS {
	case "linux":
		otherPkg = "pkg-darwin-arm64"
	case "darwin":
		otherPkg = "pkg-win32-x64"
	default:
		otherPkg = "pkg-linux-x64"
	}

	result := resolver.ResolvePackageLicense(otherPkg, "/fake/path", cfg)
	if !result.IsCrossPlatform {
		t.Fatalf("expected IsCrossPlatform=true for %q", otherPkg)
	}
}
