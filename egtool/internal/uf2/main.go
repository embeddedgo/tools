// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uf2

import (
	"flag"
	"fmt"
	"os"

	"github.com/embeddedgo/tools/egtool/internal/util"
)

const Descr = "convert an ELF file to the UF2 format"

func Main(args []string) {
	fs := flag.NewFlagSet(args[0], flag.ExitOnError)
	fs.Usage = func() {
		os.Stderr.WriteString("Usage:\n  uf2 [OPTIONS] [ELF [UF2]]\nOptions:\n")
		fs.PrintDefaults()
	}
	inc := fs.String(
		"inc", "",
		"binary files to be included BIN1:ADDR1[,BIN2:ADDR2[,...]]",
	)
	fs.Parse(args[1:])
	if fs.NArg() > 2 {
		fs.Usage()
		os.Exit(1)
	}
	elf, uf2 := util.InOutFiles(fs.Arg(0), ".elf", fs.Arg(1), ".hex")
	sections, err := util.ReadELF(elf)
	util.FatalErr("readelf", err)
	if *inc != "" {
		isec, err := util.ReadBins(*inc)
		util.FatalErr("readbins", err)
		sections = append(sections, isec...)
	}
	sections.SortByPaddr()
	for i, s := range sections {
		fmt.Printf(
			"%d: Vaddr: %#x Paddr: %#x Offset: %#x DataLen: %d\n",
			i, s.Vaddr, s.Paddr, s.Offset, len(s.Data),
		)
	}

	_ = uf2
}
