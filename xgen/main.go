// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"
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

var (
	tmpl     *template.Template
	generics bool
)

func main() {
	flag.BoolVar(&generics, "g", false, "use mmio.R* generic types")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "xgen [options] FILE1.go FILE2.go ...")
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
	}

	if generics {
		tmpl = template.Must(template.New("R").Parse(tmplR))
	} else {
		tmpl = template.Must(template.New("U").Parse(tmplU))
	}

	for _, f := range flag.Args() {
		if !strings.HasSuffix(f, ".go") {
			fmt.Fprintln(os.Stderr, "ignoring:", f)
			continue
		}
		xgen(f)
	}
}
