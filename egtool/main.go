// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/embeddedgo/tools/egtool/internal/cmd/bin"
	"github.com/embeddedgo/tools/egtool/internal/cmd/build"
	"github.com/embeddedgo/tools/egtool/internal/cmd/hex"
	"github.com/embeddedgo/tools/egtool/internal/cmd/imxmbr"
	"github.com/embeddedgo/tools/egtool/internal/cmd/isrnames"
	"github.com/embeddedgo/tools/egtool/internal/cmd/load"
	"github.com/embeddedgo/tools/egtool/internal/cmd/pioasm"
)

type tool struct {
	descr string
	main  func(cmd string, args []string)
}

var tools = map[string]tool{
	"bin":      {bin.DescrBin, bin.Main},
	"build":    {build.Descr, build.Main},
	"hex":      {hex.Descr, hex.Main},
	"imxmbr":   {imxmbr.Descr, imxmbr.Main},
	"isrnames": {isrnames.Descr, isrnames.Main},
	"load":     {load.Descr, load.Main},
	"pioasm":   {pioasm.Descr, pioasm.Main},
	"uf2":      {bin.DescrUF2, bin.Main},
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
		fmt.Fprintf(uw, "  %-*s  %s\n", maxLen, name, tools[name].descr)
	}
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" {
		printToolList()
		return
	}
	cmd, args := os.Args[1], os.Args[2:]
	tool, ok := tools[cmd]
	if !ok {
		printToolList()
		os.Exit(1)
	}
	tool.main(cmd, args)
}
