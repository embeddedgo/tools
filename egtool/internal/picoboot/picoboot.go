// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package picoboot

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"time"
	"unsafe"

	"github.com/embeddedgo/tools/egtool/internal/util"
	usb "github.com/google/gousb"
)

const magic uint32 = 0x431fd10b

const (
	cmdExclusiveAccess uint8 = 0x01
	cmdReboot          uint8 = 0x02
	cmdFlashErase      uint8 = 0x03
	cmdRead            uint8 = 0x04 | 0x80
	cmdWrite           uint8 = 0x05
	cmdExitXIP         uint8 = 0x06
	cmdEnterXIP        uint8 = 0x07
	cmdExec            uint8 = 0x08
	cmdVectorizeFlash  uint8 = 0x09
	cmdReboot2         uint8 = 0x0a
	cmdGetInfo         uint8 = 0x0b | 0x80
	cmdOTPRead         uint8 = 0x0c | 0x80
	cmdOTPWrite        uint8 = 0x0d
)

type Conn struct {
	ctx       *usb.Context
	dev       *usb.Device
	iid       uint16
	oe        *usb.OutEndpoint
	ie        *usb.InEndpoint
	cmdBuf    [32]byte
	token     uint32
	readSpec  [2]uint32
	writeSpec [2]uint32
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

	ctx, devs, err := util.OpenUSB(0x2e8a, 0x000f, busAddr)
	if err != nil {
		return
	}
	defer ctx.Close()
	if len(devs) == 0 {
		return nil, errors.New("no USB devices in BOOTSEL mode were found")
	}
	if len(devs) != 1 {
		return nil, errors.New("found more than one USB device in BOOTSEL mode")
	}

	dev := devs[0]
	var cn, in, an int
	ok := false
	for _, cfg := range dev.Desc.Configs {
		for _, id := range cfg.Interfaces {
			for _, is := range id.AltSettings {
				if is.Class == 0xff && is.SubClass == 0 && is.Protocol == 0 {
					cn, in, an = cfg.Number, id.Number, is.Alternate
					ok = true
					break
				}
			}
		}
	}
	if !ok {
		return nil, fmt.Errorf(
			"the found device %d:%d doesn't provide the expected interface",
			dev.Desc.Bus, dev.Desc.Address,
		)
	}
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
	conn = &Conn{ctx: ctx, dev: dev, iid: uint16(in), oe: oe, ie: ie}
	binary.LittleEndian.AppendUint32(conn.cmdBuf[:0], magic)
	return
}

func (c *Conn) Close() (err error) {
	err = c.ctx.Close()
	wrapErr("Close", &err)
	return
}

func (c *Conn) writeCmd(cmdId uint8, transferLength int, args any) error {
	c.token++
	cmdSize := 0
	if args != nil {
		cmdSize = binary.Size(args)
	}
	if uint(cmdSize) > 16 {
		return errors.New("wrong args size")
	}
	le := binary.LittleEndian
	buf := c.cmdBuf[:4] // persistent magic number
	buf = le.AppendUint32(buf, c.token)
	buf = append(buf, cmdId, uint8(cmdSize))
	buf = buf[:len(buf)+2] // reserved field
	buf = le.AppendUint32(buf, uint32(transferLength))
	if cmdSize != 0 {
		buf, _ = binary.Append(buf, le, args)
	}
	n := len(buf)
	buf = buf[:cap(buf)]
	clear(buf[n:]) // padd with zeros
	_, err := c.oe.Write(buf)
	return err
}

