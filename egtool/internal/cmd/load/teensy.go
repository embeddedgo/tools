// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package load

import (
	"bytes"
	"time"

	"github.com/embeddedgo/tools/egtool/internal/imxmbr"
	"github.com/embeddedgo/tools/egtool/internal/util"
	usb "github.com/google/gousb"
)

func teensy(elf, busAddr string, quiet bool) {
	ctx, devs, err := util.OpenUSB(0x16C0, 0x0478, busAddr)
	util.FatalErr("", err)
	defer ctx.Close()
	if len(devs) == 0 {
		util.Fatal("no USB devices in the bootloader mode were found")
	}
	if len(devs) != 1 {
		util.Fatal("found more than one USB device in the bootloader mode")
	}
	dev := devs[0]
	dev.SetAutoDetach(true)
	cfg, err := dev.Config(1)
	util.FatalErr("", err)
	defer cfg.Close()
	ifa, err := cfg.Interface(0, 0)
	util.FatalErr("", err)
	defer ifa.Close()

	sections, err := util.ReadELF(elf)
	util.FatalErr("", err)

	const (
		flashSize  = 16 * 1024 * 1024 // TODO: determine the actual size
		flexRAMCfg = 0x5555_5556      // 480 KiB OCRAM, 32 KiB DTCM
	)
	mbr := imxmbr.Make(flashSize, 0, flexRAMCfg)
	sections = append(sections, &util.Section{Paddr: 0x6000_0000, Data: mbr})

	img := bytes.NewBuffer(make([]byte, 0, sections.Size()*5/4))
	const pad = 0xff
	_, err = sections.Flatten(img, pad)
	util.FatalErr("", err)

	addr := uint32(sections[0].Paddr)
	if addr != 0x6000_0000 {
		// For now we don't support partial loadings.
		util.Fatal("the load address must be 0x6000_0000")
	}

	// Teensy 4.x constants
	const (
		blockSize  = 1024
		blockAlign = blockSize - 1
	)
	imgSize := (img.Len() + blockAlign) &^ blockAlign
	img.Write(util.PadBytes(nil, imgSize-img.Len(), pad))
	var buf [64 + blockSize]byte

	// Load
	cnt := 0
	for addr := 0; addr < imgSize; addr += blockSize {
		if !quiet {
			util.Progress("Loading:", addr, imgSize, 1024, "KiB")
		}
		img.Read(buf[64:])
		if addr != 0 {
			for _, b := range buf[64:] {
				if b != pad {
					goto write
				}
			}
			continue
		}
	write:
		buf[0] = byte(addr)
		buf[1] = byte(addr >> 8)
		buf[2] = byte(addr >> 16)
		util.FatalErr("", teensyWrite(dev, buf[:]))
		cnt++
	}
	if !quiet {
		util.Progress("Loaded: ", imgSize, imgSize, 1024, "KiB")
	}

	// Boot
	buf[0] = 0xff
	buf[1] = 0xff
	buf[2] = 0xff
	clear(buf[64:])
	util.FatalErr("", teensyWrite(dev, buf[:]))
}

func teensyWrite(dev *usb.Device, buf []byte) (err error) {
	for range 500 {
		_, err = dev.Control(0x21, 9, 0x0200, 0, buf)
		if err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	return
}
