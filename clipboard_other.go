//go:build !windows

package main

func copyToClipboard(s string) bool { return false }
