// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"errors"
	"strconv"
	"strings"

	usb "github.com/google/gousb"
)

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

func OpenUSB(vendor, product usb.ID, busAddr string) (ctx *usb.Context, devs []*usb.Device, err error) {
	bus, addr := parseBusAddr(busAddr)
	if busAddr != "" && bus < 0 {
		err = errors.New("bad USB device address: " + busAddr)
		return
	}
	ctx = usb.NewContext()
	devs, err = ctx.OpenDevices(func(desc *usb.DeviceDesc) bool {
		if bus >= 0 && (desc.Bus != bus || desc.Address != addr) {
			return false
		}
		if desc.Vendor != vendor || desc.Product != product {
			return false
		}
		return true
	})
	if err != nil {
		ctx.Close()
	}
	return
}
