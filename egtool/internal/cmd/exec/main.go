// Copyright 2026 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package exec

import (
	"debug/elf"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/embeddedgo/tools/egtool/internal/util"
)

const Descr = "execute the compiled program in a semihosting enviroment"

func Main(cmd string, args []string) {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(
			os.Stderr,
			"Usage:\n  %s [OPTIONS] ELF [ARGS...]\nOptions:\n",
			cmd,
		)
		fs.PrintDefaults()
	}
	arch := fs.String("exec.arch", "auto", "use the `architecture` specific execution environment")
	fs.Parse(args)
	if fs.NArg() == 0 {
		fs.Usage()
		os.Exit(1)
	}
	elfbin := fs.Arg(0)
	var execArgs []string
	semiconf := "enable=on,target=native,userspace=on"
	for _, a := range fs.Args() {
		semiconf += ",arg=" + a
	}
	if *arch == "auto" {
		f, err := elf.Open(elfbin)
		util.FatalErr("cannot open ELF binary", err)
		h := f.FileHeader
		f.Close()
		switch h.Machine {
		case elf.EM_ARM:
			if h.Entry&1 != 0 {
				*arch = "thumb"
			}
		case elf.EM_RISCV:
			if h.Entry == 0x80000000 {
				*arch = "riscv64"
			}
		default:
			util.Fatal("unknown ELF architecture code: %#x or entry address: %#x")
		}
	}
	switch *arch {
	case "thumb":
		execArgs = []string{
			"qemu-system-arm",
			"-machine", "mps2-an500",
			"-cpu", "cortex-m7",
			"-nographic",
			"-monitor", "none",
			"-serial", "none",
			"--semihosting-config", semiconf,
			"-kernel", elfbin,
		}
	case "riscv64":
		execArgs = []string{
			"qemu-system-riscv64",
			"-machine", "virt",
			"-cpu", "rv64,pmp=false,mmu=false,c=true",
			"-smp", "2",
			"-m", "32",
			"-nographic",
			"-monitor", "none",
			"-serial", "none",
			"--semihosting-config", semiconf,
			"-bios", elfbin,
		}
	default:
		util.Fatal("unknow architecture: %s", *arch)
	}
	//fmt.Println(execArgs)
	path, err := exec.LookPath(execArgs[0])
	util.FatalErr(execArgs[0], err)
	execCmd := exec.Cmd{
		Path:   path,
		Args:   execArgs,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if execCmd.Run() != nil {
		os.Exit(1)
	}
}
