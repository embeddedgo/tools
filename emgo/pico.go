// Copyright 2024 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"os"
)

func picoImage(obj, format string, buf *bytes.Buffer) {
	bin := buf.Bytes()
	switch format {
	case "bin":
		f, err := os.Create(obj + ".bin")
		dieErr(err)
		_, err = f.Write(bin)
		dieErr(err)
		dieErr(f.Close())
	case "uf2":
		f, err := os.Create(obj + ".uf2")
		dieErr(err)
		w := NewUF2Writer(f, 0x1000_0000, UF2FamilyIDPresent,
			uf2_rp2350_arm_s, len(bin))
		_, err = w.Write(bin)
		dieErr(err)
		dieErr(w.Flush())
		dieErr(f.Close())
	default:
		dieFormat(format, "rp2350")
	}
}
