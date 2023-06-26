// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"math/bits"
	"strings"
)

func imxrttweaks(gs []*Group) {
	for _, g := range gs {
		for _, p := range g.Periphs {
			imxrtonebit(p)
			switch p.Name {
			case "aipstz":
				imxrtaipstz(p)
			case "ccm":
				imxrtccm(p)
			case "ccm_analog":
				imxrtccmanalog(p)
			case "dma":
				imxrtdma(p)
			case "gpio":
				imxrtgpio(p)
			case "iomuxc":
				imxrtiomuxc(p)
			case "lpuart":
				imxrtlpuart(p)
			case "usb":
				imxrtusb(p)
			case "usbphy":
				imxrtusbphy(p)
			case "wdog":
				imxrtwdog(p)
			case "aoi", "lcdif", "usb_analog", "tmr", "enet", "tsc", "pxp", "pmu", "nvic":
				p.Insts = nil
			}
		}
	}
}

func imxrtonebit(p *Periph) {
	for _, r := range p.Regs {
		for _, bf := range r.Bits {
			if bf.Mask>>bits.TrailingZeros64(bf.Mask) != 1 {
				continue
			}
			if p.Name == "iomuxc" && bf.Name == "DAISY" {
				continue
			}
			bf.Values = nil
		}
	}
}

func imxrtaipstz(p *Periph) {
	var opacr *Reg
	for i, r := range p.Regs {
		switch {
		case r.Name == "MPR":
			for _, bf := range r.Bits {
				bf.Values = nil
			}
		case strings.HasPrefix(r.Name, "OPACR"):
			if r.Name == "OPACR" {
				opacr = r
				opacr.Len = 1
				for _, bf := range r.Bits {
					bf.Values = nil
				}
			} else {
				p.Regs[i] = nil
				opacr.Len++
			}
		}
	}
}

func imxrtgpio(p *Periph) {
	for _, r := range p.Regs {
		switch r.Name {
		case "DR", "GDIR", "PSR", "IMR", "ISR", "EDGE_SEL", "DR_SET", "DR_CLEAR", "DR_TOGGLE":
			r.Bits = nil
		case "ICR1", "ICR2":
			for _, bf := range r.Bits {
				var n string
				if strings.HasPrefix(bf.Name, "ICR") {
					n = bf.Name[3:]
					bf.Name = "IC" + n
				}
				if strings.HasPrefix(bf.Descr, "ICR") {
					bf.Descr = "Configuration for GPIO interrupt " + n
				}
				for _, v := range bf.Values {
					if v == nil {
						continue
					}
					v.Name = strings.TrimSuffix(v.Name, "_LEVEL")
					v.Name = strings.TrimSuffix(v.Name, "_EDGE")
					v.Name = bf.Name + "_" + v.Name
					v.Descr = "Interrupt " + n + v.Descr[11:]
				}
			}
		}
	}
}

func imxrtiomuxc(p *Periph) {
	firstMux, firstPad := true, true
	for _, r := range p.Regs {
		switch {
		case strings.HasPrefix(r.Name, "SW_MUX_CTL_PAD_GPIO_"):
			r.Type = "SW_MUX_CTL"
			if firstMux {
				bf := r.Bits[0]
				bf.Mask = 0xf
				bf.Values = []*BitFieldValue{
					{"ALT0", "Select ALT0 mux mode", 0},
					{"ALT1", "Select ALT1 mux mode", 1},
					{"ALT2", "Select ALT2 mux mode", 2},
					{"ALT3", "Select ALT3 mux mode", 3},
					{"ALT4", "Select ALT4 mux mode", 4},
					{"ALT5", "Select ALT5 mux mode", 5},
					{"ALT6", "Select ALT6 mux mode", 6},
					{"ALT7", "Select ALT7 mux mode", 7},
					{"ALT8", "Select ALT8 mux mode", 8},
					{"ALT9", "Select ALT9 mux mode", 9},
				}
				firstMux = false
			} else {
				r.Bits = nil
			}
		case strings.HasPrefix(r.Name, "SW_PAD_CTL_PAD_GPIO_"):
			r.Type = "SW_PAD_CTL"
			if firstPad {
				for _, bf := range r.Bits {
					if bf.Name == "SPEED" {
						for _, v := range bf.Values {
							if v.Value == 2 {
								v.Name = "SPEED_2_fast_150MHz"
							}
						}
					}
				}
				firstPad = false
			} else {
				r.Bits = nil
			}
		case strings.HasSuffix(r.Name, "_SELECT_INPUT"):
			for _, bf := range r.Bits {
				if bf.Name == "DAISY" {
					bf.Name = r.Name[:len(r.Name)-12] + "DAISY"
				}
			}
		case strings.HasSuffix(r.Name, "_SELECT_INPUT_0") || strings.HasSuffix(r.Name, "_SELECT_INPUT_1"):
			for _, bf := range r.Bits {
				if bf.Name == "DAISY" {
					rn := r.Name
					bf.Name = rn[:len(rn)-14] + "DAISY" + rn[len(rn)-2:]
				}
			}
		}
	}
}

