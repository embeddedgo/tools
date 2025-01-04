// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/embeddedgo/tools/svd"
)

type BitFieldValue struct {
	Name  string
	Descr string
	Value uint64
}

type BitField struct {
	Name   string
	Mask   uint64
	LSL    uint
	Descr  string
	Values []*BitFieldValue
}

type Reg struct {
	Offset  uint64
	BitSiz  uint
	Name    string
	Type    string
	Len     int
	Descr   string
	Bits    []*BitField
	SubRegs []*Reg
}

type IRQ struct {
	Value  int
	Name   string
	Descr  string
	Inst   *Instance
	Shared string // shared by different instances(+) / peripherals(*)
}

type Instance struct {
	Name   string
	Base   string
	Bus    string
	Periph *Periph
	IRQs   []*IRQ
	Descr  string
}

type Periph struct {
	Name     string
	OrigName string
	Insts    []*Instance
	Regs     []*Reg
}

func (p *Periph) Save(ctx *ctx) {
	dir := ctx.push(p.Name)
	defer ctx.pop()
	mkdir(dir)
	w := create(filepath.Join(dir, ctx.mcu+".go"))
	defer w.Close()

	w.donotedit()
	fmt.Fprintln(w, "//go:build", ctx.mcu)
	fmt.Fprintln(w)
	fmt.Fprintln(
		w,
		"// Package", p.Name, "provides access to the registers of the",
		strings.TrimRightFunc(p.OrigName, unicode.IsDigit), "peripheral.",
	)
	fmt.Fprintln(w, "//")
	fmt.Fprintln(w, "// Instances:")
	tw := new(tabwriter.Writer)
	tw.Init(w, 0, 0, 1, ' ', 0)
	bus := false
	for _, inst := range p.Insts {
		if inst.Bus != "-" {
			bus = true
		}
		irqs := "-"
		if len(inst.IRQs) != 0 {
			is := make([]string, len(inst.IRQs))
			for i, irq := range inst.IRQs {
				is[i] = irq.Name + irq.Shared
			}
			irqs = strings.Join(is, ",")
		}
		fmt.Fprintf(
			tw, "//  %s\t %s\t %s\t %s\t", inst.Name, inst.Base, inst.Bus, irqs,
		)
		if inst.Descr != "" && strings.ToLower(inst.Descr) != p.Name {
			fmt.Fprintf(tw, " %s\n", fixSpaces(inst.Descr))
		} else {
			fmt.Fprintln(tw)
		}
	}
	tw.Flush()
	fmt.Fprintln(w, "// Registers:")
	for _, r := range p.Regs {
		if r == nil {
			continue
		}
		if r.Name == "" {
			if r.Type == "" {
				die("register with empty name and type")
			}
			continue
		}
		fmt.Fprintf(tw, "//  0x%03X\t%2d\t ", r.Offset, r.BitSiz)
		name := r.Name
		if r.Type != "" {
			name += "(" + r.Type + ")"
		}
		if len(r.SubRegs) > 0 {
			subregs := r.SubRegs[0].Name
			for _, sr := range r.SubRegs[1:] {
				subregs += "," + sr.Name
				if sr.Type != "" {
					subregs += "(" + sr.Type + ")"
				}
				if sr.Len != 0 {
					subregs += fmt.Sprintf("[%d]", sr.Len)
				}
			}
			name += "{" + subregs + "}"
		}
		if r.Len == 0 {
			fmt.Fprintf(tw, "%s\t", name)
		} else {
			fmt.Fprintf(tw, "%s[%d]\t", name, r.Len)
		}
		if r.Descr != "" {
			fmt.Fprintf(tw, " %s\n", fixSpaces(r.Descr))
		} else {
			fmt.Fprintln(tw)
		}
	}
	tw.Flush()
	fmt.Fprintln(w, "// Import:")
	if bus {
		fmt.Fprintln(w, "// ", importRoot+"/bus")
	}
	fmt.Fprintln(w, "// ", importRoot+"/mmap")
	fmt.Fprintln(w, "package", p.Name)

	saveBits(w, p.Regs)
}

