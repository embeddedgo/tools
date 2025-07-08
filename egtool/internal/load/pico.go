// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package load

import (
	"encoding/binary"
	"fmt"

	"github.com/embeddedgo/tools/egtool/internal/util"
	usb "github.com/google/gousb"
)

const magic uint32 = 0x431fd10b

const (
	cmdExclusiveAccess uint8 = 0x01
	cmdReboot          uint8 = 0x02
	cmdFlashErase      uint8 = 0x03
	cmdRead            uint8 = 0x84
	cmdWrite           uint8 = 0x05
	cmdExitXIP         uint8 = 0x06
	cmdEnterXIP        uint8 = 0x07
	cmdExec            uint8 = 0x08
	cmdVectorizeFlash  uint8 = 0x09
	cmdReboot2         uint8 = 0x0a
	cmdGetInfo         uint8 = 0x8b
	cmdOTPRead         uint8 = 0x8c
	cmdOTPWrite        uint8 = 0x0d
)

const (
	ctrInterfaceReset   uint8 = 0x41
	ctrGetCommandStatus uint8 = 0x42
)

type command struct {
	magic          uint32 // 0x431fd10b
	token          uint32 // user provided token to identify this request by
	cmdId          uint8  // ID of the command (top bit indicates direction)
	cmdSize        uint8  // number of bytes of valid data in the args field
	_              uint16 // 0x0000
	transferLength uint32 // number of bytes the host expects to transfer
}

type getInfoArgs struct {
	typ     uint8
	param8  uint8
	param16 uint16
	params  [3]uint32
}

type cmdWriter struct {
	oe    *usb.OutEndpoint
	token uint32
	buf   [32]byte
}

func (w *cmdWriter) WriteCmd(cmdId uint8, transferLength uint32, args any) error {
	cmdSize := binary.Size(args)
	if cmdSize > 255 {
		panic("binary.Size(args) > 255")
	}
	cmd := command{
		magic:          magic,
		token:          w.token,
		cmdId:          cmdId,
		cmdSize:        uint8(cmdSize),
		transferLength: transferLength,
	}
	w.token++
	buf := w.buf[:0]
	buf, err := binary.Append(buf, binary.LittleEndian, &cmd)
	if err != nil {
		panic(err)
	}
	if args != nil {
		buf, err = binary.Append(buf, binary.LittleEndian, args)
		if err != nil {
			panic(err)
		}
	}
	if &buf[0] != &w.buf[0] {
		panic("unexpected allocation")
	}
	clear(w.buf[len(buf):]) // padd with zeros
	_, err = w.oe.Write(w.buf[:])
	return err
}

func pico(elf string) {
	ctx := usb.NewContext()
	defer ctx.Close()

	// Determine the USB device .
	var cn, in, an int
	devs, err := ctx.OpenDevices(func(desc *usb.DeviceDesc) bool {
		if desc.Vendor == 0x2e8a && desc.Product == 0x000f {
			for _, cfg := range desc.Configs {
				for _, id := range cfg.Interfaces {
					for _, is := range id.AltSettings {
						if is.Class == 0xff && is.SubClass == 0 && is.Protocol == 0 {
							cn, in, an = cfg.Number, id.Number, is.Alternate
							return true
						}
					}
				}
			}
		}
		return false
	})
	util.FatalErr("usb", err)
	if len(devs) == 0 {
		util.Fatal("no device in the boot mode is connected to the USB")
	}
	if len(devs) > 1 {
		util.Fatal("more than one device in the boot mode is connected to the USB")
	}
	dev := devs[0]

	// Determine both bulk endpoints (TX/RX).
	dev.SetAutoDetach(true)
	cfg, err := dev.Config(cn)
	util.FatalErr("usb", err)
	intf, err := cfg.Interface(in, an)
	util.FatalErr("usb", err)
	var rxn, txn int
	if n := len(intf.Setting.Endpoints); n != 2 {
		util.Fatal("exactly two USB endpoints expected but found %d", n)
	}
	for _, ed := range intf.Setting.Endpoints {
		if ed.Direction == usb.EndpointDirectionIn {
			rxn = ed.Number
		} else {
			txn = ed.Number
		}
	}
	if rxn == 0 {
		util.Fatal("there is no IN endpoint in the USB interface")
	}
	if txn == 0 {
		util.Fatal("there is no OUT endpoint in the USB interface")
	}
	ie, err := intf.InEndpoint(rxn)
	util.FatalErr("usb", err)
	oe, err := intf.OutEndpoint(txn)
	util.FatalErr("usb", err)

	w := &cmdWriter{oe: oe}
	buf := make([]byte, 256)
	le := binary.LittleEndian

	err = w.WriteCmd(cmdExclusiveAccess, 0, uint8(1))
	util.FatalErr("usb", err)
	_, err = ie.Read(nil)
	util.FatalErr("usb", err)

	err = w.WriteCmd(cmdRead, 4, [2]uint32{0x00000010, 4})
	util.FatalErr("usb", err)
	n, err := ie.Read(buf)
	util.FatalErr("usb", err)
	if n != 4 {
		util.Fatal("bad length")
	}
	util.FatalErr("usb", err)
	_, err = oe.Write(nil)

	devType := "unknown"
	switch le.Uint32(buf) & 0xffffff {
	case 0x01754d:
		devType = "RP2040"
	case 0x02754d:
		devType = "RP2350"
	}
	fmt.Printf("device:    %s\n", devType)

	err = w.WriteCmd(
		cmdGetInfo, 6*4,
		&getInfoArgs{typ: 1, params: [3]uint32{1}},
	)
	util.FatalErr("usb", err)

	n, err = ie.Read(buf)
	util.FatalErr("usb", err)
	_, err = oe.Write(nil)
	util.FatalErr("usb", err)
	if n != 6*4 {
		util.Fatal("usb: read: %d != 6*4", n)
	}

	var resp struct {
		N        uint32 // number of words (4)
		Flag     uint32 // the flag to which the information relates (1)
		Package  uint32
		DeviceID uint32
		WaferID  uint32
	}
	binary.Decode(buf[:n], le, &resp)
	fmt.Printf("package:   %#x\n", resp.Package)
	fmt.Printf("device id: %#x\n", resp.DeviceID)
	fmt.Printf("wafer id:  %#x\n", resp.WaferID)

}
