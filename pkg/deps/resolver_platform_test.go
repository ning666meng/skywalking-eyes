// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// for additional information regarding copyright ownership.
// The ASF licenses this file under the Apache License, Version 2.0 (the
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

package deps_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/apache/skywalking-eyes/pkg/deps"
)

// -----------------------------
// Mock resolver to avoid real npm commands
// -----------------------------
type mockNpmResolver struct {
	deps.NpmResolver
	mockPkgPaths []string
}

func (r *mockNpmResolver) ListPkgPaths() (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	for _, p := range r.mockPkgPaths {
		buf.WriteString(p + "\n")
	}
	return buf, nil
}

func (r *mockNpmResolver) InstallPkgs() {
	// noop
}

// -----------------------------
// Test 1: Cross-platform package should be skipped
// -----------------------------
func TestResolvePackageLicense_SkipCrossPlatform(t *testing.T) {
	resolver := &deps.NpmResolver{}
	cfg := &deps.ConfigDeps{}

	var pkg string
	switch runtime.GOOS {
	case "linux":
		pkg = "@parcel/watcher-darwin-arm64"
	case "darwin":
		pkg = "@parcel/watcher-linux-x64"
	default:
		pkg = "@parcel/watcher-linux-x64"
	}

	result := resolver.ResolvePackageLicense(pkg, "/non/existent/path", cfg)

	if result.LicenseSpdxID != "" {
		t.Fatalf("expected empty license for cross-platform package %q, got %q", pkg, result.LicenseSpdxID)
	}
}

// -----------------------------
// Test 2: Current-platform package parses normally
// -----------------------------
func TestResolvePackageLicense_CurrentPlatform(t *testing.T) {
	resolver := &deps.NpmResolver{}
	cfg := &deps.ConfigDeps{}

	tmp := t.TempDir()
	pkgFile := filepath.Join(tmp, deps.PkgFileName)
	err := os.WriteFile(pkgFile, []byte(`{
		"name": "normal-pkg",
		"license": "Apache-2.0"
	}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	result := resolver.ResolvePackageLicense("normal-pkg", tmp, cfg)

	if result.LicenseSpdxID != "Apache-2.0" {
		t.Fatalf("expected license Apache-2.0, got %q", result.LicenseSpdxID)
	}
}

// -----------------------------
// Test 3: Invalid path does not panic
// -----------------------------
func TestResolvePackageLicense_InvalidPath(t *testing.T) {
	resolver := &deps.NpmResolver{}
	cfg := &deps.ConfigDeps{}

	_ = resolver.ResolvePackageLicense("some-random-package", "/definitely/not/exist", cfg)
}

// -----------------------------
// Test 4: GetInstalledPkgs with mock paths
// -----------------------------
func TestGetInstalledPkgs_MockPaths(t *testing.T) {
	tmp := t.TempDir()
	p1 := filepath.Join(tmp, "pkg1")
	p2 := filepath.Join(tmp, "pkg2")
	os.MkdirAll(p1, 0o755)
	os.MkdirAll(p2, 0o755)

	mock := &mockNpmResolver{
		mockPkgPaths: []string{p1, p2},
	}

	pkgs := mock.GetInstalledPkgs(tmp)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	if pkgs[0].Path != p1 || pkgs[1].Path != p2 {
		t.Fatalf("unexpected package paths: %+v", pkgs)
	}
}

// -----------------------------
// Test 5: CanResolve returns true for package.json
// -----------------------------
func TestCanResolve(t *testing.T) {
	resolver := &deps.NpmResolver{}
	if !resolver.CanResolve("package.json") {
		t.Fatal("expected CanResolve to return true for package.json")
	}
	if resolver.CanResolve("other.json") {
		t.Fatal("expected CanResolve to return false for other.json")
	}
}

// -----------------------------
// Test 6: ResolveLicenseField handles string and object
// -----------------------------
func TestResolveLicenseField(t *testing.T) {
	resolver := &deps.NpmResolver{}

	// string license
	str := []byte(`"MIT"`)
	l, ok := resolver.ResolveLicenseField(str)
	if !ok || l != "MIT" {
		t.Fatal("failed to parse string license")
	}

	// object license
	obj := []byte(`{"type":"Apache-2.0"}`)
	l, ok = resolver.ResolveLicenseField(obj)
	if !ok || l != "Apache-2.0" {
		t.Fatal("failed to parse object license")
	}

	// empty
	l, ok = resolver.ResolveLicenseField([]byte{})
	if ok || l != "" {
		t.Fatal("expected empty result for empty data")
	}
}

// -----------------------------
// Test 7: ResolveLicensesField parses multiple licenses
// -----------------------------
func TestResolveLicensesField(t *testing.T) {
	resolver := &deps.NpmResolver{}
	lcs := []deps.Lcs{{Type: "MIT"}, {Type: "GPL-3.0"}}
	l, ok := resolver.ResolveLicensesField(lcs)
	if !ok || l != "MIT OR GPL-3.0" {
		t.Fatal("failed to parse multiple licenses")
	}
}
