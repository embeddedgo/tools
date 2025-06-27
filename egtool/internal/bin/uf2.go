// Copyright 2024 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bin

import (
	"encoding/binary"
	"io"
)

const (
	uf2NotMainFlash         = 0x00000001
	uf2FileContainer        = 0x00001000
	uf2FamilyIDPresent      = 0x00002000
	uf2MD5ChecksumPresent   = 0x00004000
	uf2ExtensionTagsPresent = 0x00008000
)

var uf2FamilyMap = map[string]uint32{
	"rp2040":        0xe48bff56,
	"absolute":      0xe48bff57,
	"data":          0xe48bff58,
	"rp2350_arm_s":  0xe48bff59,
	"rp2350_riscv":  0xe48bff5a,
	"rp2350_arm_ns": 0xe48bff5b,
}

type uf2block struct {
	Magic0 uint32
	Magic1 uint32
	Flags  uint32
	Addr   uint32
	Len    uint32
	Seq    uint32
	Total  uint32
	Family uint32
	Data   [256]byte
	_      [476 - 256]byte
	Magic2 uint32
}

type uf2Writer struct {
	w io.Writer
	b uf2block
}

func newUF2Writer(w io.Writer, addr, flags, family uint32, size int) *uf2Writer {
	u := new(uf2Writer)
	u.w = w
	u.b.Magic0 = 0x0a324655
	u.b.Magic1 = 0x9e5d5157
	u.b.Flags = flags
	u.b.Addr = addr
	u.b.Total = uint32((size + len(u.b.Data) - 1) / len(u.b.Data))
	u.b.Family = family
	u.b.Magic2 = 0x0ab16f30
	return u
}

func (u *uf2Writer) Write(p []byte) (n int, err error) {
	b := &u.b
	for len(p) != 0 {
		m := copy(b.Data[b.Len:], p)
		n += m
		p = p[m:]
		b.Len += uint32(m)
		if int(b.Len) == len(b.Data) {
			err = binary.Write(u.w, binary.LittleEndian, b)
			if err != nil {
				return
			}
			b.Addr += b.Len
			b.Seq++
			b.Len = 0
		}
	}
	return
}

func (u *uf2Writer) Flush() (err error) {
	b := &u.b
	if b.Len == 0 {
		return
	}
	clear(b.Data[b.Len:])
	b.Len = uint32(len(b.Data))
	err = binary.Write(u.w, binary.LittleEndian, b)
	b.Addr += b.Len
	b.Seq++
	b.Len = 0
	return
}