func (c *Conn) ExclusiveAccess(ea bool) (err error) {
	defer wrapErrStatus(c, "ExclusiveAccess", &err)
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

func (c *Conn) FlashErase(addr uint32, size int) (err error) {
	defer wrapErrStatus(c, "FlashErase", &err)
	err = c.writeCmd(cmdFlashErase, 0, &[2]uint32{addr, uint32(size)})
	if err != nil {
		return
	}
	_, err = c.ie.Read(nil)
	return
}

func (c *Conn) SetReadAddr(addr uint32) {
	c.readSpec[0] = addr
}

func (c *Conn) ReadAddr() uint32 {
	return c.readSpec[0]
}

func (c *Conn) SetWriteAddr(addr uint32) {
	c.writeSpec[0] = addr
}

func (c *Conn) WriteAddr() uint32 {
	return c.writeSpec[0]
}

// Read performs n-byte PICOBOOT read transaction (if err == nil then n is
// always equal to len(p)) starting just after the last read address (see also
// SetReadAddr).
func (c *Conn) Read(p []byte) (n int, err error) {
	defer wrapErrStatus(c, "Read", &err)
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
	defer wrapErrStatus(c, "Write", &err)
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

func (c *Conn) ExitXIP() (err error) {
	defer wrapErrStatus(c, "ExitXIP", &err)
	err = c.writeCmd(cmdExitXIP, 0, nil)
	if err != nil {
		return
	}
	_, err = c.ie.Read(nil)
	return
}

const (
	// Reboot2 types
	RebootNormal      uint32 = 0x0
	RebootBootsel     uint32 = 0x2
	RebootRAMImage    uint32 = 0x3
	RebootFlashUpdate uint32 = 0x4
	RebootPCSP        uint32 = 0xd

	// Optional flags that can be ORed to the type
	RebootToARM   uint32 = 1 << 4
	RebootToRISCV uint32 = 1 << 5
)

func (c *Conn) Reboot2(rebootType uint32, delay time.Duration, p0, p1 uint32) (err error) {
	defer wrapErrStatus(c, "Reboot2", &err)
	a := [4]uint32{
		rebootType,
		uint32(delay / time.Millisecond),
		p0, p1,
	}
	err = c.writeCmd(cmdReboot2, 0, &a)
	if err != nil {
		return
	}
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

func (c *Conn) GetInfo(info []uint32, args ...uint32) (err error) {
	defer wrapErrStatus(c, "GetInfo", &err)
	nbytes := len(info) * 4
	var a [4]uint32
	copy(a[:], args)
	err = c.writeCmd(cmdGetInfo, nbytes, &a)
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

// Token returns a token associated to the last command.
func (c *Conn) Token() uint32 {
	return c.token
}

var cmdStr = []string{
	cmdExclusiveAccess: "ExclusiveAccess",
	cmdReboot:          "Reboot",
	cmdFlashErase:      "FlashErase",
	cmdRead &^ 0x80:    "Read",
	cmdWrite:           "Write",
	cmdExitXIP:         "ExitXIP",
	cmdEnterXIP:        "EnterXIP",
	cmdExec:            "Exec",
	cmdVectorizeFlash:  "VectorizeFlash",
	cmdReboot2:         "Reboot2",
	cmdGetInfo &^ 0x80: "GetInfo",
	cmdOTPRead &^ 0x80: "OTPRead",
	cmdOTPWrite:        "OTPWrite",
}

var statusStr = [...]string{
	1:  "unknown cmd",
	2:  "invalid cmd lenght",
	3:  "invalid transfer lenght",
	4:  "invalid address",
	5:  "bad alignment",
	6:  "interleaved write",
	7:  "rebooting",
	8:  "unknown error",
	9:  "invalid state",
	10: "not permitted",
	11: "invalid arg",
	12: "buffer too small",
	13: "precondition not met",
	14: "modified data",
	15: "invalid data",
	16: "not found",
	17: "unsupported modification",
}

type StatusError struct {
	Cmd    string
	Status string
}

func (e *StatusError) Error() string {
	return e.Cmd + " status: " + e.Status
}

func wrapErrStatus(c *Conn, op string, err *error) {
	if *err == nil {
		return
	}
	_, _, err1 := c.GetCommandStatus()
	if _, ok := err1.(*StatusError); ok {
		*err = err1
		c.InterfaceReset() // best effort
		return
	}
	*err = &Error{op, *err}
}

const (
	ctrlInterfaceReset    uint8 = 0x41
	ctrlGetCmommandStatus uint8 = 0x42
)

func (c *Conn) InterfaceReset() (err error) {
	_, err = c.dev.Control(
		usb.ControlVendor|usb.ControlInterface,
		ctrlInterfaceReset, 0, c.iid, nil,
	)
	wrapErr("InterfaceReset", &err)
	return
}

func (c *Conn) GetCommandStatus() (token uint32, done bool, err error) {
	buf := c.cmdBuf[16:32]
	_, err = c.dev.Control(
		usb.ControlVendor|usb.ControlInterface|usb.ControlIn,
		ctrlGetCmommandStatus, 0, c.iid, buf,
	)
	if err != nil {
		wrapErr("GetCommandStatus", &err)
		return
	}
	le := binary.LittleEndian
	token = le.Uint32(buf[0:])
	statusId := le.Uint32(buf[4:])
	cmdId := buf[8]
	done = buf[9] == 0
	if statusId != 0 {
		cmd := "unknown"
		if cmdExclusiveAccess <= cmdId && cmdId <= cmdOTPWrite {
			cmd = cmdStr[cmdId]
		}
		status := "unknown"
		if 1 <= statusId && statusId <= 17 {
			status = statusStr[statusId]
		}
		err = &StatusError{cmd, status}
	}
	return
}
