// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pioasm

import (
	"fmt"
	"os"

	"github.com/embeddedgo/tools/egtool/internal/pioasm"
	"github.com/embeddedgo/tools/egtool/internal/util"
	"golang.org/x/tools/go/packages"
)

const Descr = "assemble file of PIO program(s) for use in applications."

func Main(cmd string, args []string) {
	if len(args) < 1 || 2 < len(args) {
		fmt.Fprintf(os.Stderr, "Usage:\n  %s INPUT [OUTPUT]\n", cmd)
		os.Exit(1)
	}
	input := args[0]
	output := input + ".go"
	if len(args) > 1 {
		output = args[1]
	}
	util.SetGOENV(false)
	pkgs, err := packages.Load(nil, "")
	util.FatalErr("", err)
	pioasm.Generate(pkgs[0].Name, input, output)
}
