// Copyright 2024 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var skipTests = map[string]string{
	"archive/tar":                         ".*InsecurePath.*|FileInfoHeader.*|Pax.*|USTARLongName",
	"archive/zip":                         ".*InsecurePath.*|Reader|FS.*|Over65kFiles|Zip64LargeDirectory|Zip64ManyRecords|Zip64DirectoryOffset|CVE202127919|CVE202141772|CVE202127919|UnderSize|Issue54801",
	"arena":                               "",
	"bytes":                               "Grow",
	"cmd/addr2line":                       "",
	"cmd/api":                             "",
	"cmd/cgo/internal/swig":               "",
	"cmd/cgo/internal/testcarchive":       "",
	"cmd/cgo/internal/testcshared":        "",
	"cmd/cgo/internal/testerrors":         "",
	"cmd/cgo/internal/testgodefs":         "",
	"cmd/cgo/internal/testlife":           "",
	"cmd/cgo/internal/testplugin":         "",
	"cmd/cgo/internal/testsanitizers":     "",
	"cmd/cgo/internal/testshared":         "",
	"cmd/cgo/internal/teststdio":          "",
	"cmd/compile/internal/amd64":          "",
	"cmd/compile/internal/dwarfgen":       "",
	"cmd/compile/internal/importer":       "",
	"cmd/compile/internal/inline/inlheur": "DumpCallSiteScoreDump|FuncProperties",
	"cmd/compile/internal/noder":          "",
	"cmd/compile/internal/rangefunc":      "",
	"cmd/compile/internal/ssa":            "",
	"cmd/compile/internal/reflectdata":    "",
	"cmd/compile/internal/syntax":         "StdLib",
	"cmd/compile/internal/test":           "",
	"cmd/compile/internal/types2":         "",
	"cmd/covdata":                         "",
	"cmd/cover":                           "",
	"cmd/doc":                             "",
	"cmd/go":                              "",
	"cmd/gofmt":                           "Rewrite|BackupFile|All",
	"cmd/internal/bootstrap_test":         "",
	"cmd/internal/buildid":                "",
}

func testPkgs(dir string) []string {
	files, err := os.ReadDir(dir)
	dieErr(err)
	var (
		pkgs    []string
		hasTest bool
	)
	for _, f := range files {
		name := f.Name()
		if !f.IsDir() {
			if strings.HasSuffix(name, "_test.go") {
				hasTest = true
			}
			continue
		}
		pkg := filepath.Join(dir, name)
		if skip, ok := skipTests[pkg]; !ok || skip != "" {
			pkgs = append(pkgs, testPkgs(pkg)...)
		}
	}
	if hasTest {
		pkgs = append([]string{dir}, pkgs...)
	}
	return pkgs
}

func stdtests(goCmd string) {
	dieErr(os.Setenv("GOTARGET", "noostest"))
	dieErr(os.Setenv("GOOS", "noos"))

	goroot, err := exec.Command(goCmd, "env", "GOROOT").Output()
	dieErr(err)
	dieErr(os.Chdir(filepath.Join(strings.TrimSpace(string(goroot)), "src")))

	pkgs := testPkgs(".")

	for _, arch := range []string{"thumb", "riscv64"} {
		dieErr(os.Setenv("GOARCH", arch))
		fmt.Print("#### GOARCH=", arch, " ####\n\n")
		dieErr(err)
		for _, pkg := range pkgs {
			cmd := &exec.Cmd{
				Path:   goCmd,
				Args:   []string{"emgo", "test", pkg, "-skip", ""},
				Stdin:  os.Stdin,
				Stdout: os.Stdout,
				Stderr: os.Stderr,
			}
			skipArg := "^(Fuzz.*|Example.*"
			if skip := skipTests[pkg]; skip != "" {
				skipArg += "|Test(" + skip + ")"
			}
			skipArg += ")$"
			cmd.Args[len(cmd.Args)-1] = skipArg
			cmd.Args = append(cmd.Args, os.Args[2:]...)
			//fmt.Println(cmd.Args)
			noosBuildTest(cmd, cfgFromEnv())
		}
	}
}
