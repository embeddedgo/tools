// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bin

import (
	"bytes"
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/embeddedgo/tools/egtool/internal/util"
)

const (
	DescrBin = "convert an ELF file to a binary image"
	DescrUF2 = "convert an ELF file to the UF2 format"
)

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
	pad := fs.Uint(
		"pad", 0xff,
		"pad `byte` used to fill gaps between sections",
	)
	var family string
	if cmd == "uf2" {
		fs.StringVar(
			&family, "family", "",
			"UF2 family `ID` (32-bit number) or a known family name:\n"+
				strings.Join(slices.Sorted(maps.Keys(uf2FamilyMap)), "\n"),
		)
	}
	fs.Parse(args)
	if fs.NArg() > 2 {
		fs.Usage()
		os.Exit(1)
	}
	elf, out := util.InOutFiles(fs.Arg(0), ".elf", fs.Arg(1), "."+cmd)
	sections, err := util.ReadELF(elf)
	util.FatalErr("readelf", err)
	if *inc != "" {
		isec, err := util.ReadBins(*inc)
		util.FatalErr("readbins", err)
		sections = append(sections, isec...)
	}
	switch cmd {
	case "bin":
		of, err := os.Create(out)
		util.FatalErr("", err)
		defer of.Close()
		_, err = sections.Flatten(of, byte(*pad))
		util.FatalErr("flatten", err)
	case "uf2":
		familyID, ok := uf2FamilyMap[family]
		if !ok {
			u, err := strconv.ParseUint(family, 0, 32)
			if err != nil {
				util.Fatal(`uf2: bad family ID: "%s"`, family)
			}
			familyID = uint32(u)
		}
		buf := bytes.NewBuffer(make([]byte, 0, sections.Size()*5/4))
		_, err = sections.Flatten(buf, byte(*pad))
		util.FatalErr("flatten", err)
		addr := uint32(sections[0].Paddr)
		if uint64(addr) != sections[0].Paddr {
			util.Fatal("uf2: the target address %#x doesn't fit in 32 bits")
		}
		of, err := os.Create(out)
		util.FatalErr("", err)
		defer of.Close()
		w := newUF2Writer(of, addr, uf2FamilyIDPresent, familyID, buf.Len())
		_, err = w.Write(buf.Bytes())
		util.FatalErr("", err)
		util.FatalErr("", w.Flush())
	}
}
