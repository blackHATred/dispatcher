package http

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
)

type SSEConfig struct {
	CORS string
}

func RegisterSSEHandler(e *echo.Echo, config SSEConfig, pointsChan <-chan [][]float32) {
	e.GET("/sse", func(c echo.Context) error {
		log.Printf("SSE: оператор %s подключился", c.Request().RemoteAddr)
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().Header().Set("Access-Control-Allow-Origin", config.CORS)
		flusher, ok := c.Response().Writer.(http.Flusher)
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, "Streaming unsupported")
		}
		for {
			select {
			case points := <-pointsChan:
				log.Printf("SSE: отправка облака точек оператору %s, количество точек: %d", c.Request().RemoteAddr, len(points))
				jsonData, err := json.Marshal(points)
				if err != nil {
					_, _ = c.Response().Write([]byte("data: []\n\n"))
				} else {
					_, _ = c.Response().Write([]byte("data: "))
					_, _ = c.Response().Write(jsonData)
					_, _ = c.Response().Write([]byte("\n\n"))
				}
				flusher.Flush()
			case <-c.Request().Context().Done():
				log.Printf("SSE: оператор %s отключился", c.Request().RemoteAddr)
				return nil
			}
		}
	})
}
