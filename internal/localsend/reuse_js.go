//go:build js

package localsend

import "syscall"

// reuseControl does nothing where there are no sockets to share.
func reuseControl(_, _ string, _ syscall.RawConn) error { return nil }
