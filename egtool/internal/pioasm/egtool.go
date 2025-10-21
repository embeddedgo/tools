// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pioasm

/*
#include <stdlib.h>

int egtoolPioasmGenerate(const char *pkgName, const char *input, const char *output);

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

func Generate(pkgName, input, output string) int {
	pn := C.CString(pkgName)
	defer C.free(unsafe.Pointer(pn))
	ci := C.CString(input)
	defer C.free(unsafe.Pointer(ci))
	co := C.CString(output)
	defer C.free(unsafe.Pointer(co))
	return int(C.egtoolPioasmGenerate(pn, ci, co))
}
