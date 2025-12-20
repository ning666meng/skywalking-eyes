// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// for additional information regarding copyright ownership.
// The ASF licenses this file under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations.

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
// 基础测试
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

func TestResolvePackageLicense_CurrentPlatform(t *testing.T) {
	resolver := &deps.NpmResolver{}
	cfg := &deps.ConfigDeps{}

	tmp := t.TempDir()
	pkgFile := filepath.Join(tmp, deps.PkgFileName)

	os.WriteFile(pkgFile, []byte(`{"name":"normal-pkg","license":"Apache-2.0"}`), 0o644)

	result := resolver.ResolvePackageLicense("normal-pkg", tmp, cfg)
	if result.LicenseSpdxID != "Apache-2.0" {
		t.Fatalf("expected license Apache-2.0, got %q", result.LicenseSpdxID)
	}
}

func TestResolvePackageLicense_InvalidPath(t *testing.T) {
	resolver := &deps.NpmResolver{}
	cfg := &deps.ConfigDeps{}

	_ = resolver.ResolvePackageLicense("some-random-package", "/definitely/not/exist", cfg)
}

// -----------------------------
// 补充测试覆盖未覆盖方法
// -----------------------------

func TestCanResolve(t *testing.T) {
	resolver := &deps.NpmResolver{}
	if !resolver.CanResolve(deps.PkgFileName) {
		t.Fatal("expected CanResolve to return true for package.json")
	}
	if resolver.CanResolve("otherfile.txt") {
		t.Fatal("expected CanResolve to return false for non-package.json")
	}
}

// MockResolver 用于替代 ListPkgPaths / GetInstalledPkgs 的外部调用
type mockResolver struct {
	deps.NpmResolver
	mockOutput io.Reader
}

func (m *mockResolver) ListPkgPaths() (io.Reader, error) {
	return m.mockOutput, nil
}

func TestGetInstalledPkgs_MockPaths(t *testing.T) {
	tmp := t.TempDir()
	nodeModules := filepath.Join(tmp, "node_modules")
	lodashPath := filepath.Join(nodeModules, "lodash")
	expressPath := filepath.Join(nodeModules, "express")

	os.MkdirAll(lodashPath, 0o755)
	os.MkdirAll(expressPath, 0o755)

	// 模拟 ListPkgPaths 输出绝对路径
	var b bytes.Buffer
	b.WriteString(lodashPath + "\n")
	b.WriteString(expressPath + "\n")

	resolver := &mockResolver{mockOutput: &b}
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

func TestNeedSkipInstallPkgs_Timeout(t *testing.T) {
	resolver := &deps.NpmResolver{}
	// 这里主要测试不 panic, 倒计时结束后返回 false
	got := resolver.NeedSkipInstallPkgs()
	if got != false {
		t.Fatal("expected NeedSkipInstallPkgs to return false on timeout")
	}
}

func TestInstallPkgs_NoPanic(t *testing.T) {
	resolver := &deps.NpmResolver{}
	// 仅验证不 panic
	resolver.InstallPkgs()
}
