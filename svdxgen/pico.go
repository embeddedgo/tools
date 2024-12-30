// Copyright 2024 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"math/bits"
	"strings"
)

func picotweaks(gs []*Group) {
	for _, g := range gs {
		for _, p := range g.Periphs {
			picointreg(p)
			switch p.Name {
			case "pads_bank":
				picopadsbank(p)
			case "sha":
				picosha(p)
			case "sio":
				picosio(p)
			case "xosc":
				picoxosc(p)
			case "pll_sys":
				picopllsys(p)
			case "otp_data", "otp_data_raw", "pads_qspi", "qmi", "usb_dpram":
				p.Insts = nil
			}
		}
	}
}

func picointreg(p *Periph) {
	// Remove bits from the integer registers, that is the registers that have
	// only one bitfield started from bit 0, without values.
	for _, r := range p.Regs {
		if len(r.Bits) == 1 {
			bf := r.Bits[0]
			if bits.TrailingZeros64(bf.Mask) == 0 && len(bf.Values) == 0 {
				r.Bits = nil
			}
		}
	}
}

func picopllsys(p *Periph) {
	for _, r := range p.Regs {
		switch {
		case r.Name == "PRIM":
			r.Type = "uint32"
		}
	}
}

func picosha(p *Periph) {
	for _, r := range p.Regs {
		switch r.Name {
		case "CSR":
			for _, bf := range r.Bits {
				if bf.Name == "DMA_SIZE" {
					bf.Values = nil
				}
			}
		}
	}
}

func picosio(p *Periph) {
	firstGPIOHI := true
	var clbits *[]*BitField
	var spinr *Reg
	for i, r := range p.Regs {
		switch {
		case strings.HasPrefix(r.Name, "GPIO_HI"):
			r.Type = "GPIO_HI"
			if firstGPIOHI {
				firstGPIOHI = false
			} else {
				r.Bits = nil
			}
		case r.Name == "TMDS_CTRL":
			for _, bf := range r.Bits {
				if bf.Name == "PIX_SHIFT" {
					bf.Values = nil
				}
			}
		case strings.Contains(r.Name, "_CTRL_LANE"):
			r.Type = "CTRL_LANE"
			switch r.Name {
			case "INTERP0_CTRL_LANE0":
				r.Bits = r.Bits[:9]
				clbits = &r.Bits
			case "INTERP1_CTRL_LANE0":
				*clbits = append(*clbits, r.Bits[8:]...)
				r.Bits = nil
			default:
				r.Bits = nil
			}
		case strings.HasPrefix(r.Name, "SPINLOCK") && r.Name != "SPINLOCK_ST":
			if r.Name == "SPINLOCK0" {
				spinr = r
				spinr.Len = 1
			} else {
				spinr.Len++
				p.Regs[i] = nil
			}
		}
	}
}

func picopadsbank(p *Periph) {
	var gpior *Reg
	for i, r := range p.Regs {
		switch {
		case r.Name == "VOLTAGE_SELECT":
			for _, bf := range r.Bits {
				if bf.Name == "VOLTAGE_SELECT" {
					bf.Name = "IOVDD"
					for _, b := range bf.Values {
						b.Name = "V" + b.Name
					}
				}
			}
		case strings.HasPrefix(r.Name, "GPIO"):
			if r.Name != "GPIO0" {
				p.Regs[i] = nil
				gpior.Len++
				break
			}
			gpior = r
			r.Len = 1
			r.Name = "GPIO"
			for _, bf := range r.Bits {
				if bf.Name == "DRIVE" {
					for _, b := range bf.Values {
						b.Name = "D" + b.Name
					}
				}
			}
		case r.Name == "SWCLK" || r.Name == "SWD":
			r.Type = "GPIO"
			r.Bits = nil
		}
	}
}

func picoxosc(p *Periph) {
	for _, r := range p.Regs {
		switch r.Name {
		case "CTRL":
			for _, bf := range r.Bits {
				switch bf.Name {
				case "FREQ_RANGE":
					for _, b := range bf.Values {
						b.Name = "FR" + b.Name
					}
				case "ENABLE":
					bf.Name = "EN"
					vals := []*BitFieldValue{
						{"ENABLED", "Oscillator is enabled but not necessarily running and stable, resets to 0", 1},
						{"BADWRITE", "An invalid value has been written to EN or FREQ_RANGE or DORMANT", 1 << 12},
						{"STABLE", "Oscillator is running and stable", 1 << 19},
					}
					bf.Values = append(bf.Values, vals...)
				}
			}
		case "STATUS":
			r.Type = "CTRL"
			r.Bits = nil
		case "DORMANT":
			r.Type = "uint32"
		}
	}
}
