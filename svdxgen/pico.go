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
		g.Name = strings.ReplaceAll(g.Name, "_", "")
		for _, p := range g.Periphs {
			p.Name = strings.ReplaceAll(p.Name, "_", "")
			picocommon(p)
			switch p.Name {
			case "clocks":
				picoclocks(p)
			case "iobank":
				picoiobank(p)
			case "padsbank":
				picopadsbank(p)
			case "resets":
				picoresets(p)
			case "sha":
				picosha(p)
			case "sio":
				picosio(p)
			case "xosc":
				picoxosc(p)
			case "pllsys":
				picopllsys(p)
			case "otpdata", "otpdataraw", "padsqspi", "qmi", "usbdpram":
				p.Insts = nil
			}
		}
	}
}

func picocommon(p *Periph) {
	for _, r := range p.Regs {
		if len(r.Bits) == 1 {
			// Untype the integer registers, that is the registers
			// that have only one bitfield started from bit 0, without values.
			bf := r.Bits[0]
			if bits.TrailingZeros64(bf.Mask) == 0 && len(bf.Values) == 0 {
				//r.Type = "uint" + strconv.FormatUint(uint64(r.BitSiz), 10)
				r.Bits = nil
				continue
			}
		}
		// Upper-case identifiers.
		for _, bf := range r.Bits {
			bf.Name = strings.ToUpper(bf.Name)
			for _, v := range bf.Values {
				v.Name = strings.ToUpper(v.Name)
			}
		}
	}
}

func picoclocks(p *Periph) {
	var gpout *Reg
	for i, r := range p.Regs {
		switch {
		case r.Name == "CLK_GPOUT0_CTRL":
			ctrl := *r
			ctrl.Name = "CTRL"
			gpout = r
			gpout.Name = "GPOUT"
			gpout.Len = 1
			gpout.SubRegs = append(gpout.SubRegs, &ctrl)
		case r.Name == "CLK_GPOUT0_DIV":
			r.Name = "DIV"
			gpout.SubRegs = append(gpout.SubRegs, r)
			p.Regs[i] = nil
		case r.Name == "CLK_GPOUT0_SELECTED":
			r.Name = "SELECTED"
			gpout.SubRegs = append(gpout.SubRegs, r)
			p.Regs[i] = nil
		case strings.HasPrefix(r.Name, "CLK_GPOUT"):
			p.Regs[i] = nil
			if strings.HasSuffix(r.Name, "_CTRL") {
				gpout.Len++
			}
		case strings.HasPrefix(r.Name, "CLK_"):
			r.Name = r.Name[4:]
			if strings.HasSuffix(r.Name, "_CTRL") {
				for _, bf := range r.Bits {
					prefix := r.Name[:len(r.Name)-4]
					if bf.Name == "SRC" {
						bf.Name = prefix + "SOURCE"
					} else {
						bf.Name = prefix + bf.Name
					}
					for _, v := range bf.Values {
						v.Name = prefix + v.Name
					}
				}
			}
		}
		switch r.Name {
		case "SYS_DIV":
			r.Type = "DIV"
			r.Bits = nil
		case "FC0_RESULT":
			r.Type = "uint32"
			for _, bf := range r.Bits {
				bf.Name = "FC0_RESULT_" + bf.Name
			}
		case "DFTCLK_XOSC_CTRL":
			r.Name = "DFT" + r.Name[6:]
			r.Type = "DFT_OSC_CTRL"
			src := r.Bits[0]
			src.Name = "CLKSRC"
			src.Values[0].Name = "CLKSRC_NULL"
			src.Values[1].Name = "CLKSRC_PLL_PRIMARY"
			src.Values[2].Name = "CLKSRC_GPIN"
		case "DFTCLK_ROSC_CTRL", "DFTCLK_LPOSC_CTRL":
			r.Name = "DFT" + r.Name[6:]
			r.Type = "DFT_OSC_CTRL"
			r.Bits = nil
		case "FC0_SRC":
			bf := r.Bits[0]
			bf.Name = "FC0_SOURCE"
			for _, v := range bf.Values {
				v.Name = "FC0_" + v.Name
			}
		case "WAKE_EN0":
			r.Type = "CLK0"
			for _, bf := range r.Bits {
				bf.Name = strings.TrimPrefix(bf.Name, "CLK_")
			}
		case "SLEEP_EN0", "ENABLED0":
			r.Type = "CLK0"
			r.Bits = nil
		case "WAKE_EN1":
			r.Type = "CLK1"
			for _, bf := range r.Bits {
				bf.Name = strings.TrimPrefix(bf.Name, "CLK_")
			}
		case "SLEEP_EN1", "ENABLED1":
			r.Type = "CLK1"
			r.Bits = nil
		}
	}
}

func picoresets(p *Periph) {
	for _, r := range p.Regs {
		switch r.Name {
		case "RESET":
			r.Type = "uint32"
		case "WDSEL", "RESET_DONE":
			r.Type = "uint32"
			r.Bits = nil
		}
	}
}

