// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

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

func k210bus(gs []*Group, ctx *ctx) {
	for _, g := range gs {
		for _, p := range g.Periphs {
			switch p.Name {
			case "gpio", "i2s", "i2c", "uart", "fpioa", "timer", "sha":
				for _, inst := range p.Insts {
					inst.Bus = "APB0"
				}
			case "aes", "wdt", "otp", "rtc":
				for _, inst := range p.Insts {
					inst.Bus = "APB1"
				}
			case "spi":
				for _, inst := range p.Insts {
					switch inst.Name {
					case "SPI0", "SPI1":
						inst.Bus = "APB2"
					case "SPI2":
						inst.Bus = "APB0"
					}
				}
			}
		}
	}

	dir := ctx.push("bus")
	defer ctx.pop()
	mkdir(dir)
	w := create(filepath.Join(dir, ctx.mcu+".go"))
	defer w.Close()
	w.donotedit()
	fmt.Fprintln(w, "// +build", ctx.mcu)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "package bus")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "const (")
	buses := []string{
		"Core",
		"TileLink",
		"AXI",
		"AHB",
		"APB0",
		"APB1",
		"APB2",
	}
	fmt.Fprintf(w, "\t%s Bus = iota\n", buses[0])
	for _, bus := range buses[1:] {
		fmt.Fprintf(w, "\t%s\n", bus)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "\tAHBLast = AHB")
	fmt.Fprintln(w, "\tAPBLast = APB2")
	fmt.Fprintln(w, ")")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "type Bus uint8")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "var str = [%d]string{\n", len(buses))
	for _, bus := range buses {
		fmt.Fprintf(w, "\t\"%s\",\n", bus)
	}
	fmt.Fprintln(w, "}")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "func (b Bus) String() string { return str[b] }")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "var buses [%d]struct{ clockHz int64 }\n", len(buses))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "func (b Bus) Clock() int64 { return buses[b].clockHz }")
	fmt.Fprintln(w, "func (b Bus) SetClock(Hz int64) { buses[b].clockHz = Hz }")
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
