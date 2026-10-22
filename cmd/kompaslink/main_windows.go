package main

import (
	"os"
	"runtime"

	"github.com/lxn/walk"

	"kompaslink/internal/ui"
)

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := ui.Run(); err != nil {
		walk.MsgBox(nil, "KompasLink", "Не удалось открыть окно: "+err.Error(), walk.MsgBoxIconError)
		os.Exit(1)
	}
}
