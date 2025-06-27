// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/embeddedgo/tools/egtool/internal/bin"
	"github.com/embeddedgo/tools/egtool/internal/hex"
)

var tools = map[string]func(args []string){
	"bin": bin.Main,
	"hex": hex.Main,
}

func printToolList() {
	names := slices.Sorted(maps.Keys(tools))
	maxLen := 0
	for _, k := range names {
		if maxLen < len(k) {
			maxLen = len(k)
		}
	}
	uw := os.Stderr
	uw.WriteString("Usage:\n  egtool COMMAND [ARGUMENTS]\n\n")
	uw.WriteString("Available commands:\n")
	for _, name := range names {
		fmt.Fprintf(uw, "  %*s  ", maxLen, name)
		tools[name](nil)
	}
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" {
		printToolList()
		return
	}
	toolMain := tools[os.Args[1]]
	if toolMain == nil {
		printToolList()
		os.Exit(1)
	}
	toolMain(os.Args[1:])
}
