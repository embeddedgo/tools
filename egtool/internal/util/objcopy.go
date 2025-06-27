// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Section struct {
	Vaddr  uint64 // address in the memory during execution
	Paddr  uint64 // phisical location of the section in the Flash/ROM
	Offset uint64 // offset in the ELF file to the beggining of the section data
	Data   []byte // section data
}

type Sections []*Section

// ReadELF reads the loadable sections of the program and returns them as
// a slice. The order of the returned sections is unspecified.
func ReadELF(name string) (Sections, error) {
	r, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	f, err := elf.NewFile(r)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ss := make(Sections, 0, 16)
	for i, s := range f.Sections {
		if s.Type != elf.SHT_PROGBITS || s.Flags&elf.SHF_ALLOC == 0 {
			if k := i + 1; k < len(f.Sections) && len(ss) != 0 {
				ns := f.Sections[k]
				if ns.Type == elf.SHT_PROGBITS && ns.Flags&elf.SHF_ALLOC != 0 {
					// Log the non-loadable sections between loadable ones.
					// TODO: elimenate/reorder such sections in go linker
					Warn(
						"readelf: skipping section '%s' (%d bytes)\n",
						s.Name, ns.Size,
					)
				}
			}
			continue
		}
		data, err := s.Data()
		if err != nil {
			return nil, err
		}
		if len(data) == 0 {
			continue
		}
		paddr := ^uint64(0)
		for _, p := range f.Progs {
			if p.Type != elf.PT_LOAD {
				continue
			}
			if p.Off <= s.Offset && s.Offset < p.Off+p.Filesz {
				paddr = p.Paddr + s.Offset - p.Off
				break
			}
		}
		ss = append(ss, &Section{s.Addr, paddr, s.Offset, data})
	}
	return ss, nil
}

// ReadBins reads binary files acording to the description and returns them
// as a slice of sections.
func ReadBins(descr string) (Sections, error) {
	bins := strings.Split(descr, ",")
	ss := make(Sections, len(bins))
	for k, ba := range bins {
		i := strings.LastIndexByte(ba, ':')
		if i <= 0 {
			return nil, fmt.Errorf("bad '%s' in the -inc option", ba)
		}
		bin, addr := ba[:i], ba[i+1:]
		s := new(Section)
		var err error
		s.Paddr, err = strconv.ParseUint(addr, 0, 64)
		if err != nil {
			return nil, fmt.Errorf("bad address in '%s': %s", addr, err)
		}
		s.Data, err = os.ReadFile(bin)
		if err != nil {
			return nil, err
		}
		ss[k] = s
	}
	return ss, nil
}

// SortByPaddr sorts sections according to the Paddr field.
func (ss Sections) SortByPaddr() {
	sort.Slice(
		ss,
		func(i, j int) bool {
			return ss[i].Paddr < ss[j].Paddr
		},
	)
}

// Flatten flattens sections by writting their data to the provided io.Writer
// according to the Paddr field (before writting the sections are sorted using
// SortPaddr method). The gaps between sections are filled using the pad byte.
func (ss Sections) Flatten(w io.Writer, pad byte) (n int, err error) {
	if len(ss) == 0 {
		return
	}
	ss.SortByPaddr()
	pa := ss[0].Paddr
	n, err = w.Write(ss[0].Data)
	if err != nil {
		return
	}
	pa += uint64(n)
	var padCache []byte
	for _, s := range ss[1:] {
		if s.Paddr < pa {
			err = errors.New("flatten: overlaping sections")
			return
		}
		m := int(s.Paddr - pa)
		if m != 0 {
			m, err = w.Write(PadBytes(&padCache, m, pad))
			n += m
			if err != nil {
				return
			}
			pa += uint64(m)
		}
		m, err = w.Write(s.Data)
		n += n
		if err != nil {
			return
		}
		pa += uint64(m)
	}
	return
}

// PadBytes returns the slice containing n byte equal b.
func PadBytes(cache *[]byte, n int, b byte) []byte {
	if len(*cache) < n {
		*cache = make([]byte, n)
		for i := range *cache {
			(*cache)[i] = b
		}
	}
	return (*cache)[:n]
}
