// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements. See the NOTICE file
// for additional information regarding copyright ownership.
// The ASF licenses this file under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance with the
// License. You may obtain a copy of the License at
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

var crossPlatformPackages = map[string]string{
	"linux":   "@parcel/watcher-darwin-arm64",
	"darwin":  "@parcel/watcher-linux-x64",
	"default": "@parcel/watcher-linux-x64",
}

var TestResolvePackageLicenseData = []struct {
	name        string
	packageName string
	pkgJSON     string
	expect      string
}{
	{
		name:        "Skip cross-platform package",
		packageName: "",
		expect:      "",
	},
	{
		name:        "Current platform package",
		packageName: "normal-pkg",
		pkgJSON: `{
			"name": "normal-pkg",
			"license": "Apache-2.0"
		}`,
		expect: "Apache-2.0",
	},
	{
		name:        "Invalid path does not crash",
		packageName: "some-random-package",
		expect:      "",
	},
	{
		name:        "Cross-platform package with package.json",
		packageName: "",
		pkgJSON:     `{"license":"MIT"}`,
		expect:      "",
	},
}

func TestResolvePackageLicense(t *testing.T) {
	resolver := new(deps.NpmResolver)
	cfg := &deps.ConfigDeps{}
	tmp := t.TempDir()

	for _, tt := range TestResolvePackageLicenseData {
		t.Run(tt.name, func(t *testing.T) {
			pkg := tt.packageName
			if pkg == "" {
				switch runtime.GOOS {
				case "linux":
					pkg = crossPlatformPackages["linux"]
				case "darwin":
					pkg = crossPlatformPackages["darwin"]
				default:
					pkg = crossPlatformPackages["default"]
				}
			}

			if tt.pkgJSON != "" {
				pkgFile := filepath.Join(tmp, "package.json")
				if err := os.WriteFile(pkgFile, []byte(tt.pkgJSON), 0644); err != nil {
					t.Fatal(err)
				}
			}

			result := resolver.ResolvePackageLicense(pkg, tmp, cfg)
			if result.LicenseSpdxID != tt.expect {
				t.Fatalf("expected license %q, got %q", tt.expect, result.LicenseSpdxID)
			}
		})
	}
}
