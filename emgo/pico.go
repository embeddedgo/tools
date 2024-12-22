// Copyright 2024 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
)

// UF2 families
const (
	uf2_rp2040        = 0xe48bff56
	uf2_absolute      = 0xe48bff57
	uf2_data          = 0xe48bff58
	uf2_rp2350_arm_s  = 0xe48bff59
	uf2_rp2350_riscv  = 0xe48bff5a
	uf2_rp2350_arm_ns = 0xe48bff5b
)

const (
	header = 0xffffded3
	footer = 0xab123579
)

// Item types
const (
	itemImageDef    = 0x42
	itemVectorTable = 0x03
	itemLast        = 0xff
)

// IMAGE_DEF items
const (
	imdImageTypeInvalid = 0 << 0
	imdImageTypeExe     = 1 << 0
	imdImageTypeData    = 2 << 0

	imdExeSecUnspec = 0 << 4
	imdExeSecNS     = 1 << 4
	imdExeSecS      = 2 << 4

	imdExeARM   = 0 << 8
	imdExeRISCV = 1 << 8

	imdExeRP2040 = 0 << 12
	imdExeRP2350 = 1 << 12

	imdExeTBYB = 1 << 15
)

func picoImage(obj, format string, buf *bytes.Buffer) {
	const binHeadLen = 128

	var imd []byte // IMAGE_DEF
	le := binary.LittleEndian
	imd = le.AppendUint32(imd, header)

	imd = append(imd, itemImageDef, 1) // IMAGE_DEF, size = 1w
	imd = le.AppendUint16(imd, imdImageTypeExe|imdExeSecS|imdExeARM|imdExeRP2350)

	imd = append(imd, itemVectorTable, 2, 0, 0)       // VECTOR_TABLE, size = 2w, pad
	imd = le.AppendUint32(imd, 0x10000000+binHeadLen) // Vector table (runtime) address

	imd = append(imd, itemLast)
	imd = le.AppendUint16(imd, uint16((len(imd)-4-1)/4)) // other items' size
	imd = append(imd, 0)                                 // pad

	imd = le.AppendUint32(imd, 0) // link to the next block relative to header

	imd = le.AppendUint32(imd, footer)

	if len(imd) > binHeadLen {
		die("rp2530 IMAGE_DEF size too big")
	}
	pad := make([]byte, binHeadLen-len(imd))
	for i := range pad {
		pad[i] = 0xff
	}

	bin := buf.Bytes()
	var w io.Writer
	switch format {
	case "bin":
		f, err := os.Create(obj + ".bin")
		dieErr(err)
		defer func() { dieErr(f.Close()) }()
		w = f
	case "uf2":
		f, err := os.Create(obj + ".uf2")
		dieErr(err)
		defer func() { dieErr(f.Close()) }()
		uf2 := NewUF2Writer(f, 0x1000_0000, UF2FamilyIDPresent,
			uf2_rp2350_arm_s, len(bin)+binHeadLen)
		defer func() { dieErr(uf2.Flush()) }()
		w = uf2
	default:
		dieFormat(format, "rp2350")
	}
	_, err := w.Write(imd)
	dieErr(err)
	_, err = w.Write(pad)
	dieErr(err)
	_, err = w.Write(bin)
	dieErr(err)
}
