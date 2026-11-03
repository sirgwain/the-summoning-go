//go:build darwin && !ios

package cmd

import (
	"time"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

// Ebitengine's window icon does not apply on macOS, so the icon goes on the Dock. It must be set
// after the application has finished launching: until then an unbundled program has no Dock
// tile, and Ebitengine makes it a regular application only when launching ends, which gives the
// tile the default icon. So a goroutine waits for that and has the main thread set the icon.
func setPlatformIcon(png []byte) {
	data := objc.ID(objc.GetClass("NSData")).Send(objc.RegisterName("dataWithBytes:length:"), unsafe.Pointer(&png[0]), uintptr(len(png)))
	icon := objc.ID(objc.GetClass("NSImage")).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("initWithData:"), data)
	if icon == 0 {
		return
	}
	app := objc.ID(objc.GetClass("NSApplication")).Send(objc.RegisterName("sharedApplication"))
	running := objc.ID(objc.GetClass("NSRunningApplication")).Send(objc.RegisterName("currentApplication")).Send(objc.RegisterName("retain"))
	go func() {
		const regular = 0 // NSApplicationActivationPolicyRegular
		for range 200 {
			if objc.Send[bool](running, objc.RegisterName("isFinishedLaunching")) &&
				objc.Send[int](app, objc.RegisterName("activationPolicy")) == regular {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		// performSelectorOnMainThread retains the icon until it has been set.
		app.Send(objc.RegisterName("performSelectorOnMainThread:withObject:waitUntilDone:"),
			objc.RegisterName("setApplicationIconImage:"), icon, false)
		icon.Send(objc.RegisterName("release"))
		running.Send(objc.RegisterName("release"))
	}()
}