func saveBits(w io.Writer, regs []*Reg) {
	for _, r := range regs {
		if r == nil {
			continue
		}
		if len(r.SubRegs) > 0 {
			saveBits(w, r.SubRegs)
			continue
		}
		if len(r.Bits) == 0 {
			continue
		}
		empty := true
		for _, bf := range r.Bits {
			if bf != nil {
				empty = false
				break
			}
		}
		if empty {
			continue
		}
		fmt.Fprintln(w, "\nconst (")
		typ := r.Type
		if typ == "" {
			typ = r.Name
		}
		for _, bf := range r.Bits {
			if bf == nil {
				continue
			}
			fmt.Fprintf(
				w, "\t%s %s = 0x%02X << %d //+ %s\n",
				bf.Name, typ, bf.Mask, bf.LSL, fixSpaces(bf.Descr),
			)
			for _, bv := range bf.Values {
				if bv == nil {
					continue
				}
				fmt.Fprintf(
					w, "\t%s %s = 0x%02X << %d",
					bv.Name, typ, bv.Value, bf.LSL,
				)
				if bv.Descr != "" {
					fmt.Fprintf(w, " //  %s\n", fixSpaces(bv.Descr))
				} else {
					fmt.Fprintln(w)
				}
			}
		}
		fmt.Fprintln(w, ")")
		if !strings.HasPrefix(typ, "int") {
			fmt.Fprintln(w, "\nconst (")
			for _, bf := range r.Bits {
				if bf == nil || bf.Name == "_" {
					continue
				}
				fmt.Fprintf(w, "\t%sn = %d\n", bf.Name, bf.LSL)
			}
			fmt.Fprintln(w, ")")
		}
	}
}

type Group struct {
	Name    string
	Periphs []*Periph
	pmap    map[string]*Periph
}

func (g *Group) Save(ctx *ctx) {
	if len(g.Periphs) > 1 {
		dir := ctx.push(g.Name)
		defer ctx.pop()
		mkdir(dir)
	}
	for _, p := range g.Periphs {
		p.Save(ctx)
	}
}

func savePeriphs(ctx *ctx) {
	gmap := make(map[string]*Group)
	pmap := make(map[string]*Periph)
	for _, sp := range ctx.spsli {
		var sdp *svd.Peripheral
		if sp.DerivedFrom != nil {
			sdp = ctx.spmap[*sp.DerivedFrom]
		}
		var gid string
		if sp.GroupName != nil {
			gid = *sp.GroupName
		} else if sdp != nil && sdp.GroupName != nil {
			gid = *sdp.GroupName
		}
		if gid == "" {
			gid = dropDigits(sp.Name)
		}
		g := gmap[gid]
		if g == nil {
			g = &Group{
				Name: strings.ToLower(gid),
				pmap: make(map[string]*Periph),
			}
			gmap[gid] = g
		}
		var p *Periph
		if sdp == nil {
			p = &Periph{Name: strings.ToLower(sp.Name), OrigName: sp.Name}
			pmap[sp.Name] = p
			g.pmap[sp.Name] = p
			if len(sp.Registers) > 0 {
				sp.Clusters = append(
					sp.Clusters,
					&svd.Cluster{Registers: sp.Registers},
				)
				sp.Registers = nil
			}
			for _, sc := range sp.Clusters {
				if len(sc.Clusters) > 0 {
					warn(sp.Name, sc.Name, ": cluster in cluster not supported")
				}
				handleRegs(p, ctx.defwidth, sc)
			}
			sort.Slice(
				p.Regs,
				func(i, k int) bool {
					return p.Regs[i].Offset < p.Regs[k].Offset
				},
			)
		} else {
			p = g.pmap[sdp.Name]
			if p == nil {
				p = pmap[sdp.Name]
				if p == nil {
					warn(sp.Name, "", "derived from non-existent "+sdp.Name)
					continue
				}
			}
		}
		inst := &Instance{
			Name:   sp.Name,
			Base:   sp.Name + "_BASE",
			Bus:    "-",
			Periph: p,
		}
		if sp.Description != nil {
			inst.Descr = strings.TrimSpace(*sp.Description)
		} else if sdp != nil && sdp.Description != nil {
			inst.Descr = strings.TrimSpace(*sdp.Description)
		}
		if len(sp.Interrupts) != 0 {
			handleIRQs(ctx, inst, sp.Interrupts)
		} else if sdp != nil && len(sdp.Interrupts) != 0 {
			handleIRQs(ctx, inst, sdp.Interrupts)
		}

		p.Insts = append(p.Insts, inst)
		ctx.instmap[inst.Name] = inst
	}

	gsli := make([]*Group, len(gmap))
	i := 0
	for _, g := range gmap {
		gsli[i] = g
		i++
		g.Periphs = make([]*Periph, len(g.pmap))
		k := 0
		for _, p := range g.pmap {
			g.Periphs[k] = p
			k++
		}
		g.pmap = nil
		switch len(g.Periphs) {
		case 0:
			continue
		case 1:
			g.Periphs[0].Name = g.Name
		default:
			sort.Slice(
				g.Periphs,
				func(i, k int) bool {
					return pnameLess(g.Periphs[i].Name, g.Periphs[k].Name)
				},
			)
		}
		for k, p := range g.Periphs {
			nodigit := dropDigits(p.Name)
			for k1, p1 := range g.Periphs {
				if k1 != k && dropDigits(p1.Name) == nodigit {
					nodigit = ""
					break
				}
			}
			if nodigit != "" {
				p.Name = nodigit
			}
		}
	}
	gmap = nil

	switch {
	case strings.HasPrefix(ctx.mcu, "stm32"):
		stm32tweaks(gsli)
		stm32bus(gsli, ctx)
	case strings.HasPrefix(ctx.mcu, "nrf5"):
		nrf5tweaks(gsli)
	case strings.HasPrefix(ctx.mcu, "k210"):
		k210tweaks(gsli)
		k210bus(gsli, ctx)
	case strings.HasPrefix(ctx.mcu, "imxrt"):
		imxrttweaks(gsli)
	case strings.HasPrefix(ctx.mcu, "rp"):
		picotweaks(gsli)
	}
	saveIRQs(ctx)

	for _, g := range gsli {
		periphs := make([]*Periph, 0, len(g.Periphs))
		for _, p := range g.Periphs {
			if len(p.Insts) == 0 {
				continue
			}
			periphs = append(periphs, p)
			sort.Slice(
				p.Insts,
				func(i, j int) bool {
					return pnameLess(p.Insts[i].Name, p.Insts[j].Name)
				},
			)
		}
		g.Periphs = periphs
		g.Save(ctx)
	}
}

