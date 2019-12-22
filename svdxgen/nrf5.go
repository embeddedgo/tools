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
			nfr5te(p)
			switch p.Name {
			case "uicr":
				nrf5uicr(p)
			case "gpio":
				nrf5gpio(p)
			default:
				p.Insts = nil
			}
		}
	}
}

func nfr5te(p *Periph) {
	for _, r := range p.Regs {
		if strings.HasPrefix(r.Name, "TASKS_") || strings.HasPrefix(r.Name, "EVENTS_") {
			r.Bits = nil
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
					b.Name = "LDETECT"
					b.Values = nil
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
