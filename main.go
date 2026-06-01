// IpDrop — простой приёмник файлов с iPhone (и любого устройства) на Windows-ПК.
//
// Идея: ПК поднимает локальную веб-страницу. На телефоне один раз открываете адрес
// (по QR-коду или по постоянному имени ipdrop.local), добавляете значок на экран
// «Домой» — и дальше отправляете фото/файлы одним тапом, без сканирования.
// Работает по Wi-Fi в пределах одной сети.
//
// По умолчанию открывается окно-панель (через WebView2). Флаг -headless запускает
// приём в фоне без окна (используется для автозапуска при входе в Windows).
//
// Это НЕ настоящий AirDrop (протокол AWDL закрыт Apple и на Windows недоступен),
// а открытый аналог в духе LocalSend, который реально работает на Windows.
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

// stableHost — постоянное имя в локальной сети (через mDNS/Bonjour).
// Адрес ipdrop.local не зависит от меняющегося IP компьютера.
const stableHost = "ipdrop"

func main() {
	attachParentConsole() // показать вывод, если запущено из терминала
	cfg := parseFlags()

	// Команды установки/удаления автозапуска не запускают сервер.
	switch {
	case cfg.installAutostart:
		mustOK(installAutostart(cfg))
		fmt.Println("✓ Автозапуск включён. IpDrop будет стартовать в фоне при входе в Windows.")
		return
	case cfg.uninstallAutostart:
		mustOK(uninstallAutostart())
		fmt.Println("✓ Автозапуск выключен.")
		return
	}

	if err := os.MkdirAll(cfg.saveDir, 0o755); err != nil {
		log.Fatalf("не удалось создать папку для приёма %q: %v", cfg.saveDir, err)
	}

	// Пытаемся занять порт. Кто первый занял — поднимает сервер;
	// второй запуск (например, двойной клик при работающем фоне) просто
	// откроет окно, подключённое к уже работающему экземпляру.
	ln, bindErr := net.Listen("tcp", fmt.Sprintf(":%d", cfg.port))
	primary := bindErr == nil

	if primary {
		ip := lanIP()

		mdnsOK := false
		if stop, err := startMDNS(cfg.port, ip); err == nil {
			mdnsOK = true
			defer stop()
		} else {
			log.Printf("mDNS не запустился (адрес ipdrop.local недоступен): %v", err)
		}

		srv := newServer(cfg, ip, mdnsOK)
		httpServer := &http.Server{
			Handler:           srv,
			ReadHeaderTimeout: 15 * time.Second,
			// Без общего ReadTimeout/WriteTimeout — иначе большие видео обрываются.
		}

		if cfg.headless {
			printBanner(srv)
			if err := httpServer.Serve(ln); err != nil {
				log.Fatalf("сервер остановлен: %v", err)
			}
			return
		}

		// Режим с окном: сервер работает в фоне, пока открыто окно панели.
		go func() {
			if err := httpServer.Serve(ln); err != nil {
				log.Printf("сервер остановлен: %v", err)
			}
		}()
	} else if cfg.headless {
		log.Fatalf("IpDrop уже запущен (порт %d занят).", cfg.port)
	}

	// --- Режим с окном ---
	runtime.LockOSThread() // WebView2 должен жить на главном потоке
	panelURL := fmt.Sprintf("http://127.0.0.1:%d/panel", cfg.port)
	if !openWindow(panelURL) {
		// WebView2 недоступен — открываем панель в браузере.
		openBrowser(panelURL)
		if primary {
			log.Printf("Окно недоступно, панель открыта в браузере: %s", panelURL)
			select {} // держим процесс живым, чтобы приём продолжал работать
		}
	}
}

// printBanner выводит инструкцию и QR-код в консоль (фоновый режим).
func printBanner(srv *server) {
	line := strings.Repeat("─", 54)
	fmt.Println(line)
	fmt.Println("  📥  IpDrop запущен (фоновый режим)")
	fmt.Println(line)
	if srv.mdns {
		fmt.Printf("  Постоянный адрес (для значка) : %s\n", srv.networkURL())
		fmt.Printf("  Запасной по IP                : http://%s:%d\n", srv.ip, srv.cfg.port)
	} else {
		fmt.Printf("  Адрес для телефона : %s\n", srv.networkURL())
	}
	fmt.Printf("  Папка приёма       : %s\n", srv.cfg.saveDir)
	fmt.Println(line)

	if qr, err := qrcode.New(srv.networkURL(), qrcode.Medium); err == nil {
		fmt.Println(qr.ToSmallString(false))
	}
	fmt.Println("  Остановить: Ctrl+C")
	fmt.Println(line)
}

// lanIP возвращает локальный IP-адрес ПК в Wi-Fi/LAN сети.
// Используем UDP-«дозвон» к внешнему адресу — пакеты не отправляются,
// но ОС выбирает исходящий интерфейс, чей адрес нам и нужен.
func lanIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		if a, ok := conn.LocalAddr().(*net.UDPAddr); ok && a.IP != nil {
			return a.IP.String()
		}
	}
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ip4 := ipnet.IP.To4(); ip4 != nil {
					return ip4.String()
				}
			}
		}
	}
	return "127.0.0.1"
}

func mustOK(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// defaultSaveDir — папка приёма по умолчанию: <домашняя>\IpDrop.
func defaultSaveDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "IpDrop")
	}
	return "received"
}