func imxrtccm(p *Periph) {
	firstCI := true
	for _, r := range p.Regs {
		switch r.Name {
		case "CSCMR1", "CSCMR2", "CS1CDR", "CS2CDR", "CDCDR", "CSCDR1", "CSCDR2":
			for _, bf := range r.Bits {
				if strings.HasSuffix(bf.Name, "_PODF") || strings.HasSuffix(bf.Name, "_PRED") {
					for _, v := range bf.Values {
						if strings.HasPrefix(v.Name, "DIVIDE_") {
							v.Name = bf.Name + v.Name[6:]
						}
					}
				}
			}
		case "CISR", "CIMR":
			r.Type = "CIR"
			if firstCI {
				for _, bf := range r.Bits {
					bf.Name = "INT_" + bf.Name
				}
				firstCI = false
			} else {
				r.Bits = nil
			}
		case "CLPCR":
			for _, bf := range r.Bits {
				if bf.Name == "LPM" {
					for _, v := range bf.Values {
						switch v.Name {
						case "LPM_0":
							v.Name = "LPM_RUN"
						case "LPM_1":
							v.Name = "LPM_WAIT"
						case "LPM_2":
							v.Name = "LPM_STOP"
						}
					}
					break
				}
			}
		default:
			if strings.HasPrefix(r.Name, "CCGR") {
				for _, bf := range r.Bits {
					bf.Name = "CG" + r.Name[4:] + "_" + bf.Name[2:]
				}
			}
		}
	}
}

func imxrtccmanalog(p *Periph) {
	firstUSB, firstAV, firstPFD := true, true, true
	for _, r := range p.Regs {
		switch {
		case strings.HasPrefix(r.Name, "PLL_USB"):
			r.Type = "PLL_USB"
			if firstUSB {
				firstUSB = false
			} else {
				r.Bits = nil
			}
		case strings.HasPrefix(r.Name, "PLL_AUDIO"), strings.HasPrefix(r.Name, "PLL_VIDEO"):
			r.Type = "PLL_AV"
			if firstAV {
				firstAV = false
			} else {
				r.Bits = nil
			}
		case strings.HasPrefix(r.Name, "PFD"):
			r.Type = "PFD"
			if firstPFD {
				firstPFD = false
			} else {
				r.Bits = nil
			}
		case r.Name == "MISC2":
			for _, bf := range r.Bits {
				if strings.HasSuffix(bf.Name, "_STEP_TIME") {
					for _, v := range bf.Values {
						v.Name = bf.Name[:len(bf.Name)-4] + v.Name
					}
				}
			}
		}
		if len(r.Name) > 4 {
			n := len(r.Name) - 4
			switch r.Name[n:] {
			case "_SET", "_CLR", "_TOG":
				if r.Type == "" {
					r.Type = r.Name[:n]
				}
				r.Bits = nil
			}
			for _, bf := range r.Bits {
				typ := r.Type
				if typ == "" {
					typ = r.Name
				}
				if strings.HasPrefix(bf.Name, typ) {
					continue
				}
				bf.Name = typ + "_" + bf.Name
				for _, v := range bf.Values {
					v.Name = typ + "_" + v.Name
				}
			}
		}
	}
}

