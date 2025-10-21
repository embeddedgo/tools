// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pioasm

import (
	"github.com/embeddedgo/tools/egtool/internal/pioasm"
)

const Descr = "assemble file of PIO program(s) for use in applications."

func Main(cmd string, args []string) {
	pioasm.Main("-h")
}
