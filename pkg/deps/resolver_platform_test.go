// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// for additional information regarding copyright ownership.
// The ASF licenses this file under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations
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
type testResolver struct {
	deps.NpmResolver
	listOutput []string // mock paths returned by ListPkgPaths
}

func (r *testResolver) InstallPkgs() {
	// do nothing
}

func (r *testResolver) ListPkgPaths() (deps.Reader, error) {
	var b bytes.Buffer
	for _, path := range r.listOutput {
		b.WriteString(path + "\n")
	}
	return &b, nil
}

// -----------------------------
// Test: Skip cross-platform packages
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
// Test: Current platform package resolves normally
// -----------------------------
func TestResolvePackageLicense_CurrentPlatform(t *testing.T) {
	tmp := t.TempDir()
	pkgFile := filepath.Join(tmp, deps.PkgFileName)
	err := os.WriteFile(pkgFile, []byte(`{
		"name": "normal-pkg",
		"license": "Apache-2.0"
	}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	resolver := &deps.NpmResolver{}
	cfg := &deps.ConfigDeps{}

	result := resolver.ResolvePackageLicense("normal-pkg", tmp, cfg)

	if result.LicenseSpdxID != "Apache-2.0" {
		t.Fatalf("expected license Apache-2.0, got %q", result.LicenseSpdxID)
	}
}

// -----------------------------
// Test: Invalid path does not crash
// -----------------------------
func TestResolvePackageLicense_InvalidPath(t *testing.T) {
	resolver := &deps.NpmResolver{}
	cfg := &deps.ConfigDeps{}

	_ = resolver.ResolvePackageLicense("random-package", "/definitely/not/exist", cfg)
}

// -----------------------------
// Test: GetInstalledPkgs using mock ListPkgPaths
// -----------------------------
func TestGetInstalledPkgs_MockPaths(t *testing.T) {
	tmp := t.TempDir()
	nodeModules := filepath.Join(tmp, "node_modules")
	_ = os.MkdirAll(nodeModules, 0o755)

	mockPaths := []string{
		filepath.Join(nodeModules, "pkg1"),
		filepath.Join(nodeModules, "pkg2"),
	}

	tr := &testResolver{listOutput: mockPaths}
	pkgs := tr.GetInstalledPkgs(nodeModules)

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	if pkgs[0].Name != "pkg1" || pkgs[1].Name != "pkg2" {
		t.Fatalf("unexpected package names: %v, %v", pkgs[0].Name, pkgs[1].Name)
	}
}

// -----------------------------
// Test: InstallPkgs does not execute npm
// -----------------------------
func TestInstallPkgs_NoCrash(t *testing.T) {
	tr := &testResolver{}
	tr.InstallPkgs() // should not panic or run npm
}
