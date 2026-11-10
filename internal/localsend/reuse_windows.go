//go:build windows

package localsend

import "syscall"

// reuseControl shares the multicast port with another LocalSend app.
func reuseControl(_, _ string, c syscall.RawConn) error {
	var out error
	if err := c.Control(func(fd uintptr) {
		out = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	}); err != nil {
		return err
	}
	return out
}
