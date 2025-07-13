// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package load

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/embeddedgo/tools/egtool/internal/picoboot"
	"github.com/embeddedgo/tools/egtool/internal/util"
)

func pico(elf string, quiet bool) {
	pb, err := picoboot.Connect("")
	util.FatalErr("picoboot", err)
	defer pb.Close()
	err = pb.ExclusiveAccess(true)
	util.FatalErr("", err)

	var buf [4]byte
	pb.SetReadAddr(0x0000_0010)
	_, err = pb.Read(buf[:])
	util.FatalErr("", err)

	devTypeStr := "unknown"
	devType := binary.LittleEndian.Uint32(buf[:]) & 0xffffff
	var uf2Type uint32
	switch devType {
	case 0x01754d:
		devTypeStr = "RP2040"
		uf2Type = 0xe48bff56
	case 0x02754d:
		devTypeStr = "RP2350"
		uf2Type = 0xe48bff59 // we support only rp2350_arm_s
	}
	if devTypeStr != "RP2350" {
		util.Fatal("unsupported device type: %s (%#x)\n", devTypeStr, devType)
	}

	// Check partition table
	var info [4]uint32
	err = pb.GetInfo(info[:], picoboot.UF2TargetPartition, uf2Type)
	util.FatalErr("", err)
	if info[0] != 3 {
		util.Fatal("picoboot: GetInfo: UF2TargetPartition: bad response")
	}
	firstSector := info[2] & 0x1fff
	lastSector := info[2] >> 13 & 0x1fff
	if firstSector == 0 && lastSector == 8191 {
		//TODO: whole flash, determine its size
	}

	sections, err := util.ReadELF(elf)
	img := bytes.NewBuffer(make([]byte, 0, sections.Size()*5/4))
	const pad = 0xff
	_, err = sections.Flatten(img, pad)
	addr := uint32(sections[0].Paddr)
	if addr != 0x1000_0000 {
		// For now we don't support partial loadings.
		util.Fatal("the load address must be 0x1000_0000")
	}
	const (
		sectSize  = 4096 // flash sector size
		sectAlign = sectSize - 1
	)
	imgSize := (img.Len() + sectAlign) &^ sectAlign
	img.Write(util.PadBytes(nil, imgSize-img.Len(), pad))
	imgBytes := img.Bytes()
	addr += firstSector * sectSize
	pb.SetWriteAddr(addr)

	util.FatalErr("", pb.ExitXIP()) // nop
	for i := 0; i < imgSize; i += sectSize {
		err = pb.FlashErase(pb.WriteAddr(), sectSize)
		util.FatalErr("", err)
		util.FatalErr("", pb.ExitXIP()) // nop

		_, err = pb.Write(imgBytes[i : i+sectSize])
		util.FatalErr("", err)
		util.FatalErr("", pb.ExitXIP()) // nop

		if !quiet {
			fmt.Print(".")
		}
	}
	if !quiet {
		fmt.Println()
	}

	err = pb.Reboot2(picoboot.RebootNormal, time.Second/2, 0, 0)
	util.FatalErr("", err)
}
