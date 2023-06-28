// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unicode"
)

func warn(info ...interface{}) {
	fmt.Fprintln(os.Stderr, info...)
}

func die(info ...interface{}) {
	fmt.Fprintln(os.Stderr, info...)
	os.Exit(1)
}

func dieErr(err error) {
	if err != nil {
		die(err)
	}
}

// cwc wraps io.WriteCloser, terminates program on any error
type cwc struct {
	c io.WriteCloser
}

func create(path string) cwc {
	f, err := os.Create(path)
	dieErr(err)
	return cwc{f}
}

func (w cwc) Write(b []byte) (int, error) {
	n, err := w.c.Write(b)
	dieErr(err)
	return n, nil
}

func (w cwc) WriteString(s string) (int, error) {
	n, err := io.WriteString(w.c, s)
	dieErr(err)
	return n, nil
}

func (w cwc) Close() error {
	var name string
	if f, ok := w.c.(*os.File); ok {
		name = f.Name()
	}
	dieErr(w.c.Close())
	if name != "" {
		name, err := filepath.Abs(name)
		dieErr(err)
		gofmt := exec.Command("gofmt", "-w", name)
		gofmt.Stdout = os.Stdout
		gofmt.Stderr = os.Stderr
		dieErr(gofmt.Run())
	}
	return nil
}

func (w cwc) donotedit() {
	io.WriteString(
		w,
		"// DO NOT EDIT THIS FILE. GENERATED BY svdxgen.\n\n",
	)
}

func mkdir(path string) {
	err := os.Mkdir(path, 0755)
	if e, ok := err.(*os.PathError); !ok ||
		e.Err != os.ErrExist && e.Err != syscall.EEXIST {
		dieErr(err)
	}
}

func pnameLess(a, b string) bool {
	i := strings.IndexFunc(a, unicode.IsDigit)
	if i < 0 {
		return a < b
	}
	abase := a[:i]
	anum, err := strconv.Atoi(a[i:])
	if err != nil {
		return a < b
	}
	i = strings.IndexFunc(b, unicode.IsDigit)
	if i < 0 {
		return a < b
	}
	bbase := b[:i]
	bnum, err := strconv.Atoi(b[i:])
	if err != nil {
		return a < b
	}
	if abase != bbase {
		return abase < bbase
	}
	return anum < bnum
}

func dropDigits(s string) string {
	return strings.Map(
		func(r rune) rune {
			if unicode.IsDigit(r) {
				return -1
			}
			return r
		},
		s,
	)
}

func fixSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func isdigit(c byte) bool {
	return uint(int(c)-'0') <= 9
}

func prefix(s string, prefixes ...string) int {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return len(p)
		}
	}
	return -1
}

func suffix(s string, suffixes ...string) int {
	for _, p := range suffixes {
		if strings.HasSuffix(s, p) {
			return len(p)
		}
	}
	return -1
}
