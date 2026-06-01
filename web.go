package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// server держит настройки и реализует http.Handler через маршрутизатор.
type server struct {
	cfg   config
	ip    string // локальный IP в сети
	mdns  bool   // доступно ли имя ipdrop.local
	mux   *http.ServeMux
	icons map[string][]byte // предрендеренные иконки по пути
}

func newServer(cfg config, ip string, mdns bool) *server {
	s := &server{
		cfg:  cfg,
		ip:   ip,
		mdns: mdns,
		mux:  http.NewServeMux(),
		icons: map[string][]byte{
			"/apple-touch-icon.png": iconPNG(180),
			"/icon-192.png":         iconPNG(192),
			"/icon-512.png":         iconPNG(512),
		},
	}
	// Для телефона:
	s.mux.HandleFunc("/", s.handlePage)
	s.mux.HandleFunc("/upload", s.handleUpload)
	s.mux.HandleFunc("/qr.png", s.handleQR)
	s.mux.HandleFunc("/manifest.webmanifest", s.handleManifest)
	for path := range s.icons {
		s.mux.HandleFunc(path, s.handleIcon)
	}
	// Панель управления для окна на ПК:
	s.mux.HandleFunc("/panel", s.handlePanel)
	// Приватный API (только с этого же компьютера):
	s.mux.HandleFunc("/api/info", s.local(s.handleInfo))
	s.mux.HandleFunc("/api/files", s.local(s.handleFiles))
	s.mux.HandleFunc("/api/file", s.local(s.handleFileGet))
	s.mux.HandleFunc("/api/open-folder", s.local(s.handleOpenFolder))
	s.mux.HandleFunc("/api/autostart", s.local(s.handleAutostart))
	return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// networkURL — адрес, который надо показывать телефону (постоянный, если есть mDNS).
func (s *server) networkURL() string {
	if s.mdns {
		return fmt.Sprintf("http://%s.local:%d", stableHost, s.cfg.port)
	}
	return fmt.Sprintf("http://%s:%d", s.ip, s.cfg.port)
}

// local оборачивает обработчик так, что он доступен только с самого компьютера.
// Команды вроде «открыть папку» и «автозапуск» не должны быть доступны с телефона.
func (s *server) local(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}

// --- Страницы ---

func (s *server) handlePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, pageHTML)
}

func (s *server) handlePanel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, panelHTML)
}

func (s *server) handleQR(w http.ResponseWriter, r *http.Request) {
	// QR всегда ведёт на сетевой адрес (а не на localhost окна панели).
	png, err := qrcode.Encode(s.networkURL(), qrcode.Medium, 320)
	if err != nil {
		http.Error(w, "qr error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(png)
}

func (s *server) handleIcon(w http.ResponseWriter, r *http.Request) {
	png, ok := s.icons[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(png)
}

func (s *server) handleManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
	io.WriteString(w, webManifest)
}

// --- API (только localhost) ---

func (s *server) handleInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"name":      "IpDrop",
		"port":      s.cfg.port,
		"dir":       s.cfg.saveDir,
		"url":       s.networkURL(),
		"ipUrl":     fmt.Sprintf("http://%s:%d", s.ip, s.cfg.port),
		"stableUrl": fmt.Sprintf("http://%s.local:%d", stableHost, s.cfg.port),
		"mdns":      s.mdns,
		"autostart": autostartEnabled(),
	})
}

type fileEntry struct {
	Name  string `json:"name"`
	Size  int64  `json:"size"`
	MTime int64  `json:"mtime"` // unix-миллисекунды
	Image bool   `json:"image"`
}

func (s *server) handleFiles(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(s.cfg.saveDir)
	if err != nil {
		writeJSON(w, []fileEntry{})
		return
	}
	var files []fileEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileEntry{
			Name:  e.Name(),
			Size:  info.Size(),
			MTime: info.ModTime().UnixMilli(),
			Image: isImage(e.Name()),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].MTime > files[j].MTime })
	if len(files) > 200 {
		files = files[:200]
	}
	writeJSON(w, files)
}

// handleFileGet отдаёт принятый файл (для превью картинок в панели).
func (s *server) handleFileGet(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	// Защита от выхода за пределы папки приёма.
	if name == "" || name != filepath.Base(name) || strings.ContainsAny(name, `/\`) {
		http.Error(w, "bad name", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.cfg.saveDir, name))
}

func (s *server) handleOpenFolder(w http.ResponseWriter, r *http.Request) {
	// explorer.exe иногда возвращает код 1 даже при успехе — ошибку игнорируем.
	_ = exec.Command("explorer", s.cfg.saveDir).Start()
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) handleAutostart(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var body struct {
			Enable bool `json:"enable"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		var err error
		if body.Enable {
			err = installAutostart(s.cfg)
		} else {
			err = uninstallAutostart()
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	writeJSON(w, map[string]any{"enabled": autostartEnabled()})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

// --- Приём файлов ---

// handleUpload принимает файлы потоково (без загрузки в память целиком),
// чтобы спокойно переваривать большие фото и видео.
func (s *server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "ожидалась форма multipart", http.StatusBadRequest)
		return
	}

	var saved []string
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, "ошибка чтения данных", http.StatusBadRequest)
			return
		}
		if part.FileName() == "" {
			continue // не файловое поле
		}

		dstPath, n, err := s.saveFile(part)
		part.Close()
		if err != nil {
			log.Printf("не удалось сохранить %q: %v", part.FileName(), err)
			http.Error(w, "ошибка сохранения файла", http.StatusInternalServerError)
			return
		}
		saved = append(saved, filepath.Base(dstPath))
		log.Printf("принято: %s (%s)", filepath.Base(dstPath), humanSize(n))
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "Принято файлов: %d", len(saved))
}

// saveFile записывает один загружаемый файл в папку приёма,
// подбирая уникальное безопасное имя.
func (s *server) saveFile(part io.Reader) (string, int64, error) {
	pf, ok := part.(interface{ FileName() string })
	name := "file"
	if ok {
		name = pf.FileName()
	}
	dstPath := uniquePath(s.cfg.saveDir, sanitizeName(name))

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", 0, err
	}
	defer dst.Close()

	n, err := io.Copy(dst, part)
	if err != nil {
		return "", 0, err
	}
	return dstPath, n, nil
}

// --- Вспомогательное ---

func isImage(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".heic", ".heif":
		return true
	}
	return false
}

// sanitizeName убирает путь и опасные символы из имени, присланного клиентом.
func sanitizeName(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.TrimSpace(name)
	for _, c := range []string{":", "*", "?", "\"", "<", ">", "|"} {
		name = strings.ReplaceAll(name, c, "_")
	}
	if name == "" || name == "." || name == ".." {
		name = "file"
	}
	return name
}

// uniquePath возвращает путь, который ещё не занят: при совпадении
// добавляет суффикс « (1)», « (2)» перед расширением.
func uniquePath(dir, name string) string {
	candidate := filepath.Join(dir, name)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d Б", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cБ", float64(n)/float64(div), "КМГТ"[exp])
}
