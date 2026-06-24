//go:build !darwin

package main

// clearNativeChrome is a no-op outside macOS; Linux transparency is handled by
// the WindowIsTranslucent option.
func clearNativeChrome() {}
