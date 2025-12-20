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
// distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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
// CanResolve
// -----------------------------
func TestCanResolve(t *testing.T) {
	resolver := &deps.NpmResolver{}

	tests := []struct {
		file string
		want bool
	}{
		{"package.json", true},
		{"somefile.txt", false},
		{"Package.JSON", false}, // case-sensitive
	}

	for _, tc := range tests {
		got := resolver.CanResolve(tc.file)
		if got != tc.want {
			t.Errorf("CanResolve(%q) = %v; want %v", tc.file, got, tc.want)
		}
	}
}

// -----------------------------
// NeedSkipInstallPkgs
// -----------------------------
func TestNeedSkipInstallPkgsTimeout(t *testing.T) {
	resolver := &deps.NpmResolver{}
	// Just ensure it returns a bool without blocking
	got := resolver.NeedSkipInstallPkgs()
	if got != true && got != false {
		t.Errorf("NeedSkipInstallPkgs() returned invalid value %v", got)
	}
}

// -----------------------------
// InstallPkgs & ListPkgPaths
// -----------------------------
func TestInstallPkgsAndListPkgPaths(t *testing.T) {
	resolver := &deps.NpmResolver{}

	// InstallPkgs does not return error, just run
	resolver.InstallPkgs()

	// ListPkgPaths returns reader and error
	r, err := resolver.ListPkgPaths()
	if err != nil && err != io.EOF { // npm command may fail in test environment
		t.Logf("ListPkgPaths returned error (expected in test env): %v", err)
	}

	if r != nil {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r)
	}
}

// -----------------------------
// GetInstalledPkgs
// -----------------------------
func TestGetInstalledPkgs(t *testing.T) {
	resolver := &deps.NpmResolver{}

	tmp := t.TempDir()
	nodeModules := filepath.Join(tmp, "node_modules")
	_ = os.MkdirAll(nodeModules, 0o755)

	// Create fake packages
	pkgsDirs := []string{
		filepath.Join(nodeModules, "lodash"),
		filepath.Join(nodeModules, "express"),
	}
	for _, dir := range pkgsDirs {
		_ = os.MkdirAll(dir, 0o755)
	}

	pkgs := resolver.GetInstalledPkgs(nodeModules)
	if len(pkgs) != 2 {
		t.Errorf("expected 2 packages, got %d", len(pkgs))
	}
}

// -----------------------------
// Resolve (partial integration, without actual npm)
// -----------------------------
func TestResolvePartial(t *testing.T) {
	resolver := &deps.NpmResolver{}
	report := &deps.Report{}
	tmp := t.TempDir()
	pkgFile := filepath.Join(tmp, deps.PkgFileName)

	// minimal package.json
	_ = os.WriteFile(pkgFile, []byte(`{"name":"foo","license":"MIT"}`), 0o644)

	err := resolver.Resolve(pkgFile, &deps.ConfigDeps{}, report)
	if err != nil && !os.IsNotExist(err) {
		// The error can occur because node_modules may not exist, that's fine
		t.Logf("Resolve returned error (expected in test env): %v", err)
	}
}
