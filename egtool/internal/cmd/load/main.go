// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package load

import (
	"flag"
	"fmt"
	"os"

	"github.com/embeddedgo/tools/egtool/internal/util"
)

const Descr = "load the program / memory range stored in a file onto the device"

func Main(cmd string, args []string) {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage:\n  %s [OPTIONS] [ELF]\nOptions:\n",
			cmd,
		)
		fs.PrintDefaults()
	}
	target := fs.String(
		"target", "auto",
		"select the target device and transport:\n"+
			"auto: try to determine the target device automatically\n"+
			"pico: RP2350 (aka Raspberry Pi Pico 2) via USB PICOBOOT\n",
	)
	quiet := fs.Bool("quiet", false, "do not print diagnostic information")
	fs.Parse(args)
	if fs.NArg() > 1 {
		fs.Usage()
		os.Exit(1)
	}
	elf := fs.Arg(0)
	if elf == "" {
		elf = util.Module() + ".elf"
	}
	if *target == "auto" {
		*target = auto(elf)
		if *target == "" {
			util.Fatal("cannot determine the target by reading %s", elf)
		}
	}
	switch *target {
	case "pico":
		pico(elf, *quiet)
	default:
		util.Fatal("unknown target: %s", *target)
	}
}