func handleRegs(p *Periph, defwidth uint, sc *svd.Cluster) {
	width := defwidth
	if sc.RegisterPropertiesGroup != nil && sc.Size != nil {
		width = uint(*sc.Size)
	}
	if i := strings.Index(sc.Name, "%s"); i > 0 {
		if sc.Name[i-1] == '[' {
			i--
		}
		cr := &Reg{
			Offset: uint64(sc.AddressOffset),
			BitSiz: width,
			Name:   sc.Name[:i],
			Len:    int(sc.Dim),
		}
		offset := uint64(0)
		for _, sr := range sc.Registers {
			if sr.DerivedFrom != nil {
				warn(p.Name, sr.Name, ": derived registers not supported")
				return
			}
			if sr.Dim != 0 {
				warn(p.Name, sr.Name, ": register dim in array cluster not supporetd")
				return
			}
			if sr.RegisterPropertiesGroup != nil && sr.Size != nil && uint(*sr.Size) != width {
				warn(p.Name, sr.Name, ": reg. width don't match cluster reg. width")
			}
			if uint64(sr.AddressOffset) != offset {
				warn(p.Name, sr.Name, ":holes in array cluster not supported")
				return
			}
			offset += uint64(width / 8)
			r := &Reg{
				Offset: uint64(sc.AddressOffset) + uint64(sr.AddressOffset),
				BitSiz: width,
				Name:   sr.Name,
			}
			if sr.Description != nil && *sr.Description != "" {
				if cr.Descr != "" {
					cr.Descr += "; "
				}
				cr.Descr += strings.TrimSpace(*sr.Description)
			}
			cr.SubRegs = append(cr.SubRegs, r)
			handleFields(r, sr.Fields)
		}
		p.Regs = append(p.Regs, cr)
	} else {
		prefix := ""
		if sc.Name != "" {
			prefix = sc.Name + "_"
		}
		for _, sr := range sc.Registers {
			if sr.DerivedFrom != nil {
				warn(p.Name, sr.Name, ": derived registers not supported")
				continue
			}
			r := &Reg{
				Offset: uint64(sc.AddressOffset) + uint64(sr.AddressOffset),
				BitSiz: width,
				Name:   prefix + sr.Name,
			}
			p.Regs = append(p.Regs, r)
			if sr.RegisterPropertiesGroup != nil && sr.Size != nil {
				r.BitSiz = uint(*sr.Size)
			}
			if sr.Description != nil {
				r.Descr = strings.TrimSpace(*sr.Description)
			}
			if i := strings.Index(r.Name, "%s"); i > 0 {
				if r.Name[i-1] == '[' {
					i--
				}
				if uint(sr.DimIncrement*8) != r.BitSiz {
					warn(p.Name, sr.Name, ": dimIncrement does not match register width")
				} else {
					r.Name = r.Name[:i]
					r.Len = int(sr.Dim)
				}
			}
			handleFields(r, sr.Fields)
		}
	}
}

