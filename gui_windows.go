package main

import (
	"log"
	"os/exec"

	webview "github.com/jchv/go-webview2"
)

// openWindow открывает окно приложения с панелью управления (через встроенный
// в Windows WebView2). Блокирует выполнение, пока окно открыто.
//
// Возвращает true, если окно удалось создать и отработать. Если WebView2
// недоступен — возвращает false (вызывающий код откроет панель в браузере).
func openWindow(url string) (ok bool) {
	ok = true
	defer func() {
		if r := recover(); r != nil {
			log.Printf("не удалось открыть окно WebView2: %v", r)
			ok = false
		}
	}()

	w := webview.NewWithOptions(webview.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		WindowOptions: webview.WindowOptions{
			Title:  "IpDrop",
			Width:  880,
			Height: 660,
			Center: true,
		},
	})
	if w == nil {
		return false
	}
	defer w.Destroy()
	w.Navigate(url)
	w.Run() // блокирует до закрытия окна
	return true
}

// openBrowser открывает URL в браузере по умолчанию (запасной вариант).
func openBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
