package main

import "flag"

// config — настройки запуска, задаются флагами командной строки.
type config struct {
	port    int
	saveDir string

	installAutostart   bool
	uninstallAutostart bool
}

func parseFlags() config {
	var cfg config
	flag.IntVar(&cfg.port, "port", 8080, "TCP-порт веб-сервера")
	flag.StringVar(&cfg.saveDir, "dir", defaultSaveDir(), "папка для сохранения принятых файлов")
	flag.BoolVar(&cfg.installAutostart, "install-autostart", false, "добавить приложение в автозапуск Windows и выйти")
	flag.BoolVar(&cfg.uninstallAutostart, "uninstall-autostart", false, "убрать приложение из автозапуска Windows и выйти")
	flag.Parse()
	return cfg
}
