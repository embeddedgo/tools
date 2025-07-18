// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imxmbr

import (
	"flag"
	"fmt"
	"os"

	"github.com/embeddedgo/tools/egtool/internal/imxmbr"
	"github.com/embeddedgo/tools/egtool/internal/util"
)

const Descr = "generate the MBR file for the I.MX RT106x microcontrollers"

func Main(cmd string, args []string) {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage:\n  %s [OPTIONS] MBR_FILE\nOptions:\n",
			cmd,
		)
		fs.PrintDefaults()
	}
	flashSize := fs.Uint("flash", 0, "flash size (KiB)")
	imageSize := fs.Uint(
		"image", 0,
		"program image size (KiB), 0 means all the remaining flash space",
	)
	flexRAMCfg := fs.Uint(
		"flexram", 0,
		"FlexRAM configuration (the value to write to the GPR17)",
	)
	fs.Parse(args)
	if fs.NArg() != 1 {
		fs.Usage()
		os.Exit(1)
	}
	*flashSize *= 1024
	*imageSize *= 1024

	mbr := imxmbr.Make(int(*flashSize), int(*imageSize), uint32(*flexRAMCfg))
	util.FatalErr("", os.WriteFile(fs.Arg(0), mbr, 0o666))
}