func imxrtdma(p *Periph) {
	var tcd Reg
	for i, r := range p.Regs {
		if r == nil {
			continue
		}
		if strings.HasPrefix(r.Name, "DCHPRI") {
			r.Type = "DCHPR"
			if r.Name != "DCHPRI3" {
				r.Bits = nil
			}
			continue
		}
		if strings.HasPrefix(r.Name, "TCD") {
			switch {
			case strings.HasPrefix(r.Name, "TCD0_"):
				r.Name = r.Name[5:]
				switch r.Name {
				case "SADDR":
					tcd = *r
					tcd.Name = "TCD"
					tcd.Len = 1
					tcd.Descr = "Transfer Control Descriptors"
					p.Regs[i] = &tcd
					r.Bits = nil
				case "SOFF":
					r.Name = "ATTR_SOFF"
					attr := p.Regs[i+1]
					p.Regs[i+1] = nil
					for _, bf := range attr.Bits {
						bf.LSL += 16
					}
					r.Bits = append(r.Bits, attr.Bits...)
				case "NBYTES_MLOFFNO":
					r.Name = "ML_NBYTES"
					r.Bits = nil
					p.Regs[i+1] = nil
					p.Regs[i+2] = nil
				case "SLAST", "DADDR":
					r.Bits = nil
				case "DOFF":
					r.Name = "ELINK_CITER_DOFF"
					r.Bits = []*BitField{
						{"DOFF", 0xffff, 0, "Destination Address Signed Offset", nil},
						{"ELINK_CITER", 0xffff, 16, "Current Minor Loop Link, Major Loop Count, Channel Linking", nil},
					}
					p.Regs[i+1] = nil
					p.Regs[i+2] = nil
				case "DLASTSGA":
					r.Bits = nil
				case "CSR":
					r.Name = "ELINK_BITER_CSR"
					r.Bits = append(
						r.Bits,
						&BitField{"ELINK_BITER", 0xffff, 16, "Beginning Minor Loop Link, Major Loop Count, Channel Linking Disabled", nil},
					)
					p.Regs[i+1] = nil
					p.Regs[i+2] = nil
				}
				tcd.SubRegs = append(tcd.SubRegs, r)
			case strings.HasSuffix(r.Name, "_SADDR"):
				tcd.Len++
			}
			if p.Regs[i] == r {
				p.Regs[i] = nil
			}
			continue
		}
		switch r.Name {
		case "CR":
			for _, bf := range r.Bits {
				if bf.Name == "ACTIVE" {
					bf.Name = "ACT"
				}
			}
		case "ES":
			for _, bf := range r.Bits {
				switch bf.Name {
				case "ERRCHN":
					bf.Name = "CNE"
				case "ECX":
					bf.Name = "CXE"
				}
			}
		case "CEEI":
			r.Type = "CTRL"
			r.Bits = []*BitField{
				{"CMASK", 31, 0, "Affect the specified channels", nil},
				{"CALL", 1, 6, "Affect all channels", nil},
				{"NOP", 1, 7, "Allows 32-bit write to selected CTRL registers", nil},
			}
		case "SEEI", "CERQ", "SERQ", "CDNE", "SSRT", "CERR", "CINT":
			r.Type = "CTRL"
			r.Bits = nil
		}
	}
}

func imxrtlpuart(p *Periph) {
	for _, r := range p.Regs {
		switch r.Name {
		case "FIFO":
			for _, bf := range r.Bits {
				switch bf.Name {
				case "RXEMPT":
					bf.Name = "RXFEMPT"
				case "TXEMPT":
					bf.Name = "TXFEMPT"
				}
			}
		}
	}
}

func imxrtusb(p *Periph) {
	var gpt, epc *Reg
	for i, r := range p.Regs {
		switch {
		case r.Name == "GPTIMER0LD":
			gpt = r
			gpt.Name = "GPTIMER"
			r.SubRegs = []*Reg{
				{Name: "LD"},
				{Name: "CTRL", Type: "GPTCTRL"},
			}
			gpt.Len = 2
			gpt.Descr = "General Purpose Timers"
		case r.Name == "GPTIMER0CTRL":
			gpt.SubRegs[1].Bits = r.Bits
			p.Regs[i] = nil
		case strings.HasPrefix(r.Name, "GPTIMER"):
			p.Regs[i] = nil
		case r.Name == "ENDPTCTRL0":
			p.Regs[i] = nil
		case r.Name == "ENDPTCTRL1":
			epc = r
			epc.Name = "ENDPTCTRL"
			epc.Descr = "Endpoint Control"
			epc.Len = 2
		case strings.HasPrefix(r.Name, "ENDPTCTRL"):
			epc.Len++
			p.Regs[i] = nil
		default:
			switch r.Name {
			case "HCCPARAMS":
				for _, bf := range r.Bits {
					bf.Name = "HP" + bf.Name
				}
			case "DCCPARAMS":
				for _, bf := range r.Bits {
					bf.Name = "DP" + bf.Name
				}
			case "ENDPTSETUPSTAT", "ID", "CAPLENGTH", "HCIVERSION", "DCIVERSION", "FRINDEX":
				r.Bits = nil
			}
		}
	}
}

func imxrtusbphy(p *Periph) {
	for _, r := range p.Regs {
		switch {
		case strings.HasPrefix(r.Name, "PWD_"):
			r.Type = "PWD"
			r.Bits = nil
		case strings.HasPrefix(r.Name, "TX_"):
			r.Type = "TX"
			r.Bits = nil
		case strings.HasPrefix(r.Name, "RX_"):
			r.Type = "RX"
			r.Bits = nil
		case strings.HasPrefix(r.Name, "CTRL_"):
			r.Type = "CTRL"
			r.Bits = nil
		case strings.HasPrefix(r.Name, "DEBUG_"):
			r.Type = "DEBUG"
			r.Bits = nil
		case strings.HasPrefix(r.Name, "DEBUG1_"):
			r.Type = "DEBUG1"
			r.Bits = nil
		case r.Name == "DEBUG":
			for _, bf := range r.Bits {
				if bf.Name == "CLKGATE" {
					bf.Name = "GATECLK"
				}
			}
		}
		for _, bf := range r.Bits {
			if strings.HasPrefix(bf.Name, "RSVD") {
				bf.Name = "_"
			}
		}
	}
}

func imxrtwdog(p *Periph) {
	for _, r := range p.Regs {
		if r.Name == "WSR" {
			r.Bits = nil
		}
	}
}
