// IpDrop — простой приёмник файлов с iPhone (и любого устройства) на Windows-ПК.
//
// Идея: ПК поднимает локальную веб-страницу. На телефоне один раз открываете адрес
// (по QR-коду или по постоянному имени ipdrop.local), добавляете значок на экран
// «Домой» — и дальше отправляете фото/файлы одним тапом, без сканирования.
// Работает по Wi-Fi в пределах одной сети.
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
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

// stableHost — постоянное имя в локальной сети (через mDNS/Bonjour).
// Адрес ipdrop.local не зависит от меняющегося IP компьютера.
const stableHost = "ipdrop"

func main() {
	cfg := parseFlags()

	// Команды установки/удаления автозапуска не запускают сервер.
	switch {
	case cfg.installAutostart:
		mustOK(installAutostart(cfg))
		fmt.Println("✓ Автозапуск включён. IpDrop будет стартовать при входе в Windows.")
		return
	case cfg.uninstallAutostart:
		mustOK(uninstallAutostart())
		fmt.Println("✓ Автозапуск выключен.")
		return
	}

	if err := os.MkdirAll(cfg.saveDir, 0o755); err != nil {
		log.Fatalf("не удалось создать папку для приёма %q: %v", cfg.saveDir, err)
	}

	ip := lanIP()
	ipURL := fmt.Sprintf("http://%s:%d", ip, cfg.port)
	stableURL := fmt.Sprintf("http://%s.local:%d", stableHost, cfg.port)

	// Публикуем постоянное имя ipdrop.local (необязательно — если не выйдет,
	// продолжаем работать по IP-адресу).
	mdnsOK := true
	if stop, err := startMDNS(cfg.port, ip); err != nil {
		log.Printf("mDNS не запустился (адрес ipdrop.local будет недоступен): %v", err)
		mdnsOK = false
	} else {
		defer stop()
	}

	// QR ведёт на постоянный адрес, чтобы значок на «Домой» пережил смену IP.
	// Если mDNS недоступен — на обычный IP-адрес.
	qrURL := stableURL
	if !mdnsOK {
		qrURL = ipURL
	}

	printBanner(cfg, stableURL, ipURL, qrURL, mdnsOK)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.port),
		Handler:           newServer(cfg),
		ReadHeaderTimeout: 15 * time.Second,
		// Без общего ReadTimeout/WriteTimeout — иначе большие видео обрываются.
	}
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("сервер остановлен: %v", err)
	}
}

// printBanner выводит инструкцию и QR-код прямо в консоль,
// чтобы можно было сразу навести камеру телефона.
func printBanner(cfg config, stableURL, ipURL, qrURL string, mdnsOK bool) {
	line := strings.Repeat("─", 54)
	fmt.Println(line)
	fmt.Println("  📥  IpDrop запущен")
	fmt.Println(line)
	if mdnsOK {
		fmt.Printf("  Постоянный адрес (для значка) : %s\n", stableURL)
		fmt.Printf("  Запасной по IP                : %s\n", ipURL)
	} else {
		fmt.Printf("  Адрес для телефона : %s\n", ipURL)
	}
	fmt.Printf("  Папка приёма       : %s\n", cfg.saveDir)
	fmt.Println(line)
	fmt.Println("  1) На iPhone откройте «Камеру» и наведите на QR-код.")
	fmt.Println("  2) На странице нажмите «Поделиться» → «На экран Домой» —")
	fmt.Println("     появится значок IpDrop, сканировать больше не нужно.")
	fmt.Println(line)

	if qr, err := qrcode.New(qrURL, qrcode.Medium); err == nil {
		fmt.Println(qr.ToSmallString(false))
	}

	fmt.Println("  (При первом запуске Windows спросит про брандмауэр —")
	fmt.Println("   разрешите доступ для «Частных сетей».)")
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
	// Запасной вариант: перебор интерфейсов.
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
