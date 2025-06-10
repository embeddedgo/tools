// Copyright 2024 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

var skipTests = map[string]string{
	"archive/tar":                                ".*InsecurePath.*|FileInfoHeader.*|Pax.*|USTARLongName",
	"archive/zip":                                ".*InsecurePath.*|Reader|FS.*|Over65kFiles|Zip64LargeDirectory|Zip64ManyRecords|Zip64DirectoryOffset|CVE202127919|CVE202141772|CVE202127919|UnderSize|Issue54801",
	"arena":                                      "",
	"bytes":                                      "Grow",
	"cmd/addr2line":                              "",
	"cmd/api":                                    "",
	"cmd/cgo/internal/swig":                      "",
	"cmd/cgo/internal/testcarchive":              "",
	"cmd/cgo/internal/testcshared":               "",
	"cmd/cgo/internal/testerrors":                "",
	"cmd/cgo/internal/testgodefs":                "",
	"cmd/cgo/internal/testlife":                  "",
	"cmd/cgo/internal/testplugin":                "",
	"cmd/cgo/internal/testsanitizers":            "",
	"cmd/cgo/internal/testshared":                "",
	"cmd/cgo/internal/teststdio":                 "",
	"cmd/compile":                                "",
	"cmd/compile/internal/amd64":                 "",
	"cmd/compile/internal/dwarfgen":              "",
	"cmd/compile/internal/importer":              "",
	"cmd/compile/internal/inline/inlheur":        "DumpCallSiteScoreDump|FuncProperties",
	"cmd/compile/internal/noder":                 "",
	"cmd/compile/internal/rangefunc":             "",
	"cmd/compile/internal/ssa":                   "",
	"cmd/compile/internal/reflectdata":           "",
	"cmd/compile/internal/syntax":                "StdLib",
	"cmd/compile/internal/test":                  "",
	"cmd/compile/internal/types2":                "",
	"cmd/covdata":                                "",
	"cmd/cover":                                  "",
	"cmd/doc":                                    "",
	"cmd/go":                                     "",
	"cmd/gofmt":                                  "Rewrite|BackupFile|All",
	"cmd/internal/bootstrap_test":                "",
	"cmd/internal/buildid":                       "",
	"cmd/internal/notsha256":                     "BlockGeneric",
	"cmd/internal/obj/arm64":                     "NoRet",
	"cmd/internal/obj/loong64":                   "NoRet",
	"cmd/internal/obj/riscv":                     "PCAlign|ImmediateSplitting|NoRet",
	"cmd/internal/obj/riscv/testdata/testbranch": "",
	"cmd/internal/pkgpath":                       "ToSymbolFunc",
	"cmd/link":                                   "",
	"cmd/nm":                                     "NonGoExecs|GoLib|GoExec",
	"cmd/objdump":                                "",
	"cmd/pack":                                   "",
	"cmd/pprof":                                  "",
	"cmd/trace":                                  "",
	"cmd/vet":                                    "",
	"compress/bzip2":                             "",
	"compress/flate":                             "DeflateFast_Reset|WriteError|BestSpeed",
	"compress/gzip":                              "",
	"crypto/boring":                              "",
	"crypto/cipher":                              "",
	"crypto/dsa":                                 "ParameterGeneration|SignAndVerify|SigningWithDegenerateKeys",
	"crypto/ecdh":                                "ECDH|GenerateKey|X25519Failure|MismatchedCurves",
	"crypto/ecdsa":                               "",
	"crypto/ed25519":                             "GenerateKey|Equal",
	"crypto/elliptic":                            "",
	"crypto/internal/fips140/bigmod":             "Mul|RightShift",
	"crypto/internal/fips140/ecdsa":              "RandomPoint",
	"crypto/internal/fips140/edwards25519/field": "Consistency|Invert",
	"crypto/internal/fips140/mlkem":              "EncodeDecode",
	"crypto/internal/fips140test":                "",
	"crypto/internal/sysrand":                    "",
	"crypto/md5":                                 "BlockGeneric",
	"crypto/mlkem":                               "RoundTrip|BadLengths",
	"crypto/rand":                                "",
	"crypto/rsa":                                 ".*PK.*CS1v15.*|NonZeroRandomBytes|ShortSessionKey|.*PSS.*|HashOverride|.*KeyGeneration|Allocations|Everything|EncryptDecryptOAEP|PSmallerThanQ|UnpaddedSignature|GnuTLSKey",
	"crypto/sha1":                                "BlockGeneric",
	"crypto/sha3":                                "CSHAKELargeS",
	"crypto/subtle":                              "XORBytes",
	"crypto/tls":                                 "",
	"crypto/x509":                                "",
	"debug/elf":                                  "",
	"embedded/rtos":                              "",
	"encoding/gob":                               "LargeSlice|CountDecodeMallocs",
	"encoding/json":                              "",
	"encoding/pem":                               "CVE202224675",
	"encoding/xml":                               "CVE202228131|CVE202230633",
	"flag":                                       "ExitCode",
	"fmt":                                        "CountMallocs|ScanInts",
	"go/build":                                   "",
	"go/doc":                                     "",
	"go/internal/gccgoimporter":                  "",
	"go/internal/srcimporter":                    "",
	"go/parser":                                  "",
	"go/types":                                   "",
	"hash/maphash":                               "Comparable|WriteComparable|SmhasherSmallKeys|SmhasherTwoNonzero|SmhasherPermutation|SmhasherText|SmhasherAvalanche|SmhasherSparse|SmhasherCyclic",
	"html/template":                              "MaxExecDepth|ExecutePanicDuringCall|ParseGlob.*|ParseFS",
	"image/jpeg":                                 "LargeImageWithShortData",
	"index/suffixarray":                          "New32|New64",
	"internal/copyright":                         "",
	"internal/coverage/cfile/testdata/issue56006": "",
	"internal/coverage/pods":                      "PodCollection",
	"internal/coverage/test":                      "CounterDataWriterReader|CounterDataAppendSegment|MetaDataWriterReader",
	"internal/diff":                               "",
	"internal/godebug":                            "",
	"internal/runtime/syscall":                    "",
	"internal/saferio":                            "ReadData",
	"internal/sync":                               "ConcurrentCache",
	"internal/synctest":                           "ReflectFuncOf",
	"internal/syscall/windows":                    "",
	"internal/testenv":                            "CleanCmdEnvPWD|CleanCmdEnvPWD",
	"internal/trace":                              "",
	"internal/zstd":                               "FileSamples|ReaderBad",
	"image/png":                                   "DimensionOverflow",
	"io":                                          "OffsetWriter_.*|WriteAt_PositionPriorToBase|CVE202230630",
	"io/fs":                                       "Glob.*|CVE202230630|ReadDirPath|ReadFilePath|WalkDir|Issue51617",
	"io/ioutil":                                   "ReadFile|WriteFile|ReadOnlyWriteFile|ReadDir|TempFile.*|TempDir.*",
	"log/slog":                                    "",
	"log/syslog":                                  "",
	"math/big":                                    "LinkerGC",
	"math/rand":                                   "ReadUniformity|DefaultRace|SeedNop",
	"mime":                                        "",
	"net":                                         "",
	"os":                                          "",
	"path/filepath":                               "Walk.*|Glob.*|.*Symlink.*|CVE202230632|NonWindowsGlobEscape|Issue13582|Abs|AbsEmptyString|Bug3486|Issue29372|Issue51617|Issue516172272189953|Escaping",
	"reflect":                                     "",
	"regexp":                                      "BadCompile|OnePassCutoff",
	"regexp/syntax":                               "ParseInvalidRegexps",
	"runtime":                                     "",
	"strings":                                     "CaseConsistency",
	"sync":                                        "PoolChain|MutexMisuse",
	"sync/atomic":                                 "NilDeref|ValueCompareAndSwapConcurrent",
	"syscall":                                     "",
	"testing":                                     "Flag|Panic.*|MorePanic|TempDir.*|.*Race.*|Chdir.*|Exec.*|BenchmarkB.*|RunningTests.*|TBHelper.*",
	"testing/fstest":                              "Symlink",
	"testing/synctest":                            "",
	"text/template":                               "MaxExecDepth|ExecutePanicDuringCall|LinkerGC|ParseFS|ParseGlob.*",
	"time":                                        "",
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
	slices.Sort(pkgs)
	return pkgs
}

func stdtests(goCmd string) {
	dieErr(os.Setenv("GOTARGET", "noostest"))
	dieErr(os.Setenv("GOOS", "noos"))

	goroot, err := exec.Command(goCmd, "env", "GOROOT").Output()
	dieErr(err)
	dieErr(os.Chdir(filepath.Join(strings.TrimSpace(string(goroot)), "src")))

	pkgs := testPkgs(".")

	for _, arch := range []string{ "thumb", "riscv64"} {
		dieErr(os.Setenv("GOARCH", arch))
		fmt.Print("\n#### GOARCH=", arch, " ####\n\n")
		dieErr(err)
		for _, pkg := range pkgs {
			cmd := &exec.Cmd{
				Path:   goCmd,
				Args:   []string{"emgo", "test", pkg, "-timeout", "30m", "-skip", ""},
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
