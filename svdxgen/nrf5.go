// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"strings"
)

func nrf5tweaks(gs []*Group) {
	for _, g := range gs {
		for _, p := range g.Periphs {
			nfr5drop(p)
			switch p.Name {
			case "ficr":
				nrf5ficr(p)
			case "gpio":
				nrf5gpio(p)
			case "uicr":
				nrf5uicr(p)
			default:
				p.Insts = nil
			}
		}
	}
}

func nfr5drop(p *Periph) {
	for _, r := range p.Regs {
		switch {
		case strings.HasPrefix(r.Name, "TASKS_") || strings.HasPrefix(r.Name, "EVENTS_"):
			r.Bits = nil
		case len(r.Bits) == 1:
			if b := r.Bits[0]; b.Mask == 0xFFFFFFFF && len(b.Values) <= 1 {
				r.Bits = nil
			}
		}
	}
}

func nrf5ficr(p *Periph) {
	var a, b, t *Reg
	for i, r := range p.Regs {
		switch r.Name {
		case "INFO_VARIANT", "INFO_PACKAGE", "INFO_RAM", "INFO_FLASH":
			for _, b := range r.Bits {
				for _, v := range b.Values {
					if v.Name == "Unspecified" {
						v.Name = b.Name[:1] + "unspec"
					} else if v.Name[0] == 'K' {
						v.Name = b.Name[:1] + v.Name[1:] + "K"
					}
				}
			}
		case "DEVICEADDRTYPE":
			for _, b := range r.Bits {
				if b.Name == "DEVICEADDRTYPE" {
					b.Name = "DEVADDRTYPE"
				}
			}
		case "PRODTEST":
			for _, b := range r.Bits {
				if b.Name == "PRODTEST" {
					b.Name = "PTEST"
				}
			}
		}
		switch {
		case r.Name == "TEMP_A0":
			a = r
			a.Name = "TEMP_A"
			a.Len = 1
			a.Descr = "Slope definition"
		case strings.HasPrefix(r.Name, "TEMP_A"):
			a.Len++
			p.Regs[i] = nil
		case r.Name == "TEMP_B0":
			b = r
			b.Name = "TEMP_B"
			b.Len = 1
			b.Descr = "Y-intercept"
		case strings.HasPrefix(r.Name, "TEMP_B"):
			b.Len++
			p.Regs[i] = nil
		case r.Name == "TEMP_T0":
			t = r
			t.Name = "TEMP_T"
			t.Len = 1
			t.Descr = "Segment end"
		case strings.HasPrefix(r.Name, "TEMP_T"):
			t.Len++
			p.Regs[i] = nil
		}
	}
}

func nrf5gpio(p *Periph) {
	for _, r := range p.Regs {
		switch r.Name {
		case "OUT", "OUTSET", "OUTCLR", "IN", "DIR", "DIRSET", "DIRCLR", "LATCH":
			r.Bits = nil
		case "DETECTMODE":
			for _, b := range r.Bits {
				if b.Name == "DETECTMODE" {
					b.Name = "DETECT"
					for _, v := range b.Values {
						switch v.Name {
						case "Default":
							v.Name = "Direct"
						case "LDETECT":
							v.Name = "Latched"
						}
					}
				}
			}
		case "PIN_CNF":
			for _, b := range r.Bits {
				switch b.Name {
				case "DIR":
					b.Name = "DIRECTION"
				case "PULL":
					for _, v := range b.Values {
						if v.Name == "Disabled" {
							v.Name = "NoPull"
						}
					}
				}
			}
		}
	}
}

func nrf5uicr(p *Periph) {
	for _, r := range p.Regs {
		switch r.Name {
		case "NRFFW", "NRFHW", "CUSTOMER":
			r.Bits = nil
		case "NFCPINS":
			for _, b := range r.Bits {
				for _, v := range b.Values {
					if v.Name == "Disabled" {
						v.Name = "GPIO"
					}
				}
			}
		case "DEBUGCTRL":
			for _, b := range r.Bits {
				switch b.Name {
				case "CPUNIDEN":
					for _, v := range b.Values {
						v.Name = "ID" + v.Name
					}
				case "CPUFPBEN":
					for _, v := range b.Values {
						v.Name = "FPB" + v.Name
					}
				}
			}
		case "REGOUT0":
			for _, b := range r.Bits {
				for _, v := range b.Values {
					switch {
					case v.Name == "DEFAULT":
						v.Name = "Vdefault"
					case len(v.Name) == 3 && v.Name[1] == 'V':
						v.Name = "V" + v.Name[0:1] + "_" + v.Name[2:3]
					}
				}
			}
		}
	}
}
