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

var skipTests1 = map[string]string{
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
	"cmd/internal/notsha256":              "BlockGeneric",
	"cmd/internal/obj/riscv":              "PCAlign|ImmediateSplitting",
	"cmd/internal/pkgpath":                "ToSymbolFunc",
	"cmd/link":                            "",
	"cmd/nm":                              "NonGoExecs|GoLib|GoExec",
	"cmd/objdump":                         "",
	"cmd/pack":                            "",
	"cmd/pprof":                           "",
	"cmd/trace":                           "",
	"cmd/vet":                             "",
	"compress/bzip2":                      "",
	"compress/flate":                      "DeflateFast_Reset|WriteError",
	"compress/gzip":                       "",
	"crypto/boring":                       "",
	"debug/elf":                           "",
	"embedded/rtos":                       "",
	"encoding/gob":                        "LargeSlice",
}

var skipTests = map[string]string{
	"archive/tar":                         "",
	"archive/zip":                         "",
	"arena":                               "",
	"bufio":                               "",
	"bytes":                               "",
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
	"cmd/asm/internal/asm":                "",
	"cmd/asm/internal/lex":                "",
	"cmd/cgo/internal/test":               "",
	"cmd/cgo/internal/testfortran":        "",
	"cmd/cgo/internal/testnocgo":          "",
	"cmd/cgo/internal/testso":             "",
	"cmd/cgo/internal/testtls":            "",
	"cmd/compile/internal/abt":            "",
	"cmd/compile/internal/base":           "",
	"cmd/compile/internal/compare":        "",
	"cmd/compile/internal/devirtualize":   "",
	"cmd/compile/internal/inline/inlheur": "",
	"cmd/compile/internal/ir":             "",
	"cmd/compile/internal/logopt":         "",
	"cmd/compile/internal/loopvar":        "",
	"cmd/compile/internal/noder":          "",
	"cmd/compile/internal/rangefunc":      "",
	"cmd/compile/internal/ssa":            "",
	"cmd/compile/internal/reflectdata":    "",
	"cmd/compile/internal/syntax":         "",
	"cmd/compile/internal/test":           "",
	"cmd/compile/internal/types2":         "",
	"cmd/covdata":                         "",
	"cmd/cover":                           "",
	"cmd/doc":                             "",
	"cmd/go":                              "",
	"cmd/compile/internal/typecheck":      "",
	"cmd/compile/internal/types":          "",
	"cmd/dist":                            "",
	"cmd/distpack":                        "",
	"cmd/fix":                             "",
	"cmd/gofmt":                           "",
	"cmd/internal/bootstrap_test":         "",
	"cmd/internal/buildid":                "",
	"cmd/internal/archive":                "",
	"cmd/internal/cov":                    "",
	"cmd/internal/dwarf":                  "",
	"cmd/internal/edit":                   "",
	"cmd/internal/goobj":                  "",
	"cmd/internal/moddeps":                "",
	"cmd/internal/notsha256":              "",
	"cmd/internal/obj":                    "",
	"cmd/internal/obj/arm64":              "",
	"cmd/internal/obj/ppc64":              "",
	"cmd/internal/obj/riscv":              "",
	"cmd/internal/objabi":                 "",
	"cmd/internal/pkgpath":                "",
	"cmd/internal/pkgpattern":             "",
	"cmd/internal/quoted":                 "",
	"cmd/internal/src":                    "",
	"cmd/internal/test2json":              "",
	"cmd/internal/testdir":                "",
	"cmd/link":                            "",
	"cmd/nm":                              "",
	"cmd/objdump":                         "",
	"cmd/pack":                            "",
	"cmd/pprof":                           "",
	"cmd/trace":                           "",
	"cmd/vet":                             "",
	"cmp":                                 "",
	"compress/bzip2":                      "",
	"compress/flate":                      "",
	"compress/gzip":                       "",
	"crypto/boring":                       "",
	"compress/lzw":                        "",
	"compress/zlib":                       "",
	"container/heap":                      "",
	"container/list":                      "",
	"container/ring":                      "",
	"context":                             "",
	"crypto":                              "",
	"crypto/aes":                          "",
	"database/sql":                        "",
	"database/sql/driver":                 "",
	"debug/buildinfo":                     "",
	"debug/dwarf":                         "",
	"debug/elf":                           "",
	"debug/gosym":                         "",
	"debug/macho":                         "",
	"debug/pe":                            "",
	"debug/plan9obj":                      "",
	"embedded/rtos":                       "",
	"embed":                               "",
	"embed/internal/embedtest":            "",
	"encoding/ascii85":                    "",
	"encoding/asn1":                       "",
	"encoding/base32":                     "",
	"encoding/base64":                     "",
	"encoding/binary":                     "",
	"encoding/csv":                        "",
	"encoding/gob":                        "LargeSlice",
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
			noosBuildTestVet(cmd, cfgFromEnv())
		}
	}
}
