// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bin

import (
	"flag"
	"fmt"
	"os"

	"github.com/embeddedgo/tools/egtool/internal/util"
)

const shortDescr = "convert an ELF file to a binary image"

func Main(args []string) {
	if len(args) == 0 {
		fmt.Println(shortDescr)
		return
	}
	fs := flag.NewFlagSet(args[0], flag.ExitOnError)
	fs.Usage = func() {
		os.Stderr.WriteString("Usage:\n  bin [OPTIONS] [ELF [BIN]]\nOptions:\n")
		fs.PrintDefaults()
	}
	inc := fs.String(
		"inc", "",
		"binary files to be included BIN1:ADDR1[,BIN2:ADDR2[,...]]",
	)
	pad := fs.Uint(
		"pad", 0xff,
		"pad `byte` used to fill gaps between sections",
	)
	fs.Parse(args[1:])
	if fs.NArg() > 2 {
		fs.Usage()
		os.Exit(1)
	}
	elf, bin := util.InOutFiles(fs.Arg(0), ".elf", fs.Arg(1), ".bin")
	sections, err := util.ReadELF(elf)
	util.FatalErr("readelf", err)
	if *inc != "" {
		isec, err := util.ReadBins(*inc)
		util.FatalErr("readbins", err)
		sections = append(sections, isec...)
	}
	/*for i, s := range sections {
		fmt.Printf(
			"%d: Vaddr: %#x Paddr: %#x Offset: %#x DataLen: %d\n",
			i, s.Vaddr, s.Paddr, s.Offset, len(s.Data),
		)
	}*/
	w, err := os.Create(bin)
	util.FatalErr("", err)
	defer w.Close()
	_, err = sections.Flatten(w, byte(*pad))
	util.FatalErr("flatten", err)
}
