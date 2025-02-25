// Copyright 2024 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
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
			case "dma":
				picodma(p)
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
			case "spi":
				picospi(p)
			case "pllsys":
				picopllsys(p)
			case "qmi":
				picoqmi(p)
			case "ticks":
				picoticks(p)
			case "uart":
				picouart(p)
			case "xosc":
				picoxosc(p)
			case "otpdata", "otpdataraw", "padsqspi", "usbdpram":
				p.Insts = nil
			}
		}
	}
}

func picocommon(p *Periph) {
	for _, inst := range p.Insts {
		if inst.Name == "DMA" {
			inst.Name = "DMA0"
		}
		for _, irq := range inst.IRQs {
			irq.Name = strings.Replace(irq.Name, "_IRQ", "", 1)
			if strings.HasPrefix(irq.Name, "DMA_") {
				irq.Name = "DMA0" + irq.Name[3:]
			}
		}
	}
	for _, r := range p.Regs {
		if len(r.Bits) == 1 {
			// Untype the integer registers, that is the registers
			// that have only one bitfield started from bit 0, without values.
			bf := r.Bits[0]
			if bf.LSL == 0 && len(bf.Values) == 0 {
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

func picospi(p *Periph) {
	for _, r := range p.Regs {
		r.Name = strings.TrimPrefix(r.Name, "SSP")
		r.Descr = strings.ReplaceAll(r.Descr, "SSP", "")
		switch r.Name {
		case "IMSC":
			r.Type = "INT"
			for _, bf := range r.Bits {
				bf.Name = strings.TrimSuffix(bf.Name, "M")
				if i := strings.Index(bf.Descr, " mask: "); i > 0 {
					bf.Descr = bf.Descr[:i] + "."
				}
			}
		case "RIS", "MIS", "ICR":
			r.Type = "INT"
			r.Bits = nil
		}
	}
}

func picodma(p *Periph) {
	var (
		ch, chdbg, irq, mpu Reg
		tim, secch, secirq  *Reg
	)
	for i, r := range p.Regs {
		switch {
		case strings.HasPrefix(r.Name, "CH"):
			p.Regs[i] = nil
			if strings.Contains(r.Name, "_DBG_") {
				if strings.HasPrefix(r.Name, "CH0_") {
					r.Name = r.Name[4:]
					if r.Name == "DBG_CTDREQ" {
						chdbg = *r
						chdbg.Name = "CH_DBG"
						chdbg.Descr = "Channel debug registers"
						chdbg.Len = 1
						p.Regs[i] = &chdbg
					}
					chdbg.SubRegs = append(chdbg.SubRegs, r)
				} else if strings.HasSuffix(r.Name, "DBG_CTDREQ") {
					chdbg.Len++
				}
			} else {
				if strings.HasPrefix(r.Name, "CH0_") {
					r.Name = r.Name[4:]
					switch {
					case r.Name == "READ_ADDR":
						ch = *r
						ch.Name = "CH"
						ch.Descr = "Channel status and control registers"
						ch.Len = 1
						p.Regs[i] = &ch
					case strings.Contains(r.Name, "TRANS_COUNT") && r.Name != "TRANS_COUNT":
						r.Type = "TRANS_COUNT"
					case strings.Contains(r.Name, "CTRL"):
						r.Type = "CTRL"
					}
					ch.SubRegs = append(ch.SubRegs, r)
				} else if strings.HasSuffix(r.Name, "_TRANS_COUNT_TRIG") {
					ch.Len++
				}
			}
		case strings.HasPrefix(r.Name, "INT"):
			p.Regs[i] = nil
			if r.Name == "INTR" {
				r.Name = "INTR0"
			}
			if strings.HasSuffix(r.Name, "0") {
				r.Name = r.Name[3 : len(r.Name)-1]
				if r.Name == "R" {
					irq = *r
					irq.Name = "INT"
					irq.Descr = "Interrupt status and control registers"
					irq.Len = 1
					p.Regs[i] = &irq
				}
				irq.SubRegs = append(irq.SubRegs, r)
			} else if strings.HasPrefix(r.Name, "INTE") {
				irq.Len++
			}
		case strings.HasPrefix(r.Name, "TIMER"):
			if r.Name == "TIMER0" {
				tim = r
				r.Name = "TIMER"
				r.Len = 1
			} else {
				tim.Len++
				p.Regs[i] = nil
			}
		case strings.HasPrefix(r.Name, "SECCFG_CH"):
			if r.Name == "SECCFG_CH0" {
				secch = r
				r.Name = "SECCFG_CH"
				r.Len = 1
			} else {
				secch.Len++
				p.Regs[i] = nil
			}
			for _, bf := range r.Bits {
				switch bf.Name {
				case "P":
					bf.Name = "PRI_CH"
				case "S":
					bf.Name = "SEC_CH"
				}
			}
		case strings.HasPrefix(r.Name, "SECCFG_IRQ"):
			if r.Name == "SECCFG_IRQ0" {
				secirq = r
				r.Name = "SECCFG_IRQ"
				r.Len = 1
			} else {
				secirq.Len++
				p.Regs[i] = nil
			}
			for _, bf := range r.Bits {
				switch bf.Name {
				case "P":
					bf.Name = "PRI_IRQ"
				case "S":
					bf.Name = "SEC_IRQ"
				}
			}
		case r.Name == "MPU_CTRL":
			for _, bf := range r.Bits {
				switch bf.Name {
				case "P":
					bf.Name = "DEF_PRI"
				case "S":
					bf.Name = "DEF_SEC"
				}
			}
		case strings.HasPrefix(r.Name, "MPU_"):
			p.Regs[i] = nil
			if strings.HasSuffix(r.Name, "0") {
				r.Name = r.Name[4 : len(r.Name)-1]
				if r.Name == "BAR" {
					mpu = *r
					mpu.Name = "MPU"
					mpu.Descr = "DMA Memory Protection Unit registers"
					mpu.Len = 1
					p.Regs[i] = &mpu
					r.Bits = nil
				} else if r.Name == "LAR" {
					for _, bf := range r.Bits {
						switch bf.Name {
						case "EN":
							bf.Name = "REN"
						case "P":
							bf.Name = "RPRI"
						case "S":
							bf.Name = "RSEC"
						}
					}
				}
				mpu.SubRegs = append(mpu.SubRegs, r)
			} else if strings.HasPrefix(r.Name, "MPU_BAR") {
				mpu.Len++
			}
		case r.Name == "SNIFF_CTRL":
			for _, bf := range r.Bits {
				switch bf.Name {
				case "EN":
					bf.Name = "SEN"
				case "BSWAP":
					bf.Name = "SBSWAP"
				}
			}
		}
	}
}

func picouart(p *Periph) {
	for _, r := range p.Regs {
		r.Descr = strings.TrimSuffix(r.Descr, ", "+r.Name)
		r.Name = strings.TrimPrefix(r.Name, "UART")
		switch r.Name {
		case "DR":
			r.Bits = nil
			r.Descr = "Data register (bits: 0-7 data, 8-11 Rx error, see RSR)"
		case "RSR":
			r.Descr = strings.TrimSuffix(r.Descr, ", UARTRSR/UARTECR")
		case "IMSC", "RIS", "MIS", "ICR":
			r.Type = "INT"
			if r.Name != "RIS" {
				r.Bits = nil
				break
			}
			for _, bf := range r.Bits {
				name := strings.TrimSuffix(bf.Name, "RMIS")
				if name != bf.Name {
					name += "MI"
				} else {
					name = strings.TrimRight(bf.Name, "RIS") + "I"
				}
				bf.Name = name
				if i := strings.Index(bf.Descr, " status."); i > 0 {
					bf.Descr = bf.Descr[:i] + "."
				}
			}
		}
	}
}

func picoqmi(p *Periph) {
	var m, atrans *Reg
	for i, r := range p.Regs {
		switch r.Name {
		case "DIRECT_TX", "M0_RFMT":
			for _, bf := range r.Bits {
				if strings.HasSuffix(bf.Name, "_WIDTH") {
					prefix := bf.Name[:len(bf.Name)-5]
					for _, v := range bf.Values {
						v.Name = prefix + v.Name
					}
				}
			}
		}
		switch r.Name {
		case "M0_TIMING":
			m = r
			tim := *r
			tim.Name = "TIMING"
			m.Name = "M"
			m.Descr = "Configuration register for memory address windows"
			m.Len = 1
			m.SubRegs = append(m.SubRegs, &tim)
			for _, bf := range tim.Bits {
				if bf.Name == "PAGEBREAK" {
					for _, v := range bf.Values {
						v.Name = "PB_" + v.Name
					}
				}
			}
		case "M0_RFMT", "M0_WFMT", "M0_RCMD", "M0_WCMD":
			r.Name = r.Name[3:]
			r.Type = r.Name[1:]
			m.SubRegs = append(m.SubRegs, r)
			p.Regs[i] = nil
			if r.Name == "WFMT" || r.Name == "WCMD" {
				r.Bits = nil
				break
			}
			if r.Name == "RCMD" {
				break
			}
			for _, bf := range r.Bits {
				switch bf.Name {
				case "PREFIX_LEN", "SUFFIX_LEN", "DUMMY_LEN":
					for _, v := range bf.Values {
						v.Name = bf.Name[:4] + "_" + v.Name
					}
				}
			}
		case "ATRANS0":
			atrans = r
			atrans.Name = "ATRANS"
			atrans.Len = 1
		case "DIRECT_CSR":
			for _, bf := range r.Bits {
				switch bf.Name {
				case "CLKDIV", "RXDELAY":
					bf.Name = "D" + bf.Name
				}
			}
		default:
			switch {
			case strings.HasSuffix(r.Name, "_TIMING"):
				m.Len++
				p.Regs[i] = nil
			case strings.HasSuffix(r.Name, "_RFMT") ||
				strings.HasSuffix(r.Name, "_RCMD") ||
				strings.HasSuffix(r.Name, "_WFMT") ||
				strings.HasSuffix(r.Name, "_WCMD"):
				p.Regs[i] = nil
			case strings.HasPrefix(r.Name, "ATRANS"):
				atrans.Len++
				p.Regs[i] = nil
			}
		}
	}
}

func picoticks(p *Periph) {
	// Add non-register constants at the beginning of p.Regs.
	p.Regs = append([]*Reg{{Type: "int"}}, p.Regs...)
	tickIDs := p.Regs[0]

	var tick *Reg
	for i, r := range p.Regs {
		if i == 0 {
			continue
		}
		_ctrl := strings.HasSuffix(r.Name, "_CTRL")
		if _ctrl {
			name := r.Name[:len(r.Name)-5]
			tickIDs.Bits = append(tickIDs.Bits, &BitField{
				Name:  name,
				Mask:  uint64(i / 3),
				Descr: "Index to the " + name + " register in the T array",
			})
		}
		switch {
		case r.Name == "PROC0_CTRL":
			tick = r
			ctrl := *r
			ctrl.Name = "CTRL"
			tick.Name = "T"
			tick.Len = 1
			tick.SubRegs = append(tick.SubRegs, &ctrl)
		case r.Name == "PROC0_CYCLES":
			r.Name = "CYCLES"
			tick.SubRegs = append(tick.SubRegs, r)
			p.Regs[i] = nil
		case r.Name == "PROC0_COUNT":
			r.Name = "COUNT"
			tick.SubRegs = append(tick.SubRegs, r)
			p.Regs[i] = nil
		default:
			p.Regs[i] = nil
			if _ctrl {
				tick.Len++
			}
		}
	}
}

func picoclocks(p *Periph) {
	// Add non-register constants at the beginning of p.Regs.
	p.Regs = append([]*Reg{{Type: "int"}}, p.Regs...)
	clkIDs := p.Regs[0]

	var clk *Reg
	for i, r := range p.Regs {
		if i == 0 {
			continue
		}
		if strings.HasPrefix(r.Name, "CLK_") {
			r.Name = r.Name[4:]
			if strings.HasPrefix(r.Name, "SYS_RESUS_") {
				continue
			}
			k := strings.IndexByte(r.Name, '_')
			prefix := r.Name[:k+1]
			_ctrl := strings.HasSuffix(r.Name, "_CTRL")
			_div := strings.HasSuffix(r.Name, "_DIV")
			if _ctrl || _div {
				if strings.HasPrefix(prefix, "GPOUT") {
					if prefix == "GPOUT0_" {
						prefix = "GPOUT_"
					} else {
						r.Bits = nil
					}
				}
				for _, bf := range r.Bits {
					bf.Name = prefix + bf.Name
					for _, v := range bf.Values {
						v.Name = prefix + v.Name
					}
				}
			} else {
				prefix = prefix[:len(prefix)-1]
				clkIDs.Bits = append(clkIDs.Bits, &BitField{
					Name:  prefix,
					Mask:  uint64(clk.Len - 1),
					Descr: "Index to the " + prefix + " register in the CLK array",
				})
			}
			switch {
			case r.Name == "GPOUT0_CTRL":
				clk = r
				ctrl := *r
				ctrl.Name = "CTRL"
				clk.Name = "CLK"
				clk.Len = 1
				clk.SubRegs = append(clk.SubRegs, &ctrl)
			case r.Name == "GPOUT0_DIV":
				r.Name = "DIV"
				clk.SubRegs = append(clk.SubRegs, r)
				p.Regs[i] = nil
			case r.Name == "GPOUT0_SELECTED":
				r.Name = "SELECTED"
				clk.SubRegs = append(clk.SubRegs, r)
				p.Regs[i] = nil
			case _ctrl:
				clk.Len++
				clk.SubRegs[0].Bits = append(clk.SubRegs[0].Bits, r.Bits...)
				p.Regs[i] = nil
			case _div:
				clk.SubRegs[1].Bits = append(clk.SubRegs[1].Bits, r.Bits...)
				p.Regs[i] = nil
			default:
				clk.SubRegs[2].Bits = append(clk.SubRegs[2].Bits, r.Bits...)
				p.Regs[i] = nil
			}
			continue
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
			r.Type = "CLK0_EN"
			for _, bf := range r.Bits {
				bf.Name = strings.TrimPrefix(bf.Name, "CLK_") + "_EN"
			}
		case "SLEEP_EN0", "ENABLED0":
			r.Type = "CLK0_EN"
			r.Bits = nil
		case "WAKE_EN1":
			r.Type = "CLK_EN1"
			for _, bf := range r.Bits {
				bf.Name = strings.TrimPrefix(bf.Name, "CLK_") + "_EN"
			}
		case "SLEEP_EN1", "ENABLED1":
			r.Type = "CLK1_EN"
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
