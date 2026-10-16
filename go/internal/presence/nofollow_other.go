//go:build !unix

package presence

// nofollow is nothing where the platform has no such open flag. The Lstat
// refusal still holds there, and the app the record is for runs on macOS.
const nofollow = 0
