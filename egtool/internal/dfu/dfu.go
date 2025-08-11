// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dfu

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/embeddedgo/tools/egtool/internal/util"
	usb "github.com/google/gousb"
)

type Conn struct {
	ctx       *usb.Context
	dev       *usb.Device
	iid       uint16
	statusBuf [6]byte
	poolSpeed uint
}

type Error struct {
	Op  string
	Err error
}

func (e *Error) Unwrap() error {
	return e.Err
}

func (e *Error) Error() string {
	return "dfu: " + e.Op + ": " + e.Err.Error()
}

func wrapErr(op string, err *error) {
	if *err != nil {
		*err = &Error{op, *err}
	}
}

// Connect connects to the USB device in the DFU mode. You can connect to the
// concrete device on the USB bus by providing BUS:DEV string where both BUS and
// DEV are decimal unsigned integers. If busAddr is empty connect will try to
// find a DFU device on the bus (it will return an error if there are more than
// one such devices).
func Connect(vendor, product usb.ID, busAddr string, poolSpeed uint) (conn *Conn, err error) {
	defer wrapErr("Connect", &err)

	ctx, devs, err := util.OpenUSB(vendor, product, busAddr)
	if err != nil {
		return
	}
	defer ctx.Close()

	var dev *usb.Device
	var alts [][3]int
	for _, d := range devs {
		for _, cfg := range d.Desc.Configs {
			for _, id := range cfg.Interfaces {
				for _, is := range id.AltSettings {
					if is.Class == 0xfe && is.SubClass == 1 && is.Protocol == 2 && len(is.Endpoints) == 0 {
						if dev == nil {
							dev = d
						} else if dev != d {
							err = errors.New("found more than one USB device in DFU mode")
							return
						}
						alts = append(
							alts,
							[3]int{cfg.Number, is.Number, is.Alternate},
						)
					}
				}
			}
		}
	}
	if dev == nil {
		err = errors.New("no USB devices in DFU mode were found")
		return
	}

	var alt *[3]int
	for _, a := range alts {
		var aname string
		aname, err = dev.InterfaceDescription(a[0], a[1], a[2])
		if err != nil {
			return
		}
		if strings.Contains(strings.ToLower(aname), "flash") {
			if alt != nil {
				err = fmt.Errorf(
					"device %d:%d has more than one valid DFU configuration",
					dev.Desc.Bus, dev.Desc.Address,
				)
				return
			}
			alt = &a
		}
	}

	dev.SetAutoDetach(true)
	cfg, err := dev.Config(alt[0])
	if err != nil {
		return
	}
	_, err = cfg.Interface(alt[1], alt[2])
	if err != nil {
		return
	}

	conn = &Conn{ctx: ctx, dev: dev, iid: uint16(alt[1]), poolSpeed: poolSpeed}
	return
}

func (c *Conn) Close() (err error) {
	err = c.ctx.Close()
	wrapErr("Close", &err)
	return
}

var statusStr = [...]string{
	1:  "file is not for this target",
	2:  "file fails a vendor-specific verification test",
	3:  "unable to write memory",
	4:  "memory erase function failed",
	5:  "memory erase check failed",
	6:  "program memory function failed",
	7:  "programmed memory failed verification",
	8:  "memory address is out of range",
	9:  "premature DFU_DNLOAD with wLength = 0",
	10: "firmware is corrupt",
	11: "vendor-specific error",
	12: "unexpected USB reset signaling",
	13: "unexpected power on reset",
	14: "unknown error",
	15: "stalled an unexpected request",
}

// DFU states
const (
	appIdle              uint8 = 0
	appDetach            uint8 = 1
	dfuIdle              uint8 = 2
	dfuDnloadSync        uint8 = 3
	dfuDnbusy            uint8 = 4
	dfuDnloadIdle        uint8 = 5
	dfuManifestSync      uint8 = 6
	dfuManifest          uint8 = 7
	dfuManifestWaitReset uint8 = 8
	dfuUploadIdle        uint8 = 9
	dfuError             uint8 = 10
)

var stateStr = [...]string{
	appIdle:              "app idle",
	appDetach:            "app detach",
	dfuIdle:              "DFU idle",
	dfuDnloadSync:        "DFU download sync",
	dfuDnbusy:            "DFU download busy",
	dfuDnloadIdle:        "DFU download idle",
	dfuManifestSync:      "DFU manifest sync",
	dfuManifest:          "DFU manifest",
	dfuManifestWaitReset: "DFU manifest wait reset",
	dfuUploadIdle:        "DFU upload idle",
	dfuError:             "DFU error",
}

// DFU sequests
const (
	reqDetach    uint8 = 0x00
	reqDnload    uint8 = 0x01
	reqUpload    uint8 = 0x02
	reqGetStatus uint8 = 0x03
	reqClrStatus uint8 = 0x04
	reqGetState  uint8 = 0x05
	reqAbort     uint8 = 0x06
)

func wrapErrStatus(c *Conn, op string, err *error) {
	defer wrapErr(op, err)
	if *err != nil {
		return
	}
again:
	_, *err = c.dev.Control(
		usb.ControlIn|usb.ControlClass|usb.ControlInterface,
		reqGetStatus, 0, c.iid, c.statusBuf[:],
	)
	if *err != nil {
		op += ": GetStatus"
		return
	}
	status := c.statusBuf[0]
	state := c.statusBuf[4]
	if state == dfuError {
		_, *err = c.dev.Control(
			usb.ControlOut|usb.ControlClass|usb.ControlInterface,
			reqClrStatus, 0, c.iid, nil,
		)
		if *err != nil {
			op += ": ClrStatus"
			return
		}
	}
	if status != 0 {
		es := "unknown error"
		if int(status) < len(statusStr) {
			es = statusStr[status]
		}
		*err = errors.New(es)
		return
	}
	if state == dfuDnbusy {
		pollTimeout := uint(c.statusBuf[1]) +
			uint(c.statusBuf[2])<<8 +
			uint(c.statusBuf[3])<<16

		time.Sleep(time.Duration(pollTimeout/c.poolSpeed) * time.Millisecond)
		goto again
	}
}

func (c *Conn) Download(blockNum uint16, p []byte) (err error) {
	_, err = c.dev.Control(
		usb.ControlOut|usb.ControlClass|usb.ControlInterface,
		reqDnload, blockNum, c.iid, p,
	)
	wrapErrStatus(c, "Download", &err)
	return
}
