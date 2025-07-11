// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hex

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/embeddedgo/tools/egtool/internal/util"
	"github.com/marcinbor85/gohex"
)

const Descr = "convert an ELF file to the Intel HEX format"

func Main(cmd string, args []string) {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage:\n  %s [OPTIONS] [ELF [%s]]\nOptions:\n",
			cmd, strings.ToUpper(cmd),
		)
		fs.PrintDefaults()
	}
	inc := fs.String(
		"inc", "",
		"binary files to be included BIN1:ADDR1[,BIN2:ADDR2[,...]]",
	)
	fs.Parse(args)
	if fs.NArg() > 2 {
		fs.Usage()
		os.Exit(1)
	}
	elf, out := util.InOutFiles(fs.Arg(0), ".elf", fs.Arg(1), ".hex")
	sections, err := util.ReadELF(elf)
	util.FatalErr("readelf", err)
	if *inc != "" {
		isec, err := util.ReadBins(*inc)
		util.FatalErr("readbins", err)
		sections = append(sections, isec...)
	}
	sections.SortByPaddr()
	mem := gohex.NewMemory()
	for _, s := range sections {
		mem.AddBinary(uint32(s.Paddr), s.Data)
	}
	of, err := os.Create(out)
	util.FatalErr("", err)
	defer of.Close()
	err = mem.DumpIntelHex(of, 16)
	util.FatalErr("dumpintelhex", err)
}
