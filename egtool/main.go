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
	"github.com/embeddedgo/tools/egtool/internal/uf2"
)

type tool struct {
	descr string
	main  func(args []string)
}

var tools = map[string]tool{
	"bin": {bin.Descr, bin.Main},
	"hex": {hex.Descr, hex.Main},
	"uf2": {uf2.Descr, uf2.Main},
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
		fmt.Fprintf(uw, "  %*s  %s\n", maxLen, name, tools[name].descr)
	}
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" {
		printToolList()
		return
	}
	tool, ok := tools[os.Args[1]]
	if !ok {
		printToolList()
		os.Exit(1)
	}
	tool.main(os.Args[1:])
}
