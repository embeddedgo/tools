// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package load

import (
	"encoding/binary"
	"fmt"

	"github.com/embeddedgo/tools/egtool/internal/picoboot"
	"github.com/embeddedgo/tools/egtool/internal/util"
)

func pico(elf string) {
	c, err := picoboot.Connect("")
	util.FatalErr("picoboot", err)
	err = c.SetExclusiveAccess(true)
	util.FatalErr("", err)

	buf := make([]byte, 16)
	le := binary.LittleEndian

	c.SetReadAddr(0x0000_0010)
	_, err = c.Read(buf[:4])
	util.FatalErr("", err)

	devType := "unknown"
	devTypeID := le.Uint32(buf) & 0xffffff
	switch devTypeID {
	case 0x01754d:
		devType = "RP2040"
	case 0x02754d:
		devType = "RP2350"
	}
	if devType != "RP2350" {
		util.Fatal("unsupported device type: %s (%#x)\n", devType, devTypeID)
	}

	var info [16]uint32

	err = c.GetInfo([4]uint32{picoboot.Partition, picoboot.PTInfo}, info[:5])
	util.FatalErr("", err)
	if info[0] != 4 || info[1] != picoboot.PTInfo {
		util.Fatal("picoboot: GetInfo: Partition: bad response")
	}
	fmt.Printf("partition count:          %d\n", info[2])
	fmt.Printf("unpartitioned perm&loc:   %#x\n", info[3])
	fmt.Printf("unpartitioned perm&flags: %#x\n", info[4])
}
