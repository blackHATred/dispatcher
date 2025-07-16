package main

import (
	"dispatcher/internal/delivery/http"
	quicDelivery "dispatcher/internal/delivery/quic"
	"dispatcher/internal/usecase"
	gzipCompressor "dispatcher/internal/usecase/compressor/gzip"
	voxelCompressor "dispatcher/internal/usecase/compressor/voxel"
	"flag"
	"fmt"
	"github.com/labstack/echo/v4"
	"log"
)

var (
	pointsChan chan [][]float32
	byteChan   chan []byte
)

func main() {
	listenPort := flag.Int("port", 8081, "Порт для QUIC сервера")
	listenIP := flag.String("ip", "0.0.0.0", "IP для QUIC сервера")
	ssePort := flag.Int("sse-port", 8080, "Порт SSE")
	sseIP := flag.String("sse-ip", "0.0.0.0", "IP SSE")
	cors := flag.String("cors", "*", "CORS")
	flag.Parse()

	pointsChan = make(chan [][]float32, 1024)
	byteChan = make(chan []byte, 1024)

	processor := usecase.NewPointCloudProcessor(0)
	processor.SetCompressors(
		// Обратный порядок компрессоров по сравнению с клиентом:
		// Сначала gzip (так как он применяется последним при компрессии),
		// затем voxel (так как он применяется первым при компрессии)
		gzipCompressor.NewGzipCompressor(),
		voxelCompressor.NewVoxelCompressor(0.01),
	)
	go processor.Rx(byteChan, pointsChan)

	go func() {
		addr := *listenIP + ":" + fmt.Sprint(*listenPort)
		err := quicDelivery.StartQUICServer(addr, func(data []byte) {
			byteChan <- data
		}, func(clientAddr string) {
			log.Printf("QUIC: клиент %s подключился", clientAddr)
		})
		if err != nil {
			log.Fatalf("Ошибка QUIC сервера: %v", err)
		}
	}()

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	http.RegisterSSEHandler(e, http.SSEConfig{CORS: *cors}, pointsChan)
	e.GET("/*", echo.WrapHandler(http.StaticHandler()))
	fmt.Printf("SSE сервер запущен на %s:%d\n", *sseIP, *ssePort)
	err := e.Start(fmt.Sprintf("%s:%d", *sseIP, *ssePort))
	if err != nil {
		log.Fatalf("Ошибка запуска SSE сервера: %v\n", err)
	}
}
