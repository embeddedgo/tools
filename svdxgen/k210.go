// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "strings"

func k210tweaks(gs []*Group) {
	for _, g := range gs {
		for _, p := range g.Periphs {
			k210common(p)
			switch p.Name {
			case "sysctl":
				k210sysctl(p)
			case "gpiohs", "gpio":
				k210gpio(p)
			case "fpioa":
			default:
				p.Insts = nil
			}
		}
	}
}

func k210common(p *Periph) {
	for _, r := range p.Regs {
		r.Name = strings.ToUpper(r.Name)
		for _, b := range r.Bits {
			b.Name = strings.ToUpper(b.Name)
			for _, v := range b.Values {
				v.Name = strings.ToUpper(v.Name)
			}
		}
	}
}

func k210sysctl(p *Periph) {
	var pll *Reg
	for i, r := range p.Regs {
		switch {
		case r.Name == "PLL0":
			pll = r
			pll.Name = "PLL"
			pll.Len = 1
			pll.Descr = "PLL controllers"
			for _, b := range r.Bits {
				if b.Name == "TEST_EN" {
					b.Name += "_CKIN_SEL"
					b.Mask = 3
				}
			}
		case r.Name == "PLL1" || r.Name == "PLL2":
			pll.Len++
			p.Regs[i] = nil
		case strings.HasPrefix(r.Name, "DMA_SEL"):
			r.Bits = nil // TODO: understand the registers/bits structuree
		case r.Name == "SOFT_RESET":
			r.Bits = nil
		}
	}
}

func k210gpio(p *Periph) {
	for _, r := range p.Regs {
		r.Bits = nil
	}
}
