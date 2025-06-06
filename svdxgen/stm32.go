// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func stm32tweaks(gs []*Group) {
	for _, g := range gs {
		switch g.Name {
		case "gpio":
			stm32gpio(g)
		case "spi":
			stm32spi(g)
		case "tim":
			stm32tim(g)
		}
		for _, p := range g.Periphs {
			stm32irq(p)
			switch p.Name {
			case "dma":
				stm32dma(p)
			case "dmamux", "dmamux1", "dmamux2":
				stm32dmamux(p)
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
			case "rtc":
				stm32rtc(p)
			case "sdio":
				stm32sdio(p)
			case "syscfg":
				stm32syscfg(p)
			}
		}
	}
}

func stm32irq(p *Periph) {
	for _, inst := range p.Insts {
		for _, irq := range inst.IRQs {
			irq.Name = strings.ToUpper(irq.Name)
			if strings.HasPrefix(irq.Name, "DMA_STR") {
				irq.Name = "DMA1_STR" + irq.Name[7:]
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
	fmt.Fprintln(w, "//go:build", ctx.mcu)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "package bus")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "const (")
	buses := make([]string, 1, 8)
	buses[0] = "Core"
	fmt.Fprintf(w, "\t%s Bus = iota\n", buses[0])
	for _, r := range rcc.Regs {
		if r == nil {
			continue
		}
		i := strings.Index(r.Name, "ENR")
		if i < 3 || i > 5 {
			continue
		}
		if r.Name[0] != 'A' || r.Name[2] != 'B' {
			continue
		}
		bus := r.Name[:i]
		if bus[i-1] == 'L' || bus[i-1] == 'H' {
			bus = bus[:i-1]
		}
		if r.Name[i-1] != 'H' && r.Name[len(r.Name)-1] != '2' {
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
	sort.Strings(buses[1:])
	var ahbLast, apbLast string
	namelen := len(buses[0])
	for _, bus := range buses[1:] {
		fmt.Fprintf(w, "\t%s\n", bus)
		if bus[1] == 'H' {
			ahbLast = bus
		} else {
			apbLast = bus
		}
		if len(bus) != namelen {
			namelen = 0
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
		switch r.Name {
		case "DR", "CRCPR", "RXCRCR", "TXCRCR", "TXDR", "RXDR", "CRCPOLY", "TXCRC", "RXCRC", "UDRDR":
			r.Bits = nil
		case "IER", "IFCR":
			r.Type = "SR"
			r.Bits = nil
		}
	}
}

func stm32tim(g *Group) {
	tim := g.Periphs[0]
	if len(g.Periphs) > 1 {
		tim.Name = "tim"
		tim.OrigName = "TIM"
		for _, p := range g.Periphs[1:] {
			tim.Insts = append(tim.Insts, p.Insts...)
			p.Insts = nil
			if len(tim.Regs) < len(p.Regs) {
				tim.Regs = p.Regs
			}
		}
	}
	var ccmr *Reg
	for i, r := range tim.Regs {
		switch r.Name {
		case "CNT", "PSC", "ARR", "RCR", "DMAR":
			r.Bits = nil
			continue
		}
		switch {
		case strings.HasPrefix(r.Name, "CCR"):
			r.Bits = nil
		case strings.HasPrefix(r.Name, "CCMR"):
			switch {
			case strings.HasSuffix(r.Name, "_Output"):
				r.Name = r.Name[:5]
				if k := strings.IndexByte(r.Descr, '('); k >= 0 {
					r.Descr = r.Descr[:k]
				}
				ccmr = r
			case strings.HasSuffix(r.Name, "_Input"):
				for _, bf := range r.Bits {
					if bf.Name[0] == 'I' {
						ccmr.Bits = append(ccmr.Bits, bf)
					}
				}
				tim.Regs[i] = nil
			}
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

func stm32dmamux(p *Periph) {
	for i, r := range p.Regs {
		r.Name = strings.TrimPrefix(r.Name, "DMAMUX_")
		for _, bf := range r.Bits {
			if bf.Mask == 1 {
				bf.Values = nil
			}
		}
		switch {
		case strings.HasSuffix(r.Name, "CR"):
			switch r.Name[0] {
			case 'C':
				if r.Name != "C0CR" {
					p.Regs[i] = nil
					break
				}
				r.Name = "CCR"
				r.Len = 16
				if r.Descr == "" {
					r.Descr = "DMA request line multiplexer channel x control register"
				}
				for _, bf := range r.Bits {
					if bf.Name == "SPOL" {
						bf.Values = []*BitFieldValue{
							{"SPOL_NONE", "No event, i.e. no synchronization nor detection.", 0},
							{"SPOL_RISING", "Rising edge", 1},
							{"SPOL_FALLING", "Falling edge", 2},
							{"SPOL_BOTH", "Rising and falling edges", 3},
						}
					}
				}
			case 'R':
				if r.Name != "RG0CR" {
					p.Regs[i] = nil
					break
				}
				r.Name = "RGCR"
				r.Len = 8
				if r.Descr == "" {
					r.Descr = "DMA request generator channel x control register"
				}
				for _, bf := range r.Bits {
					if bf.Name == "GPOL" {
						bf.Values = []*BitFieldValue{
							{"GPOL_NONE", "No event, i.e. no synchronization nor detection.", 0},
							{"GPOL_RISING", "Rising edge", 1},
							{"GPOL_FALLING", "Falling edge", 2},
							{"GPOL_BOTH", "Rising and falling edges", 3},
						}
					}
				}
			}
		case r.Name == "CSR":
			if r.Descr == "" {
				r.Descr = "DMA request line multiplexer interrupt channel status register"
			}
		case r.Name == "CFR":
			r.Type = "CSR"
			r.Bits = nil
			if r.Descr == "" {
				r.Descr = "DMA request line multiplexer interrupt clear flag register"
			}
		case r.Name == "RGSR":
			if r.Descr == "" {
				r.Descr = "DMA request generator status register"
			}
		case r.Name == "RGCFR":
			r.Type = "RGSR"
			r.Bits = nil
			if r.Descr == "" {
				r.Descr = "DMA request generator clear flag register"
			}
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
		case "PDKEYR", "KEYR", "KEYR1", "OPTKEYR":
			r.Bits = nil
		case "OPTSR_CUR", "OPTSR_PRG":
			r.Type = "OPTSR"
			if r.Name == "OPTSR_PRG" {
				r.Bits = nil
			}
		case "PRAR_CUR1", "PRAR_PRG1":
			r.Type = "PRAR"
			if r.Name == "PRAR_PRG1" {
				r.Bits = nil
			}
		case "SCAR_CUR1", "SCAR_PRG1":
			r.Type = "SCAR"
			if r.Name == "SCAR_PRG1" {
				r.Bits = nil
			}
		case "WPSN_CUR1R", "WPSN_PRG1R":
			r.Type = "WPSN"
			if r.Name == "WPSN_PRG1R" {
				r.Bits = nil
			}
		case "BOOT_CURR", "BOOT_PRGR":
			r.Type = "BOOT"
			if r.Name == "BOOT_PRGR" {
				r.Bits = nil
			}
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
			continue
		}
		switch r.Name {
		case "CFGR":
			for _, bf := range r.Bits {
				switch bf.Name {
				case "SW":
					if bf.Values == nil {
						bf.Values = []*BitFieldValue{
							{"SW_HSI", "HSI oscillator selected as system clock", 0},
							{"SW_CSI", "CSI oscillator selected as system clock", 1},
							{"SW_HSE", "HSE oscillator selected as system clock", 2},
							{"SW_PLL1", "PLL1 selected as system clock", 3},
						}
					}
				case "SWS":
					if bf.Values == nil {
						bf.Values = []*BitFieldValue{
							{"SWS_HSI", "HSI oscillator used as system clock", 0},
							{"SWS_CSI", "CSI oscillator used as system clock", 1},
							{"SWS_HSE", "HSE oscillator used as system clock", 2},
							{"SWS_PLL1", "PLL1 used as system clock", 3},
						}
					}
				}
			}
		case "PLLCKSELR":
			for _, bf := range r.Bits {
				switch bf.Name {
				case "PLLSRC":
					if bf.Values == nil {
						bf.Values = []*BitFieldValue{
							{"PLLSRC_HSI", "HSI selected as PLL clock", 0},
							{"PLLSRC_CSI", "CSI selected as PLL clock", 1},
							{"PLLSRC_HSE", "HSE selected as PLL clock", 2},
							{"PLLSRC_NONE", "No clock to DIVMx divider and PLLs", 3},
						}
					}
				}
			}
		case "BDCR":
			for _, bf := range r.Bits {
				switch bf.Name {
				case "RTCSRC":
					bf.Name = "RTCSEL"
					if bf.Values == nil {
						bf.Values = []*BitFieldValue{
							{"RTCSEL_NONE", "no clock", 0},
							{"RTCSEL_LSE", "LSE oscillator clock used as RTC clock", 1},
							{"RTCSEL_LSI", "LSI oscillator clock used as RTC clock", 2},
							{"RTCSEL_HSE", "HSE clock divided by RTCPRE value is used as RTC clock", 3},
						}
					}
				}
			}
		case "CIFR":
			for _, bf := range r.Bits {
				if bf.Name == "CSIRDY" {
					bf.Name = "CSIRDYF"
				}
			}
		}
	}
}

func stm32rtc(p *Periph) {
	for _, r := range p.Regs {
		if strings.HasPrefix(r.Name, "RTC_") {
			r.Name = r.Name[4:]
		}
	}
	var bkpr *Reg
	for i, r := range p.Regs {
		switch {
		case strings.HasPrefix(r.Name, "ALRM"):
			if strings.HasSuffix(r.Name, "SSR") {
				r.Type = "ALRMSSR"
			} else {
				r.Type = "ALRMR"
			}
			if strings.HasPrefix(r.Name, "ALRMA") {
				for _, b := range r.Bits {
					b.Name = "A" + b.Name
				}
			} else {
				r.Bits = nil
			}
		case r.Name == "BKP0R":
			r.Name = "BKPR"
			r.Len = 1
			r.Bits = nil
			bkpr = r
		case r.Name == "TSTR":
			r.Type = "TR"
			r.Bits = nil
		case r.Name == "TSDR":
			r.Type = "DR"
			r.Bits = nil
		case r.Name == "TSSSR":
			r.Type = "SSR"
			r.Bits = nil
		case strings.HasPrefix(r.Name, "BKP"):
			bkpr.Len++
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
