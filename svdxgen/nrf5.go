// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "strings"

func nrf5tweaks(gs []*Group) {
	for _, g := range gs {
		for _, p := range g.Periphs {
			for _, r := range p.Regs {
				nfr5te(r.Bits)
			}
		}
	}
}

func nfr5te(bfs []*BitField) {
	for i, bf := range bfs {
		if strings.HasPrefix(bf.Name, "TASKS_") || bf.Name == "Trigger" ||
			strings.HasPrefix(bf.Name, "EVENTS_") ||
			bf.Name == "NotGenerated" || bf.Name == "Generated" {
			bfs[i] = nil
		}
	}

}
