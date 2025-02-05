// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

type reg struct {
	Name    string
	Type    string
	NewT    bool
	BitSiz  int
	Len     int
	Offset  uint64
	Bits    []string
	SubRegs []*reg
	BitRegs []*reg
}

func regNameTypeLen(types map[string]bool, f, name string) (rname, typ string, newt bool, length int) {
	if name[len(name)-1] == ']' {
		i := strings.LastIndexByte(name, '[')
		if i <= 0 {
			fdie(f, "bad register format: %s", name)
		}
		l, err := strconv.ParseUint(name[i+1:len(name)-1], 0, 0)
		if err != nil {
			fdie(f, "bad array length in %s", name)
		}
		length = int(l)
		name = name[:i]
	}
	if name[len(name)-1] == '}' {
		i := strings.LastIndexByte(name, '{')
		if i <= 0 {
			fdie(f, "bad register format: %s", name)
		}
		typ = name[:i]
	} else if name[len(name)-1] == ')' {
		i := strings.LastIndexByte(name, '(')
		if i <= 0 {
			fdie(f, "bad register format: %s", name)
		}
		typ = name[i+1 : len(name)-1]
		name = name[:i]
	}
	if typ == "" {
		typ = name
	}
	newt = !types[typ]
	if newt {
		types[typ] = true
	}
	return name, typ, newt, length
}

func registers(f string, lines []string, decls []ast.Decl) ([]*reg, []string) {
	var (
		regs    []*reg
		nextoff uint64
	)
	types := map[string]bool{
		"int64": true, "int32": true, "int16": true, "int8": true,
		"uint64": true, "uint32": true, "uint16": true, "uint8": true,
	}
loop:
	for len(lines) > 0 {
		line := strings.TrimSpace(lines[0])
		switch line {
		case "Import:", "Instances:":
			break loop
		}
		lines = lines[1:]
		if line == "" {
			continue
		}
		offstr, line := split(line)
		sizstr, line := split(line)
		name, _ := split(line)
		switch "" {
		case offstr:
			fdie(f, "no register offset")
		case sizstr:
			fdie(f, "no register bit size")
		case name:
			fdie(f, "no register name")
		}
		var size int
		switch sizstr {
		case "64":
			size = 8
		case "32":
			size = 4
		case "16":
			size = 2
		case "8":
			size = 1
		default:
			fdie(f, "bad register size %s: not 8, 16, 32, 64", sizstr)
		}
		name, typ, newt, length := regNameTypeLen(types, f, name)
		var subregs []*reg
		if name[len(name)-1] == '}' {
			n := strings.IndexByte(name, '{')
			if n <= 0 {
				fdie(f, "bad register name: %s", name)
			}
			for _, sname := range strings.Split(name[n+1:len(name)-1], ",") {
				sname, styp, snewt, slen := regNameTypeLen(types, f, sname)
				if sname == "_" {
					sname = ""
				}
				sr := &reg{
					Name:   sname,
					Type:   styp,
					NewT:   snewt,
					BitSiz: size * 8,
					Len:    slen,
				}
				subregs = append(subregs, sr)
			}
			name = name[:n]
		}
		offset, err := strconv.ParseUint(offstr, 0, 64)
		if err != nil {
			fdie(f, "%s: bad offset %s: %v", name, offstr, err)
		}
		if offset&uint64(size-1) != 0 || offset < nextoff {
			fdie(f, "%s: bad offset %s for %s-bit register", name, offstr, sizstr)
		}
		for offset > nextoff {
			siz := 4
			for nextoff+uint64(siz) > offset || nextoff&uint64(siz-1) != 0 {
				siz >>= 1
			}
			var lastres *reg
			if len(regs) > 0 {
				lastres = regs[len(regs)-1]
				if lastres.Name != "" || lastres.BitSiz != siz*8 {
					lastres = nil
				}
			}
			if lastres != nil {
				if lastres.Len == 0 {
					lastres.Len = 2
				} else {
					lastres.Len++
				}
			} else {
				regs = append(regs, &reg{
					BitSiz: siz * 8,
					Offset: nextoff,
				})
			}
			nextoff += uint64(siz)
		}
		r := &reg{
			Name:    name,
			Type:    typ,
			NewT:    newt,
			BitSiz:  size * 8,
			Len:     length,
			Offset:  offset,
			SubRegs: subregs,
			BitRegs: subregs,
		}
		if len(subregs) == 0 {
			r.BitRegs = []*reg{r}
		}
		if length == 0 {
			length = 1
		}
		nextoff += uint64(size) * uint64(length) * uint64(len(r.BitRegs))
		regs = append(regs, r)
	}
	regmap := make(map[string]*reg)
	for _, r := range regs {
		for _, br := range r.BitRegs {
			regmap[br.Name] = br
		}
	}
	for _, d := range decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, s := range g.Specs {
			v := s.(*ast.ValueSpec)
			t, ok := v.Type.(*ast.Ident)
			if !ok {
				continue
			}
			r := regmap[t.Name]
			if r == nil {
				continue
			}
			if v.Comment == nil {
				continue
			}
			var n int
			for cl := v.Comment.List; n < len(v.Comment.List); n++ {
				if c := cl[n]; c != nil && strings.HasPrefix(c.Text, "//+") {
					break
				}
			}
			if n == len(v.Comment.List) {
				continue
			}
			for _, id := range v.Names {
				if id.Name != "_" {
					r.Bits = append(r.Bits, id.Name)
				}
			}
		}
	}
	if generics {
		for _, r := range regs {
			sr := r.SubRegs
			i := 0
			for {
				if i < len(sr) {
					r = sr[i]
				}
				if len(r.Bits) == 0 && r.Type == r.Name {
					// Avoid new type and use raw uintN register instead.
					r.Type = "uint" + strconv.Itoa(r.BitSiz)
					r.NewT = false
				}
				if i++; i >= len(sr) {
					break
				}
			}
		}
	}
	return regs, lines
}
