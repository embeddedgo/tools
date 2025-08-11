// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package load

import (
	"bytes"
	"os"

	"github.com/embeddedgo/tools/egtool/internal/dfu"
	"github.com/embeddedgo/tools/egtool/internal/util"
	usb "github.com/google/gousb"
)

func dfuDev(target string, elf, busAddr string, quiet bool) {
	var (
		vendor, product usb.ID
		blkId           uint16
		blkSize         int
		poolSpeed       uint = 1
	)
	switch target {
	case "stm32":
		vendor, product = 0x0483, 0xdf11
		blkId, blkSize = 2, 1024
		poolSpeed = 64
		// FIXME: blkSize = 2048 doesn't work and gousb doesn't provide access
		// to the extra descriptors where the DFU functional one provides the
		// wTransferSize
	}
	conn, err := dfu.Connect(vendor, product, busAddr, poolSpeed)
	util.FatalErr("", err)
	defer conn.Close()

	sections, err := util.ReadELF(elf)
	util.FatalErr("", err)
	img := bytes.NewBuffer(make([]byte, 0, sections.Size()*5/4))
	const pad = 0xff
	_, err = sections.Flatten(img, pad)
	util.FatalErr("", err)

	if target == "stm32" {
		os.Stderr.WriteString("Erasing flash... ")
		massErase := [1]byte{0x41}
		err = conn.Download(0, massErase[:])
		util.FatalErr("", err)
		os.Stderr.WriteString("done\n")
	}

	imgBytes, imgSize := img.Bytes(), img.Len()

	for i := 0; i < imgSize; i += blkSize {
		if !quiet {
			util.Progress("Loading:", i, imgSize, 1024, "KiB")
		}
		blk := imgBytes[i:]
		if len(blk) > blkSize {
			blk = blk[:blkSize]
		}
		err = conn.Download(blkId, blk)
		util.FatalErr("", err)
		blkId++
	}
	err = conn.Download(blkId, nil)
	if target != "stm32" {
		util.FatalErr("", err)
	}
	if !quiet {
		util.Progress("Loaded: ", imgSize, imgSize, 1024, "KiB")
	}

}
