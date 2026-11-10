//go:build !windows && !js

package localsend

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// reuseControl lets LocalSend's multicast port be bound by several apps,
// as the official app and this one may run side by side.
func reuseControl(_, _ string, c syscall.RawConn) error {
	var out error
	if err := c.Control(func(fd uintptr) {
		if err := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
			out = err
			return
		}
		// Not every Unix has it; losing the port to another app is not
		// fatal, as announcements still reach peers.
		_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	}); err != nil {
		return err
	}
	return out
}
