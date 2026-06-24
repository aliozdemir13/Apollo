//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

// makeWindowsTransparent forces every app window to be non-opaque with a clear
// background and no shadow. Wails v2 leaves the NSWindow opaque, which on recent
// macOS renders the system "Liquid Glass" backing material (and a rectangular
// shadow) behind our rounded, transparent widget. This removes it entirely.
static void makeWindowsTransparent(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        for (NSWindow *w in [NSApp windows]) {
            [w setOpaque:NO];
            [w setHasShadow:NO];
            [w setBackgroundColor:[NSColor clearColor]];
            [w invalidateShadow];
        }
    });
}
*/
import "C"

// clearNativeChrome strips the opaque backing + shadow from the window.
func clearNativeChrome() {
	C.makeWindowsTransparent()
}
