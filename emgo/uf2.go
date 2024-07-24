// Copyright 2024 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/binary"
	"io"
	"unsafe"
)

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

type UF2Writer struct {
	w io.Writer
	b uf2block
}

func NewUF2Writer(w io.Writer, addr, flags, family uint32, size int) *UF2Writer {
	u := new(UF2Writer)
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

func (u *UF2Writer) WriteString(s string) (n int, err error) {
	b := &u.b
	for len(s) != 0 {
		m := copy(b.Data[b.Len:], s)
		n += m
		s = s[m:]
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

func (u *UF2Writer) Write(p []byte) (n int, err error) {
	return u.WriteString(*(*string)(unsafe.Pointer(&p)))
}

func (u *UF2Writer) Flush() (err error) {
	b := &u.b
	if b.Len == 0 {
		return
	}
	clear(b.Data[b.Len:])
	err = binary.Write(u.w, binary.LittleEndian, b)
	b.Addr += b.Len
	b.Seq++
	b.Len = 0
	return
}
