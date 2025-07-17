// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

func Warn(f string, args ...any) {
	fmt.Fprintf(os.Stderr, f+"\n", args...)
}

func Fatal(f string, args ...any) {
	fmt.Fprintf(os.Stderr, f+"\n", args...)
	os.Exit(1)
}

// FatalError prints an error description and exits the program if the
// err != nil.
func FatalErr(what string, err error) {
	if err == nil {
		return
	}
	s := err.Error() + "\n"
	if what != "" {
		s = what + ": " + s
	}
	os.Stderr.WriteString(s)
	os.Exit(1)
}

// DirName returns the last element of the path to the current working
// directory.
func DirName() string {
	dir, err := os.Getwd()
	FatalErr("", err)
	dir = filepath.Base(dir)
	if dir == "/" || dir == "." {
		dir = ""
	}
	return dir
}

func Module() string {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	FatalErr("", err)
	gomod := filepath.Clean(string(bytes.TrimRightFunc(out, unicode.IsSpace)))
	if gomod == "" || gomod == os.DevNull {
		Fatal("go.mod file not found in current directory or any parent directory")
	}
	f, err := os.Open(gomod)
	FatalErr("", err)
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fs := bytes.Fields(sc.Bytes())
		if len(fs) >= 2 && string(fs[0]) == "module" {
			return string(fs[1])
		}
	}
	if err := sc.Err(); err != nil {
		FatalErr("", err)
	}
	Fatal("there is no module directive in " + gomod)
	return ""
}

// InOutFiles infers the name of the input and output files from the name of the
// current working directory if the inName is an empty strings.
func InOutFiles(inName, inSuffix, outName, outSuffix string) (string, string) {
	if inName == "" {
		fs, err := os.Stat("go.mod")
		if err != nil || !fs.Mode().IsRegular() {
			inName = DirName()
		} else {
			inName = Module()
		}
		inName += inSuffix
	}
	if outName == "" {
		outName = strings.TrimSuffix(inName, inSuffix) + outSuffix
	}
	return inName, outName
}

var pbuf = make([]byte, 80)

const (
	ptodo = "                         ] "
	pdone = " [========================="
)

func Progress(pre string, cur, max, scale int, post string) {
	pbuf = pbuf[:0]
	pbuf = append(pbuf, '\r')
	pbuf = append(pbuf, pre...)
	done := 25 * cur / max
	pbuf = append(pbuf, pdone[:2+done]...)
	pbuf = append(pbuf, ptodo[done:]...)
	pbuf = strconv.AppendInt(pbuf, int64(cur/scale), 10)
	pbuf = append(pbuf, ' ')
	pbuf = append(pbuf, post...)
	if cur == max {
		pbuf = append(pbuf, '\n')
	}
	os.Stderr.Write(pbuf)

}
