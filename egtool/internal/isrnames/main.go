// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package isrnames

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/embeddedgo/tools/egtool/internal/util"
)

const Descr = "generate ISR names based on the hal/irq package"

func Main(cmd string, args []string) {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage:\n  %s [OUTPUT] \nOptions:\n",
			cmd,
		)
		fs.PrintDefaults()
	}
	fs.Parse(args)
	if fs.NArg() > 2 {
		fs.Usage()
		os.Exit(1)
	}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ParseComments)
	util.FatalErr("", err)
	var handlers []string
	for _, pkg := range pkgs {
		if pkg.Name != "main" {
			continue
		}
		for _, file := range pkg.Files {
			for _, d := range file.Decls {
				if f, ok := d.(*ast.FuncDecl); ok {
					if f.Doc == nil {
						continue
					}
					for _, c := range f.Doc.List {
						if !strings.HasPrefix(c.Text, "//go:") {
							continue
						}
						s := c.Text[5:]
						if i := strings.IndexFunc(s, unicode.IsSpace); i >= 0 {
							s = s[:i]
						}
						if s == "interrupthandler" {
							handlers = append(handlers, f.Name.Name)
							continue
						}
						if _, ok := slices.BinarySearch(directives[:], s); !ok {
							util.Fatal(
								"%v: unknown function directive: //go:%s",
								fset.Position(c.Slash), s,
							)
						}
					}
				}
			}
		}
	}
	fmt.Println(handlers)
}

// Function directives, sorted for binary search.
var directives = [...]string{
	"linkname",
	"noescape",
	"noinline",
	"norace",
	"nosplit",
	"nowritebarrier",
	"nowritebarrierrec",
	"systemstack",
	"uintptrescapes",
	"uintptrkeepalive",
	"yeswritebarrierrec",
}
