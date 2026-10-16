//go:build unix

package presence

import "syscall"

// nofollow is the flag that makes the open refuse a symlink at the path. The
// Lstat before it is the readable refusal; this is the one that holds when
// the link appears between the two.
const nofollow = syscall.O_NOFOLLOW
