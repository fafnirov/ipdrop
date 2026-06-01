package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// server держит настройки и реализует http.Handler через маршрутизатор.
type server struct {
	cfg   config
	mux   *http.ServeMux
	icons map[string][]byte // предрендеренные иконки по пути
}

func newServer(cfg config) *server {
	s := &server{
		cfg: cfg,
		mux: http.NewServeMux(),
		icons: map[string][]byte{
			"/apple-touch-icon.png": iconPNG(180),
			"/icon-192.png":         iconPNG(192),
			"/icon-512.png":         iconPNG(512),
		},
	}
	s.mux.HandleFunc("/", s.handlePage)
	s.mux.HandleFunc("/upload", s.handleUpload)
	s.mux.HandleFunc("/qr.png", s.handleQR)
	s.mux.HandleFunc("/manifest.webmanifest", s.handleManifest)
	for path := range s.icons {
		s.mux.HandleFunc(path, s.handleIcon)
	}
	return s
}

// handleIcon отдаёт предрендеренную PNG-иконку.
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

// handleManifest отдаёт манифест PWA.
func (s *server) handleManifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
	io.WriteString(w, webManifest)
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// handlePage отдаёт страницу для выбора и отправки файлов.
func (s *server) handlePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, pageHTML)
}

// handleQR отдаёт QR-код текущего адреса как PNG (для показа на самой странице).
func (s *server) handleQR(w http.ResponseWriter, r *http.Request) {
	url := "http://" + r.Host
	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "qr error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

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

// sanitizeName убирает путь и опасные символы из имени, присланного клиентом.
func sanitizeName(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.TrimSpace(name)
	// Запрещённые в Windows символы имени файла.
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
