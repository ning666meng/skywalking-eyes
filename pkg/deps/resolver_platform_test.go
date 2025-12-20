// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY
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
// Mock NpmResolver to avoid real npm calls
// -----------------------------
type mockResolver struct {
	deps.NpmResolver
	mockPaths []string
}

func (r *mockResolver) InstallPkgs() {
	// do nothing
}

func (r *mockResolver) ListPkgPaths() (*bytes.Buffer, error) {
	buffer := &bytes.Buffer{}
	for _, p := range r.mockPaths {
		buffer.WriteString(p + "\n")
	}
	return buffer, nil
}

// -----------------------------
// Test: skip cross-platform packages
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
		t.Fatalf("expected empty license for cross-platform package %q, got %q",
			pkg, result.LicenseSpdxID)
	}
}

// -----------------------------
// Test: current platform package with package.json
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
// Test: invalid path should not panic
// -----------------------------
func TestResolvePackageLicense_InvalidPath(t *testing.T) {
	resolver := &deps.NpmResolver{}
	cfg := &deps.ConfigDeps{}

	_ = resolver.ResolvePackageLicense("some-random-package", "/definitely/not/exist", cfg)
}

// -----------------------------
// Test: GetInstalledPkgs with mock paths
// -----------------------------
func TestGetInstalledPkgs_MockPaths(t *testing.T) {
	tmp := t.TempDir()
	// simulate two packages
	p1 := filepath.Join(tmp, "pkg1")
	p2 := filepath.Join(tmp, "pkg2")
	os.MkdirAll(p1, 0o755)
	os.MkdirAll(p2, 0o755)

	mock := &mockResolver{
		mockPaths: []string{p1, p2},
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
// Test: CanResolve
// -----------------------------
func TestCanResolve(t *testing.T) {
	resolver := &deps.NpmResolver{}
	if !resolver.CanResolve(deps.PkgFileName) {
		t.Fatal("CanResolve should return true for package.json")
	}
	if resolver.CanResolve("other.json") {
		t.Fatal("CanResolve should return false for non-package.json")
	}
}