func handleFields(r *Reg, sfs []*svd.Field) {
	for _, sf := range sfs {
		if sf.DerivedFrom != nil {
			warn(r.Name, sf.Name, ": derived fields not supported")
			continue
		}
		bf := &BitField{Name: sf.Name}
		r.Bits = append(r.Bits, bf)
		if sf.Description != nil {
			bf.Descr = strings.TrimSpace(*sf.Description)
		}
		switch {
		case sf.BitRangeOffsetWidth != nil:
			bf.LSL = uint(sf.BitRangeOffsetWidth.BitOffset)
			bf.Mask = 1
			if w := sf.BitRangeOffsetWidth.BitWidth; w != nil {
				bf.Mask = 1<<*w - 1
			}
		case sf.BitRangeLSBMSB != nil:
			lsb := uint(sf.BitRangeLSBMSB.LSB)
			msb := uint(sf.BitRangeLSBMSB.MSB)
			bf.LSL = lsb
			bf.Mask = 1<<(1+msb-lsb) - 1
		case sf.BitRangePattern != nil:
			br := *sf.BitRangePattern
			if len(br) >= 5 && br[0] == '[' && br[len(br)-1] == ']' {
				br = br[1 : len(br)-1]
				colon := strings.IndexByte(br, ':')
				if colon > 0 && colon < len(br)-1 {
					msb, merr := strconv.ParseUint(br[:colon], 10, 6)
					lsb, lerr := strconv.ParseUint(br[colon+1:], 10, 6)
					if merr == nil && lerr == nil {
						bf.LSL = uint(lsb)
						bf.Mask = 1<<(1+msb-lsb) - 1
						break
					}
				}
			}
			warn(r.Name, sf.Name, ": bad bit-range pattern")
			continue
		default:
			warn(r.Name, sf.Name, ": bit-range not specified")
			continue
		}
		for _, sevs := range sf.EnumeratedValues {
			for _, sev := range sevs.EnumeratedValue {
				if sev.Name == nil || sev.Value == nil {
					continue
				}
				val, err := sev.Val()
				dieErr(err)
				bv := &BitFieldValue{
					Name:  *sev.Name,
					Value: val,
				}
				bf.Values = append(bf.Values, bv)
				if sev.Description != nil {
					bv.Descr = strings.TrimSpace(*sev.Description)
				}
			}
		}
	}
	sort.Slice(
		r.Bits,
		func(i, k int) bool { return r.Bits[i].LSL < r.Bits[k].LSL },
	)
	for _, bf := range r.Bits {
		sort.Slice(
			bf.Values,
			func(i, k int) bool {
				return bf.Values[i].Value < bf.Values[k].Value
			},
		)
	}
}

func handleIRQs(ctx *ctx, inst *Instance, sirqs []*svd.Interrupt) {
	for _, sirq := range sirqs {
		irq := &IRQ{Value: int(sirq.Value), Name: sirq.Name, Inst: inst}
		if sirq.Description != nil {
			irq.Descr = strings.TrimSpace(*sirq.Description)
		}
		ctx.irqmap[irq.Value] = append(ctx.irqmap[irq.Value], irq)
		inst.IRQs = append(inst.IRQs, irq)
	}
}

func saveIRQs(ctx *ctx) {
	irqs := make([]*IRQ, 0, len(ctx.irqmap))
	for _, is := range ctx.irqmap {
		if len(is) <= 1 {
			irqs = append(irqs, is...)
		} else {
			for k, irq := range is {
				irq.Shared = "+"
				for k1, irq1 := range is {
					if k != k1 && irq1.Inst.Periph != irq.Inst.Periph {
						irq.Shared = "*"
					}
				}
				name := is[0].Name
				for _, irq1 := range is[:k] {
					if irq1.Name == name {
						name = ""
						break
					}
				}
				if name != "" {
					irqs = append(irqs, irq)
				}
			}
		}
	}
	sort.Slice(irqs, func(i, j int) bool { return irqs[i].Value < irqs[j].Value })

	dir := ctx.push("irq")
	defer ctx.pop()
	mkdir(dir)
	w := create(filepath.Join(dir, ctx.mcu+".go"))
	defer w.Close()

	w.donotedit()
	fmt.Fprintln(w, "//go:build", ctx.mcu)
	fmt.Fprintln(w)
	fmt.Fprintln(
		w, "// Package irq provides the list of supported external interrupts.",
	)
	fmt.Fprintln(w, "package irq")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "const (")
	for _, irq := range irqs {
		fmt.Fprintf(w, "\t%s = %d", irq.Name, irq.Value)
		if irq.Descr != "" {
			fmt.Fprintln(w, "//", fixSpaces(irq.Descr))
		} else {
			fmt.Fprintln(w)
		}
	}
	fmt.Fprintln(w, ")")
}
