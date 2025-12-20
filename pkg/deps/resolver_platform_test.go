// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// for additional information regarding copyright ownership.
// The ASF licenses this file under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance with the
// License.  You may obtain a copy of the License at
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
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/apache/skywalking-eyes/pkg/deps"
)

//
// TC-NEW-001
// Regression test: cross-platform npm binary packages should be skipped
// (Node.js 24 introduces such packages via npm ls output)
//
func TestResolvePackageLicense_SkipCrossPlatformPackage(t *testing.T) {
	resolver := &deps.NpmResolver{}
	cfg := &deps.ConfigDeps{}

	var crossPlatformPkg string
	switch runtime.GOOS {
	case "linux":
		crossPlatformPkg = "@parcel/watcher-darwin-arm64"
	case "darwin":
		crossPlatformPkg = "@parcel/watcher-linux-x64"
	default:
		crossPlatformPkg = "@parcel/watcher-linux-x64"
	}

	result := resolver.ResolvePackageLicense(
		crossPlatformPkg,
		"/non/existent/path",
		cfg,
	)

	if result.LicenseSpdxID != "" {
		t.Fatalf(
			"expected empty license for cross-platform package %q, got %q",
			crossPlatformPkg,
			result.LicenseSpdxID,
		)
	}
}

//
// TC-NEW-002
// Behavior test: current-platform packages should still be parsed normally
//
func TestResolvePackageLicense_CurrentPlatformPackage(t *testing.T) {
	resolver := &deps.NpmResolver{}
	cfg := &deps.ConfigDeps{}

	tmp := t.TempDir()
	pkgJSON := filepath.Join(tmp, "package.json")

	err := os.WriteFile(pkgJSON, []byte(`{
		"name": "normal-pkg",
		"license": "Apache-2.0"
	}`), 0644)
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
			"expected license Apache-2.0 for current-platform package, got %q",
			result.LicenseSpdxID,
		)
	}
}

//
// TC-NEW-003
// Safety test: malformed or unexpected package paths should not cause panic
//
func TestResolvePackageLicense_InvalidPathDoesNotCrash(t *testing.T) {
	resolver := &deps.NpmResolver{}
	cfg := &deps.ConfigDeps{}

	result := resolver.ResolvePackageLicense(
		"some-random-package",
		"/definitely/not/exist",
		cfg,
	)

	// No panic is the main assertion.
	_ = result
}

//
// TC-NEW-004
// Regression test: cross-platform npm package should be skipped
// even if package.json exists
//
func TestResolvePackageLicense_CrossPlatformWithPkgJSON(t *testing.T) {
	resolver := &deps.NpmResolver{}
	cfg := &deps.ConfigDeps{}

	tmp := t.TempDir()
	err := os.WriteFile(
		filepath.Join(tmp, "package.json"),
		[]byte(`{"license":"MIT"}`),
		0644,
	)
	if err != nil {
		t.Fatal(err)
	}

	var crossPlatformPkg string
	switch runtime.GOOS {
	case "linux":
		crossPlatformPkg = "@parcel/watcher-darwin-arm64"
	case "darwin":
		crossPlatformPkg = "@parcel/watcher-linux-x64"
	default:
		crossPlatformPkg = "@parcel/watcher-linux-x64"
	}

	result := resolver.ResolvePackageLicense(crossPlatformPkg, tmp, cfg)

	if result.LicenseSpdxID != "" {
		t.Fatalf(
			"expected empty license for cross-platform package %q even with package.json, got %q",
			crossPlatformPkg,
			result.LicenseSpdxID,
		)
	}
}
