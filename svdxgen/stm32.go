// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

func stm32tweaks(gs []*Group) {
	for _, g := range gs {
		switch g.Name {
		case "gpio":
			stm32gpio(g)
		case "spi":
			stm32spi(g)
		}
		for _, p := range g.Periphs {
			switch p.Name {
			case "dma":
				stm32dma(p)
			case "exti":
				stm32exti(p)
			case "flash":
				stm32flash(p)
			case "fmc", "fsmc":
				stm32fmc(p)
			case "pwr":
				stm32pwr(p)
			case "rcc":
				stm32rcc(p)
			case "sdio":
				stm32sdio(p)
			case "syscfg":
				stm32syscfg(p)
			}
		}
	}
}

func stm32bus(gs []*Group, ctx *ctx) {
	var rcc *Periph
gloop:
	for _, g := range gs {
		if g.Name == "rcc" {
			for _, p := range g.Periphs {
				if p.Name == "rcc" {
					rcc = p
					break gloop
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
	buses := make([]string, 1, 8)
	buses[0] = "Core"
	fmt.Fprintf(w, "\t%s Bus = iota\n", buses[0])
	namelen := len(buses[0])
	var ahbLast, apbLast string
	for _, r := range rcc.Regs {
		if r == nil {
			continue
		}
		i := strings.Index(r.Name, "ENR")
		if i <= 0 {
			continue
		}
		if c := r.Name[i-1]; c != 'B' && !unicode.IsDigit(rune(c)) {
			continue
		}
		bus := r.Name[:i]
		if len(r.Name) == i+3 || len(r.Name) == i+4 && r.Name[i+3] == '1' {
			fmt.Fprintf(w, "\t%s\n", bus)
			if bus[1] == 'H' {
				ahbLast = bus
			} else {
				apbLast = bus
			}
			if len(bus) != namelen {
				namelen = 0
			}
			buses = append(buses, bus)
		}
		for _, bf := range r.Bits {
			iname := strings.TrimSuffix(bf.Name, "EN")
			inames := []string{iname}
			switch iname {
			case "ADC12":
				inames = []string{"ADC1", "ADC2"}
			case "SPI1":
				inames = append(inames, "I2S2ext", "I2S3ext")
			}
			for _, iname = range inames {
				if inst := ctx.instmap[iname]; inst != nil {
					inst.Bus = bus
				}
			}
		}
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "\tAHBLast =", ahbLast)
	fmt.Fprintln(w, "\tAPBLast =", apbLast)
	fmt.Fprintln(w, ")")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "type Bus uint8")
	fmt.Fprintln(w)
	if namelen == 0 {
		fmt.Fprintf(w, "var str = [%d]string{\n", len(buses))
		for _, bus := range buses {
			fmt.Fprintf(w, "\t\"%s\",\n", bus)
		}
		fmt.Fprintln(w, "}")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "func (b Bus) String() string { return str[b] }")
	} else {
		fmt.Fprintln(w, "func (b Bus) String() string {")
		fmt.Fprintf(w, "\ti := int(b) * %d\n", namelen)
		fmt.Fprintf(
			w, "\treturn \"%s\"[i:i+%d]\n",
			strings.Join(buses, ""), namelen,
		)
		fmt.Fprintln(w, "}")
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "var buses [%d]struct{ clockHz int64 }\n", len(buses))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "func (b Bus) Clock() int64 { return buses[b].clockHz }")
	fmt.Fprintln(w, "func (b Bus) SetClock(Hz int64) { buses[b].clockHz = Hz }")
}

func stm32gpio(g *Group) {
	if len(g.Periphs) <= 1 {
		return
	}
	gpio := g.Periphs[0]
	for _, p := range g.Periphs[1:] {
		gpio.Insts = append(gpio.Insts, p.Insts...)
		p.Insts = nil
	}
	g.Periphs = g.Periphs[:1]
	gpio.Name = "gpio"
	gpio.OrigName = "GPIO"
	for _, r := range g.Periphs[0].Regs {
		if r.Name == "BRR" {
			for _, bf := range r.Bits {
				if strings.HasPrefix(bf.Name, "BR") {
					bf.Name = "BC" + bf.Name[2:]
					bf.Descr = "Port x reset (clear) bit y"
				}
			}
		}
	}
}

func stm32spi(g *Group) {
	spi := g.Periphs[0]
	if len(g.Periphs) > 1 {
		spi.Name = "spi"
		spi.OrigName = "SPI"
		for _, p := range g.Periphs[1:] {
			spi.Insts = append(spi.Insts, p.Insts...)
			p.Insts = nil
			if len(spi.Regs) < len(p.Regs) {
				spi.Regs = p.Regs
			}
		}
	}
	for _, r := range spi.Regs {
		switch {
		case r.Name == "DR":
			r.Bits = nil
		}
	}
}

func stm32dma(p *Periph) {
	var st, ch *Reg
	for i, r := range p.Regs {
		switch {
		case r.Name == "LISR" || r.Name == "LIFCR":
			r.Name = r.Name[1:]
			r.Descr = strings.TrimPrefix(r.Descr, "low")
			r.Len = 2
		case r.Name == "HISR" || r.Name == "HIFCR":
			p.Regs[i] = nil

		case r.Name == "S0CR":
			cr := *r
			cr.Name = "CR"
			st = r
			st.Name = "S"
			st.Len = 1
			st.Descr = "stream configuration and controll registers"
			st.SubRegs = append(st.SubRegs, &cr)
		case strings.HasPrefix(r.Name, "S0"):
			r.Name = r.Name[2:]
			st.SubRegs = append(st.SubRegs, r)
			p.Regs[i] = nil
		case len(r.Name) > 2 && r.Name[0] == 'S' && '0' <= r.Name[1] && r.Name[1] <= '9':
			if strings.HasSuffix(r.Name, "NDTR") {
				st.Len++
			}
			p.Regs[i] = nil

		case r.Name == "CCR1":
			cr := *r
			cr.Name = "CR"
			ch = r
			ch.Name = "C"
			ch.Len = 1
			ch.Descr = "channel configuration and controll registers"
			ch.SubRegs = append(ch.SubRegs, &cr)
		case len(r.Name) > 2 && r.Name[0] == 'C' && r.Name[len(r.Name)-1] == '1':
			r.Name = r.Name[1 : len(r.Name)-1]
			ch.SubRegs = append(ch.SubRegs, r)
			p.Regs[i] = nil
		case len(r.Name) > 2 && r.Name[0] == 'C' && '0' <= r.Name[len(r.Name)-1] && r.Name[len(r.Name)-1] <= '9':
			if strings.HasPrefix(r.Name, "CCR") {
				ch.Len++
			}
			p.Regs[i] = nil
		}
	}
	if ch != nil {
		if padn := 5 - len(ch.SubRegs); padn > 0 {
			pad := &Reg{Name: "_"}
			if padn > 1 {
				pad.Len = padn
			}
			ch.SubRegs = append(ch.SubRegs, pad)
		}
	}
}

func stm32exti(p *Periph) {
	for _, r := range p.Regs {
		r.Bits = nil
	}
}

func stm32flash(p *Periph) {
	for i, r := range p.Regs {
		switch r.Name {
		case "PDKEYR", "KEYR", "OPTKEYR":
			r.Bits = nil
		case "ACR_":
			for k := i; k < len(p.Regs); k++ {
				p.Regs[k] = nil
			}
			inst := *p.Insts[0]
			p.Insts = append(p.Insts, &inst)
			p.Insts[0].Name = "FLASH1"
			p.Insts[1].Name = "FLASH2"
			p.Insts[1].Base += fmt.Sprintf("+0x%X", r.Offset)
			return
		}
	}
}

func stm32fmc(p *Periph) {
	var bct, bwtr *Reg
	for i, r := range p.Regs {
		switch {
		case r.Name == "BCR1":
			cr := *r
			cr.Name = "CR"
			bct = r
			bct.Name = "BCT"
			bct.Len = 1
			bct.Descr = "chip-select control and timing registers"
			bct.SubRegs = append(bct.SubRegs, &cr)
		case r.Name == "BTR1":
			r.Name = "TR"
			bct.SubRegs = append(bct.SubRegs, r)
			p.Regs[i] = nil
		case strings.HasPrefix(r.Name, "BCR"):
			bct.Len++
			p.Regs[i] = nil
		case strings.HasPrefix(r.Name, "BTR"):
			p.Regs[i] = nil

		case r.Name == "BWTR1":
			bwtr = r
			bwtr.Name = "BWTR"
			bwtr.Len = 1
			bwtr.Descr = "write timing registers"
		case strings.HasPrefix(r.Name, "BWTR"):
			bwtr.Len++
			p.Regs[i] = nil
		}
	}
}

func stm32pwr(p *Periph) {
	var pudc *Reg
	for i, r := range p.Regs {
		switch {
		case r.Name == "PUCRA":
			pucr := *r
			pucr.Name = "PUCR"
			pudc = r
			pudc.Name = "PUDC"
			pudc.Len = 1
			pudc.Descr = "Power Port x pull-up/pull-down control registers"
			pudc.SubRegs = append(pudc.SubRegs, &pucr)
		case r.Name == "PDCRA":
			r.Name = "PDCR"
			pudc.SubRegs = append(pudc.SubRegs, r)
			p.Regs[i] = nil
		case strings.HasPrefix(r.Name, "PUCR"):
			pudc.Len++
			p.Regs[i] = nil
		case strings.HasPrefix(r.Name, "PDCR"):
			p.Regs[i] = nil
		}
	}
}

func stm32rcc(p *Periph) {
	for i, r := range p.Regs {
		// BUG: core specific registers not supported
		if strings.HasPrefix(r.Name, "C1_") || strings.HasPrefix(r.Name, "C2_") {
			p.Regs[i] = nil
		}
	}
}

func stm32sdio(p *Periph) {
	var resp *Reg
	for i, r := range p.Regs {
		switch {
		case r.Name == "RESP1":
			resp = r
			resp.Name = "RESP"
			resp.Len = 1
			resp.Descr = "response registers"
		case r.Name != "RESPCMD" && strings.HasPrefix(r.Name, "RESP"):
			resp.Len++
			p.Regs[i] = nil
		}
	}
}

func stm32syscfg(p *Periph) {
	var exticr *Reg
	for i, r := range p.Regs {
		switch {
		case r.Name == "EXTICR1":
			exticr = r
			exticr.Name = "EXTICR"
			exticr.Len = 1
			exticr.Descr = "select GPIO port for EXTI line (4 x 4bit)"
			exticr.Bits = nil
		case strings.HasPrefix(r.Name, "EXTICR"):
			exticr.Len++
			p.Regs[i] = nil
		}
	}
}
