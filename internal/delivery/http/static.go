//go:build !dev

package http

import (
	"dispatcher/web"
	"fmt"
	"io"
	"net/http"
)

func StaticHandler() http.Handler {
	fs := http.FS(web.StaticFiles)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" || path == "" {
			// Отдаём index.html при корневом запросе
			file, err := fs.Open("dist/index.html")
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			defer file.Close()
			stat, _ := file.Stat()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
			_, _ = io.Copy(w, file)
			return
		}

		// Всегда ищем файл внутри dist
		filePath := "dist" + path
		file, err := fs.Open(filePath)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer file.Close()
		stat, _ := file.Stat()
		// Простейшее определение Content-Type
		if ext := getContentType(path); ext != "" {
			w.Header().Set("Content-Type", ext)
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
		_, _ = io.Copy(w, file)
	})
}

func getContentType(path string) string {
	// Можно использовать mime.TypeByExtension, но для простоты:
	if len(path) > 4 && path[len(path)-4:] == ".css" {
		return "text/css; charset=utf-8"
	}
	if len(path) > 3 && path[len(path)-3:] == ".js" {
		return "application/javascript"
	}
	if len(path) > 4 && path[len(path)-4:] == ".svg" {
		return "image/svg+xml"
	}
	if len(path) > 5 && path[len(path)-5:] == ".html" {
		return "text/html; charset=utf-8"
	}
	return ""
}
