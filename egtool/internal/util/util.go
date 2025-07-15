// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

// InOutFiles infers the name of the input and output files from the name of the
// current working directory if the inName is an empty strings.
func InOutFiles(inName, inSuffix, outName, outSuffix string) (string, string) {
	if inName == "" {
		inName = DirName() + inSuffix
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
	pbuf = append(pbuf, pdone[:done+2]...)
	pbuf = append(pbuf, ptodo[done:]...)
	pbuf = strconv.AppendInt(pbuf, int64(cur/scale), 10)
	pbuf = append(pbuf, ' ')
	pbuf = append(pbuf, post...)
	if cur == max {
		pbuf = append(pbuf, '\n')
	}
	os.Stderr.Write(pbuf)

}