func picoiobank(p *Periph) {
	var gpio, intr, p0inte, p0intf, p0ints, p1inte, p1intf, p1ints, dinte, dintf, dints *Reg
	for i, r := range p.Regs {
		switch {
		case r.Name == "GPIO0_STATUS":
			status := *r
			status.Name = "STATUS"
			gpio = r
			gpio.Name = "GPIO"
			gpio.Len = 1
			gpio.SubRegs = append(gpio.SubRegs, &status)
		case r.Name == "GPIO0_CTRL":
			r.Name = "CTRL"
			gpio.SubRegs = append(gpio.SubRegs, r)
			p.Regs[i] = nil
			for _, bf := range r.Bits {
				switch {
				case bf.Name == "FUNCSEL":
					v := bf.Values
					v[0].Name = "F0"
					v[1].Name = "F1_SPI"
					v[2].Name = "F2_UART"
					v[3].Name = "F3_I2C"
					v[4].Name = "F4_PWM"
					v[5].Name = "F5_SIO"
					v[6].Name = "F6_PIO0"
					v[7].Name = "F7_PIO1"
					v[8].Name = "F8_PIO2"
					v[9].Name = "F9"
					v[10].Name = "F10_USB"
				case strings.HasSuffix(bf.Name, "OVER"):
					prefix := bf.Name[:len(bf.Name)-4] + "_"
					for _, v := range bf.Values {
						v.Name = prefix + v.Name
					}
				}
			}
		case strings.HasPrefix(r.Name, "GPIO"):
			if strings.HasSuffix(r.Name, "_STATUS") {
				gpio.Len++
			}
			p.Regs[i] = nil
		case strings.HasPrefix(r.Name, "IRQSUMMARY_"):
			n := len(r.Name)
			if r.Name[n-1] == '0' {
				r.Name = r.Name[:n-1]
				r.Len = 2
				r.Bits = nil
			} else {
				p.Regs[i] = nil
			}
		case strings.HasPrefix(r.Name, "INTR"):
			n := len(r.Name)
			if r.Name[n-1] != '0' {
				p.Regs[i] = nil
				intr.Len++
				break
			}
			r.Name = r.Name[:n-1]
			r.Len = 1
			r.Bits = nil
			intr = r
		case strings.HasPrefix(r.Name, "PROC0_INTE"):
			n := len(r.Name)
			if r.Name[n-1] != '0' {
				p.Regs[i] = nil
				p0inte.Len++
				break
			}
			r.Name = r.Name[:n-1]
			r.Len = 1
			r.Bits = nil
			p0inte = r
		case strings.HasPrefix(r.Name, "PROC0_INTF"):
			n := len(r.Name)
			if r.Name[n-1] != '0' {
				p.Regs[i] = nil
				p0intf.Len++
				break
			}
			r.Name = r.Name[:n-1]
			r.Len = 1
			r.Bits = nil
			p0intf = r
		case strings.HasPrefix(r.Name, "PROC0_INTS"):
			n := len(r.Name)
			if r.Name[n-1] != '0' {
				p.Regs[i] = nil
				p0ints.Len++
				break
			}
			r.Name = r.Name[:n-1]
			r.Len = 1
			r.Bits = nil
			p0ints = r
		case strings.HasPrefix(r.Name, "PROC1_INTE"):
			n := len(r.Name)
			if r.Name[n-1] != '0' {
				p.Regs[i] = nil
				p1inte.Len++
				break
			}
			r.Name = r.Name[:n-1]
			r.Len = 1
			r.Bits = nil
			p1inte = r
		case strings.HasPrefix(r.Name, "PROC1_INTF"):
			n := len(r.Name)
			if r.Name[n-1] != '0' {
				p.Regs[i] = nil
				p1intf.Len++
				break
			}
			r.Name = r.Name[:n-1]
			r.Len = 1
			r.Bits = nil
			p1intf = r
		case strings.HasPrefix(r.Name, "PROC1_INTS"):
			n := len(r.Name)
			if r.Name[n-1] != '0' {
				p.Regs[i] = nil
				p1ints.Len++
				break
			}
			r.Name = r.Name[:n-1]
			r.Len = 1
			r.Bits = nil
			p1ints = r
		case strings.HasPrefix(r.Name, "DORMANT_WAKE_INTE"):
			n := len(r.Name)
			if r.Name[n-1] != '0' {
				p.Regs[i] = nil
				dinte.Len++
				break
			}
			r.Name = r.Name[:n-1]
			r.Len = 1
			r.Bits = nil
			dinte = r
		case strings.HasPrefix(r.Name, "DORMANT_WAKE_INTF"):
			n := len(r.Name)
			if r.Name[n-1] != '0' {
				p.Regs[i] = nil
				dintf.Len++
				break
			}
			r.Name = r.Name[:n-1]
			r.Len = 1
			r.Bits = nil
			dintf = r
		case strings.HasPrefix(r.Name, "DORMANT_WAKE_INTS"):
			n := len(r.Name)
			if r.Name[n-1] != '0' {
				p.Regs[i] = nil
				dints.Len++
				break
			}
			r.Name = r.Name[:n-1]
			r.Len = 1
			r.Bits = nil
			dints = r
		}
	}
}

func picopllsys(p *Periph) {
	p.Name = "pll"
	for _, inst := range p.Insts {
		inst.Name = strings.TrimPrefix(inst.Name, "PLL_")
	}
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
	var spin *Reg
	for i, r := range p.Regs {
		switch {
		case strings.HasPrefix(r.Name, "GPIO_HI"):
			r.Type = "uint32"
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
				spin = r
				spin.Len = 1
			} else {
				spin.Len++
				p.Regs[i] = nil
			}
		}
	}
}

func picopadsbank(p *Periph) {
	var gpio *Reg
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
				gpio.Len++
				break
			}
			gpio = r
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
			for _, bf := range r.Bits {
				for _, v := range bf.Values {
					v.Name += "_VAL"
				}
			}
		}
	}
}
