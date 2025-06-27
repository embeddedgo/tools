// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hex

import (
	"flag"
	"fmt"
	"os"

	"github.com/embeddedgo/tools/egtool/internal/util"
	"github.com/marcinbor85/gohex"
)

const shortDescr = "convert an ELF file to the Intel HEX format"

func Main(args []string) {
	if len(args) == 0 {
		fmt.Println(shortDescr)
		return
	}
	fs := flag.NewFlagSet(args[0], flag.ExitOnError)
	fs.Usage = func() {
		os.Stderr.WriteString("Usage:\n  bin [OPTIONS] [ELF [HEX]]\nOptions:\n")
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
	elf, hex := util.InOutFiles(fs.Arg(0), ".elf", fs.Arg(1), ".hex")
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
	mem := gohex.NewMemory()
	for _, s := range sections {
		mem.AddBinary(uint32(s.Paddr), s.Data)
	}
	w, err := os.Create(hex)
	util.FatalErr("", err)
	defer w.Close()
	err = mem.DumpIntelHex(w, 16)
	util.FatalErr("dumpintelhex", err)
}
