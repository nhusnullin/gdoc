//go:build !unix

package main

// lockIsFree is nothing where the platform has no flock. The app is macOS
// only and openPresenceApp asks the platform first, so this answer is never
// what decides anything: it is here so the one binary builds for the three
// platforms the release carries.
func lockIsFree(string) bool { return false }
