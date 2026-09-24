package exec

import (
	"os/signal"
	"syscall"
)

func handleSIGQUIT() {
	signal.Ignore(syscall.SIGQUIT)
}
