// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pioasm

/*
#include <stdlib.h>

int pioasmMain(int argc, char *argv[]);

*/
import "C"
import "unsafe"

func makeArgs(arg0 string, args []string) []*C.char {
	ret := make([]*C.char, len(args)+1)
	ret[0] = C.CString(arg0)
	for i, s := range args {
		ret[i+1] = C.CString(s)
	}
	return ret
}

func freeArgs(cargs []*C.char) {
	for _, p := range cargs {
		C.free(unsafe.Pointer(p))
	}
}

func Main(args ...string) int {
	if len(args) == 0 {
		return 0
	}
	cargs := makeArgs("pioasm", args)
	defer freeArgs(cargs)
	return int(C.pioasmMain(C.int(len(cargs)), &cargs[0]))
}
