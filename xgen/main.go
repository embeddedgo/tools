// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

func xgen(f string) {
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, f, nil, parser.ParseComments)
	checkErr(err)
	pkg := a.Name.Name
	var constraints []string
	for _, cg := range a.Comments {
		for len(cg.List) > 0 {
			c := strings.TrimLeft(cg.List[0].Text, "/*")
			c = strings.TrimSpace(c)
			if strings.HasPrefix(c, "go:build") {
				constraints = append(constraints, c)
			} else if strings.HasPrefix(c, "Peripheral:") ||
				strings.HasPrefix(c, "Instances:") {

				periph(pkg, f, cg.Text(), a.Decls, constraints)
				return
			}
			cg.List = cg.List[1:]
			continue
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		die("xgen FILE1.go FILE2.go ...")
	}

	for _, f := range os.Args[1:] {
		if !strings.HasSuffix(f, ".go") {
			fmt.Fprintln(os.Stderr, "ignoring:", f)
			continue
		}
		xgen(f)
	}
}
