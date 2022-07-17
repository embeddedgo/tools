// Copyright 2019 Michal Derkacz. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/embeddedgo/tools/svd"
)

var importRoot string

type ctx struct {
	devname  string
	spsli    []*svd.Peripheral
	spmap    map[string]*svd.Peripheral
	irqmap   map[int][]*IRQ
	instmap  map[string]*Instance
	defwidth uint
	mcu      string
	path     []string
}

func (ctx *ctx) push(dir string) string {
	ctx.path = append(ctx.path, dir)
	return filepath.Join(ctx.path...)
}

func (ctx *ctx) pop() {
	ctx.path = ctx.path[:len(ctx.path)-1]
}

func svdxgen(file string, wg *sync.WaitGroup) {
	ctx := new(ctx)

	ctx.mcu = filepath.Base(file)
	if i := strings.LastIndexByte(ctx.mcu, '.'); i >= 0 {
		ctx.mcu = ctx.mcu[:i]
	}
	ctx.mcu = strings.ToLower(ctx.mcu)

	data, err := ioutil.ReadFile(file)
	dieErr(err)

	dev := new(svd.Device)
	dieErr(xml.Unmarshal(data, &dev))

	ctx.devname = dev.Name
	ctx.spsli = dev.Peripherals
	ctx.spmap = make(map[string]*svd.Peripheral, len(ctx.spsli))
	ctx.instmap = make(map[string]*Instance)
	ctx.irqmap = make(map[int][]*IRQ)
	for _, p := range ctx.spsli {
		ctx.spmap[p.Name] = p
	}
	ctx.defwidth = uint(dev.Width)
	if dev.RegisterPropertiesGroup != nil && dev.Size != nil {
		ctx.defwidth = uint(*dev.Size)
	}

	saveMmap(ctx)
	savePeriphs(ctx)

	wg.Done()
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: svdxgen IMPORT_ROOT SVD_FILE...")
		os.Exit(1)
	}
	importRoot = os.Args[1]
	files := os.Args[2:]
	wg := new(sync.WaitGroup)
	wg.Add(len(files))
	for _, file := range files {
		go svdxgen(file, wg)
	}
	wg.Wait()
}
