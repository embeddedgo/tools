// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package load

import (
	"debug/elf"
	"os"

	"github.com/embeddedgo/tools/egtool/internal/util"
)

func auto(name string) string {
	r, err := os.Open(name)
	util.FatalErr("", err)
	defer r.Close()
	f, err := elf.NewFile(r)
	util.FatalErr("read ELF", err)
	defer f.Close()
	syms, err := f.Symbols()
	util.FatalErr("read ELF", err)
	for _, s := range syms {
		if s.Name == "picometa" {
			return "pico"
		}
	}
	return ""
}
