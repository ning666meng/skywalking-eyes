// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information regarding copyright ownership.
// The ASF licenses this file under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package deps_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/apache/skywalking-eyes/pkg/deps"
)

// -----------------------------
// ResolvePackageLicense 测试
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
// CanResolve 测试
// -----------------------------

func TestCanResolve(t *testing.T) {
	resolver := &deps.NpmResolver{}

	if !resolver.CanResolve("package.json") {
		t.Fatal("expected CanResolve to return true for package.json")
	}
	if resolver.CanResolve("otherfile.txt") {
		t.Fatal("expected CanResolve to return false for non-package.json")
	}
}

// -----------------------------
// ListPkgPaths & GetInstalledPkgs 测试（模拟 npm 输出）
// -----------------------------

type mockResolver struct {
	deps.NpmResolver
	mockOutput io.Reader
}

func (r *mockResolver) ListPkgPaths() (io.Reader, error) {
	return r.mockOutput, nil
}

func TestGetInstalledPkgs_MockPaths(t *testing.T) {
	tmp := t.TempDir()
	nodeModules := filepath.Join(tmp, "node_modules")
	os.MkdirAll(filepath.Join(nodeModules, "lodash"), 0o755)
	os.MkdirAll(filepath.Join(nodeModules, "express"), 0o755)

	var b bytes.Buffer
	b.WriteString(filepath.Join(nodeModules, "lodash") + "\n")
	b.WriteString(filepath.Join(nodeModules, "express") + "\n")

	resolver := &mockResolver{
		mockOutput: &b,
	}

	pkgs := resolver.GetInstalledPkgs(nodeModules)
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}

	names := map[string]bool{}
	for _, p := range pkgs {
		names[p.Name] = true
	}
	if !names["lodash"] || !names["express"] {
		t.Fatal("expected packages 'lodash' and 'express' to be present")
	}
}

// -----------------------------
// 安全性测试（异常路径、防御分支）
// -----------------------------

func TestResolvePkgFile_NonExistent(t *testing.T) {
	resolver := &deps.NpmResolver{}
	_, err := resolver.ParsePkgFile("/definitely/not/exist/package.json")
	if err == nil {
		t.Fatal("expected error for non-existent package.json")
	}
}
