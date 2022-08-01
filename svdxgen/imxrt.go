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
	for _, r := range p.Regs {
		if strings.HasPrefix(r.Name, "DCHPRI") {
			r.Type = "DCHPR"
			if r.Name != "DCHPRI3" {
				r.Bits = nil
			}
			continue
		}
		switch r.Name {
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

func imxrtwdog(p *Periph) {
	for _, r := range p.Regs {
		if r.Name == "WSR" {
			r.Bits = nil
		}
	}
}
