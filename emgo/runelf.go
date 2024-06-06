// Copyright 2024 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"debug/elf"
	"os"
	"os/exec"
)

func runELF(elfbin string) int {
	// TODO: use debug/buildinfo.ReadFile if it'll support buildinfo in RODATA
	f, err := elf.Open(elfbin)
	dieErr(err)
	h := f.FileHeader
	f.Close()
	var args []string
	switch h.Machine {
	case elf.EM_ARM:
		if h.Entry&1 != 0 {
			args = []string{
				"qemu-system-arm",
				"-machine", "mps2-an500",
				"-cpu", "cortex-m7",
				"-nographic",
				"-monitor", "none",
				"-serial", "none",
				"--semihosting-config", "enable=on,target=native,userspace=on",
				"-kernel", elfbin,
			}
		}
	case elf.EM_RISCV:
		if h.Entry == 0x80000000 {
			args = []string{
				"qemu-system-riscv64",
				"-machine", "virt",
				"-cpu", "rv64,pmp=false,mmu=false,c=false",
				"-smp", "2",
				"-m", "32",
				"-nographic",
				"-monitor", "none",
				"-serial", "none",
				"--semihosting-config", "enable=on,target=native,userspace=on",
				"-bios", elfbin,
			}
		}
	}
	if len(args) == 0 {
		die(elfbin + ": unknown ELF image")
	}
	path, err := exec.LookPath(args[0])
	dieErr(err)
	cmd := &exec.Cmd{
		Path:   path,
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if cmd.Run() != nil {
		return 1
	}
	return 0
}
