// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"github.com/embeddedgo/tools/svd"
)

type MemGroup struct {
	Descr string
	Bases []*MemBase
}

func (g *MemGroup) WriteTo(w io.Writer) {
	if g.Descr != "" {
		fmt.Fprintln(w, "//", g.Descr)
	}
	fmt.Fprintln(w, "const (")
	for _, b := range g.Bases {
		fmt.Fprintf(w, "\t%s_BASE uintptr = %#X", b.Name, b.Addr)
		if b.Descr != "" {
			fmt.Fprintln(w, " //", b.Descr)
		} else {
			fmt.Fprintln(w)
		}
	}
	fmt.Fprintln(w, ")")
}

type MemBase struct {
	Name  string
	Addr  uint64
	Descr string
}

func saveMmap(ctx *ctx) {
	gmap := make(map[string]*MemGroup)
	for _, p := range ctx.spsli {
		var dp *svd.Peripheral
		if p.DerivedFrom != nil {
			dp = ctx.spmap[*p.DerivedFrom]
		}
		var gname string
		if p.GroupName != nil {
			gname = *p.GroupName
		} else if dp != nil && dp.GroupName != nil {
			gname = *dp.GroupName
		}
		g := gmap[gname]
		if g == nil {
			g = &MemGroup{Descr: gname}
			gmap[gname] = g
		}
		b := &MemBase{Name: p.Name, Addr: uint64(p.BaseAddress)}
		if p.Description != nil {
			b.Descr = *p.Description
		} else if dp != nil && dp.Description != nil {
			b.Descr = *dp.Description
		}
		b.Descr = fixSpaces(b.Descr)
		g.Bases = append(g.Bases, b)
	}
	gsli := make([]*MemGroup, len(gmap))
	k := 0
	for _, g := range gmap {
		sort.Slice(g.Bases, func(i, j int) bool { return pnameLess(g.Bases[i].Name, g.Bases[j].Name) })
		gsli[k] = g
		k++
	}
	sort.Slice(gsli, func(i, j int) bool { return gsli[i].Descr < gsli[j].Descr })

	dir := ctx.push("mmap")
	defer ctx.pop()
	mkdir(dir)
	w := create(filepath.Join(dir, ctx.mcu+".go"))
	defer w.Close()
	w.donotedit()
	fmt.Fprintln(w, "// +build", ctx.mcu)
	fmt.Fprintln(w)
	fmt.Fprintln(
		w, "// Package mmap provides base memory adresses for all peripherals.",
	)
	fmt.Fprintln(w, "package mmap")
	for _, g := range gsli {
		g.WriteTo(w)
	}
}
