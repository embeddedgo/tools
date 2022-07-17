// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

func imxrttweaks(gs []*Group) {
	for _, g := range gs {
		for _, p := range g.Periphs {
			switch p.Name {
			case "gpio":
				imxrtgpio(p)
			case "aoi", "lcdif", "usb_analog", "tmr", "enet", "tsc", "pxp",
				"ccm_analog", "pmu", "nvic":
				p.Insts = nil
			}
		}
	}
}

func imxrtgpio(p *Periph) {
	for _, r := range p.Regs {
		if len(r.Name) == 4 && r.Name[:3] == "ICR" {
			n := r.Name[3]
			r.Name = r.Name[:3]
			if n == '1' {
				r.Name += "A"
			} else {
				r.Name += "B"
			}
		}
	}
}
