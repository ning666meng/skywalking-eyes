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
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package deps_test

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/apache/skywalking-eyes/pkg/deps"
)

// -----------------------------
// Original tests
// -----------------------------

func TestResolvePackageLicense_SkipCrossPlatform(t *testing.T) {
	resolver := &deps.NpmResolver{}
	cfg := &deps.ConfigDeps{}

	var pkg string
	switch os := os.Getenv("GOOS"); os {
	case "linux":
		pkg = "@parcel/watcher-darwin-arm64"
	case "darwin":
		pkg = "@parcel/watcher-linux-x64"
	default:
		pkg = "@parcel/watcher-linux-x64"
	}

	result := resolver.ResolvePackageLicense(
		pkg,
		"/non/existent/path",
		cfg,
	)

	if result.LicenseSpdxID != "" {
		t.Fatalf(
			"expected empty license for cross-platform package %q, got %q",
			pkg, result.LicenseSpdxID,
		)
	}
}

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

	result := resolver.ResolvePackageLicense(
		"normal-pkg",
		tmp,
		cfg,
	)

	if result.LicenseSpdxID != "Apache-2.0" {
		t.Fatalf(
			"expected license Apache-2.0, got %q",
			result.LicenseSpdxID,
		)
	}
}

func TestResolvePackageLicense_InvalidPath(t *testing.T) {
	resolver := &deps.NpmResolver{}
	cfg := &deps.ConfigDeps{}

	_ = resolver.ResolvePackageLicense(
		"some-random-package",
		"/definitely/not/exist",
		cfg,
	)
}

// -----------------------------
// Additional coverage tests
// -----------------------------

func TestCanResolve(t *testing.T) {
	resolver := &deps.NpmResolver{}
	if !resolver.CanResolve("package.json") {
		t.Fatal("package.json should be resolvable")
	}
	if resolver.CanResolve("other.json") {
		t.Fatal("other.json should not be resolvable")
	}
}

func TestResolveLicenseFieldAndLicensesField(t *testing.T) {
	resolver := &deps.NpmResolver{}

	// license string
	lcs, ok := resolver.ResolveLicenseField([]byte(`"MIT"`))
	if !ok || lcs != "MIT" {
		t.Fatalf("expected MIT, got %q", lcs)
	}

	// license object
	lcs, ok = resolver.ResolveLicenseField([]byte(`{"type":"Apache-2.0"}`))
	if !ok || lcs != "Apache-2.0" {
		t.Fatalf("expected Apache-2.0, got %q", lcs)
	}

	// licenses array
	arr := []deps.Lcs{{Type: "MIT"}, {Type: "GPL-3.0"}}
	lcsStr, ok := resolver.ResolveLicensesField(arr)
	if !ok || lcsStr != "MIT OR GPL-3.0" {
		t.Fatalf("expected MIT OR GPL-3.0, got %q", lcsStr)
	}
}

func TestParsePkgFileCustom(t *testing.T) {
	tmp := t.TempDir()
	pkgFile := filepath.Join(tmp, deps.PkgFileName)
	content := `{"name":"testpkg","license":"MIT","version":"1.0.0"}`
	if err := os.WriteFile(pkgFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	resolver := &deps.NpmResolver{}
	pkg, err := resolver.ParsePkgFile(pkgFile)
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Name != "testpkg" {
		t.Fatalf("expected testpkg, got %q", pkg.Name)
	}
	if pkg.Version != "1.0.0" {
		t.Fatalf("expected 1.0.0, got %q", pkg.Version)
	}
}

func TestResolveLcsFileCustom(t *testing.T) {
	tmp := t.TempDir()
	licenseFile := filepath.Join(tmp, "LICENSE")
	content := "MIT LICENSE CONTENT"
	if err := os.WriteFile(licenseFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	resolver := &deps.NpmResolver{}
	result := &deps.Result{}
	if err := resolver.ResolveLcsFile(result, tmp, &deps.ConfigDeps{}); err != nil {
		t.Fatal(err)
	}
	if result.LicenseContent != content {
		t.Fatalf("expected license content, got %q", result.LicenseContent)
	}
	if result.LicenseFilePath != licenseFile {
		t.Fatalf("expected license file path, got %q", result.LicenseFilePath)
	}
}

// Mock resolver for GetInstalledPkgs
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

func (r *mockNpmResolver) GetInstalledPkgs(pkgDir string) []*deps.Package {
	buffer, _ := r.ListPkgPaths()
	sc := bufio.NewScanner(buffer)
	pkgs := []*deps.Package{}
	for sc.Scan() {
		p := sc.Text()
		pkgs = append(pkgs, &deps.Package{
			Name: filepath.Base(p),
			Path: p,
		})
	}
	return pkgs
}

func TestGetInstalledPkgs_MockPaths(t *testing.T) {
	tmp := t.TempDir()
	p1 := filepath.Join(tmp, "pkg1")
	p2 := filepath.Join(tmp, "pkg2")

	mock := &mockNpmResolver{
		mockPkgPaths: []string{p1, p2},
	}

	pkgs := mock.GetInstalledPkgs(tmp)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
}
