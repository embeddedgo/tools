// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "strings"

func imxrttweaks(gs []*Group) {
	for _, g := range gs {
		for _, p := range g.Periphs {
			switch p.Name {
			case "gpio":
				imxrtgpio(p)
			case "iomuxc":
				imxrtiomuxc(p)
			case "aoi", "lcdif", "usb_analog", "tmr", "enet", "tsc", "pxp",
				"ccm_analog", "pmu", "nvic":
				p.Insts = nil
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
	firstMux := true
	firstPad := true
	for _, r := range p.Regs {
		switch {
		case strings.HasPrefix(r.Name, "SW_MUX_CTL_PAD_GPIO_"):
			r.Type = "SW_MUX_CTL"
			if firstMux {
				firstMux = false
			} else {
				r.Bits = nil
			}
		case strings.HasPrefix(r.Name, "SW_PAD_CTL_PAD_GPIO_"):
			r.Type = "SW_PAD_CTL"
			if firstPad {
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
