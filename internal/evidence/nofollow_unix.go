//go:build !windows

package evidence

import "syscall"

// openFlags refuse a symlink at the final path component and keep a FIFO from blocking open.
const openFlags = syscall.O_NOFOLLOW | syscall.O_NONBLOCK
