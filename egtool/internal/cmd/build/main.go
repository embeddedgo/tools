// Copyright 2025 The Embedded Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package build

import (
	"os"
	"os/exec"

	"github.com/embeddedgo/tools/egtool/internal/util"
)

const Descr = "run `go build` with GOENV set to the found go.env file"

func Main(cmd string, args []string) {
	util.SetGOENV()
	goCmd, err := exec.LookPath("go")
	util.FatalErr("", err)
	c := &exec.Cmd{
		Path:   goCmd,
		Args:   append([]string{goCmd, cmd}, args...),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	err = c.Run()
	if err == nil {
		return
	}
	if ee, ok := err.(*exec.ExitError); ok {
		os.Exit(ee.ProcessState.ExitCode())
	}
	util.FatalErr("", err)
}
