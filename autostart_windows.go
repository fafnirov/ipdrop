package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

// runKeyPath — стандартная ветка автозапуска для текущего пользователя
// (не требует прав администратора).
const (
	runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValue   = "AirDropReceiver"
)

// installAutostart прописывает запуск приложения при входе в Windows,
// сохраняя выбранные порт и папку приёма. Стартует в фоновом режиме (без окна),
// чтобы при входе не выскакивало окно — панель открывается по двойному клику.
func installAutostart(cfg config) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("не удалось определить путь к программе: %w", err)
	}

	// Команда автозапуска с теми же настройками, что и сейчас.
	cmd := fmt.Sprintf(`"%s" -headless -port %d -dir "%s"`, exe, cfg.port, cfg.saveDir)

	key, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("не удалось открыть ветку реестра автозапуска: %w", err)
	}
	defer key.Close()

	if err := key.SetStringValue(runValue, cmd); err != nil {
		return fmt.Errorf("не удалось записать значение автозапуска: %w", err)
	}
	return nil
}

// uninstallAutostart убирает запись автозапуска. Отсутствие записи — не ошибка.
func uninstallAutostart() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	defer key.Close()

	if err := key.DeleteValue(runValue); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

// autostartEnabled сообщает, прописан ли автозапуск сейчас.
func autostartEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	_, _, err = key.GetStringValue(runValue)
	return err == nil
}
