package main

import (
	"log"
	"os"

	"golang.org/x/sys/windows"
)

// Приложение собирается как GUI (-H windowsgui), поэтому при двойном клике
// чёрное окно консоли не появляется. Но если программу запустили из терминала
// (например, с флагом -install-autostart), вывод всё равно нужно показать —
// для этого прицепляемся к консоли родительского процесса.
var procAttachConsole = windows.NewLazySystemDLL("kernel32.dll").NewProc("AttachConsole")

func attachParentConsole() {
	const attachParent = ^uintptr(0) // ATTACH_PARENT_PROCESS (DWORD -1)
	if r, _, _ := procAttachConsole.Call(attachParent); r == 0 {
		return // родительской консоли нет (запуск двойным кликом) — это нормально
	}
	if h, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0); err == nil {
		os.Stdout = h
		os.Stderr = h
		log.SetOutput(h)
	}
}
