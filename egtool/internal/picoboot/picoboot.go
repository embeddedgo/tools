// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package picoboot

import (
	"encoding/binary"
	"errors"
	"io"
	"strconv"
	"strings"
	"unsafe"

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

type Conn struct {
	usbCtx    *usb.Context
	oe        *usb.OutEndpoint
	ie        *usb.InEndpoint
	cmdBuf    [32]byte
	token     uint32
	readSpec  [2]uint32
	writeSpec [2]uint32
}

func parseBusAddr(busAddr string) (int, int) {
	s := strings.Split(busAddr, ":")
	if len(s) != 2 {
		return -1, -1
	}
	bus, err := strconv.ParseUint(s[0], 10, 8)
	if err != nil {
		return -1, -1
	}
	dev, err := strconv.ParseUint(s[1], 10, 8)
	if err != nil {
		return -1, -1
	}
	return int(bus), int(dev)
}

type Error struct {
	Op  string
	Err error
}

func (e *Error) Unwrap() error {
	return e.Err
}

func (e *Error) Error() string {
	return "picoboot: " + e.Op + ": " + e.Err.Error()
}

func wrapErr(op string, err *error) {
	if *err != nil {
		*err = &Error{op, *err}
	}
}

// Connect connects to the USB device in PICOBOOT mode. You can connect to the
// concrete device on the USB bus by providing BUS:DEV string where both BUS
// and DEV are decimal unsigned integers. If busAddr is empty connect will try
// to find a PICOBOOT device on the bus (it will return an error if there are
// more than one such devices).
func Connect(busAddr string) (conn *Conn, err error) {
	defer wrapErr("Connect", &err)
	bus, addr := parseBusAddr(busAddr)
	if busAddr != "" && bus < 0 {
		return nil, errors.New("bad USB device address: " + busAddr)
	}
	ctx := usb.NewContext()
	var cn, in, an int
	devs, err := ctx.OpenDevices(func(desc *usb.DeviceDesc) bool {
		if bus >= 0 && (desc.Bus != bus || desc.Address != addr) {
			return false
		}
		if desc.Vendor != 0x2e8a || desc.Product != 0x000f {
			return false
		}
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
		return false
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			ctx.Close()
		}
	}()
	if len(devs) == 0 {
		return nil, errors.New("no USB devices in BOOTSEL mode were found")
	}
	if len(devs) != 1 {
		return nil, errors.New("found more than one USB device in BOOTSEL mode")
	}
	dev := devs[0]
	dev.SetAutoDetach(true)

	// Determine PICOBOOT bulk endpoints (TX/RX).
	cfg, err := dev.Config(cn)
	if err != nil {
		return nil, err
	}
	intf, err := cfg.Interface(in, an)
	if err != nil {
		return nil, err
	}
	var rxn, txn int
	if n := len(intf.Setting.Endpoints); n != 2 {
		return nil, errors.New("want exactly two USB bulk endpoints")
	}
	for _, ed := range intf.Setting.Endpoints {
		if ed.Direction == usb.EndpointDirectionIn {
			rxn = ed.Number
		} else {
			txn = ed.Number
		}
	}
	if rxn == 0 {
		return nil, errors.New("no USB IN endpoint in the USB interface")
	}
	if txn == 0 {
		return nil, errors.New("no USB OUT endpoint in the USB interface")
	}
	ie, err := intf.InEndpoint(rxn)
	if err != nil {
		return nil, err
	}
	oe, err := intf.OutEndpoint(txn)
	if err != nil {
		return nil, err
	}
	conn = &Conn{usbCtx: ctx, oe: oe, ie: ie}
	binary.LittleEndian.AppendUint32(conn.cmdBuf[:0], magic)
	return
}

func (c *Conn) Close() (err error) {
	err = c.usbCtx.Close()
	wrapErr("Close", &err)
	return
}

func (c *Conn) writeCmd(cmdId uint8, transferLength int, args any) error {
	cmdSize := binary.Size(args)
	if uint(cmdSize) > 16 {
		return errors.New("wrong args size")
	}
	le := binary.LittleEndian
	buf := c.cmdBuf[:4] // persistent magic number
	buf = le.AppendUint32(buf, c.token)
	buf = append(buf, cmdId, uint8(cmdSize))
	buf = buf[:len(buf)+2] // reserved field
	buf = le.AppendUint32(buf, uint32(transferLength))
	buf, _ = binary.Append(buf, le, args)
	n := len(buf)
	buf = buf[:cap(buf)]
	clear(buf[n:]) // padd with zeros
	_, err := c.oe.Write(buf)
	c.token++
	return err
}

func (c *Conn) SetExclusiveAccess(ea bool) (err error) {
	defer wrapErr("SetExclusiveAccess", &err)
	var arg uint8
	if ea {
		arg = 1
	}
	err = c.writeCmd(cmdExclusiveAccess, 0, &arg)
	if err != nil {
		return
	}
	_, err = c.ie.Read(nil)
	return
}

func (c *Conn) SetReadAddr(addr uint32) {
	c.readSpec[0] = addr
}

func (c *Conn) SetWriteAddr(addr uint32) {
	c.writeSpec[0] = addr
}

// Read performs n-byte PICOBOOT read transaction (if err == nil then n is
// always equal to len(p)) starting just after the last read address (see also
// SetReadAddr).
func (c *Conn) Read(p []byte) (n int, err error) {
	defer wrapErr("Read", &err)
	c.readSpec[1] = uint32(len(p))
	err = c.writeCmd(cmdRead, len(p), &c.readSpec)
	if err != nil {
		return
	}
	n, err = io.ReadFull(c.ie, p)
	if err != nil {
		return
	}
	c.readSpec[0] += uint32(n)
	_, err = c.oe.Write(nil)
	return
}

func (c *Conn) Write(p []byte) (n int, err error) {
	defer wrapErr("Write", &err)
	c.writeSpec[1] = uint32(len(p))
	err = c.writeCmd(cmdWrite, len(p), &c.writeSpec)
	if err != nil {
		return
	}
	n, err = c.oe.Write(p)
	if err != nil {
		return
	}
	c.writeSpec[0] += uint32(n)
	_, err = c.ie.Read(nil)
	return
}

// GetInfo information type (args[0])
const (
	InfoSys            uint32 = 1
	Partition          uint32 = 2
	UF2TargetPartition uint32 = 3
	UF2Status          uint32 = 4
)

// GetInfo InfoSys flags (args[1])
const (
	ChipInfo     uint32 = 1 << 0
	Critical     uint32 = 1 << 1
	CPUInfo      uint32 = 1 << 2
	FlashDevInfo uint32 = 1 << 3
	BootRandom   uint32 = 1 << 4
	BootInfo     uint32 = 1 << 6
)

// GetInfo Partition flgs (args[1])
const (
	PTInfo                    uint32 = 1 << 0
	PartitionLocationAndFlags uint32 = 1 << 4
	PartitionID               uint32 = 1 << 5
	PartitionFamilyIDs        uint32 = 1 << 6
	PartitionName             uint32 = 1 << 7
	SinglePartition           uint32 = 1 << 15
)

func (c *Conn) GetInfo(args [4]uint32, info []uint32) (err error) {
	defer wrapErr("GetInfo", &err)
	nbytes := len(info) * 4
	err = c.writeCmd(cmdGetInfo, nbytes, &args)
	if err != nil {
		return
	}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(&info[0])), nbytes)
	n, err := c.ie.Read(buf)
	if err != nil {
		return
	}
	_, err = c.oe.Write(nil)
	if err != nil {
		return
	}
	if nbytes != n {
		return errors.New("read less data than requested")
	}
	le := binary.LittleEndian
	for i := range info {
		info[i] = le.Uint32(buf[i*4:])
	}
	return
}
