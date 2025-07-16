package main

import (
	deliveryHttp "dispatcher/internal/delivery/http"
	deliveryUdp "dispatcher/internal/delivery/udp"
	"dispatcher/internal/usecase"
	gzipCompressor "dispatcher/internal/usecase/compressor/gzip"
	voxelCompressor "dispatcher/internal/usecase/compressor/voxel"
	"flag"
	"fmt"
	"github.com/labstack/echo/v4"
	"log"
)

const (
	velodynePort = 2368
)

var pointsChan chan [][]float32

func main() {
	// Флаги
	listenPort := flag.Int("port", velodynePort, "Порт для HTTP и UDP сервера")
	listenIP := flag.String("ip", "0.0.0.0", "IP для прослушивания UDP и HTTP сервера")
	ssePort := flag.Int("sse-port", 8080, "Порт SSE")
	sseIP := flag.String("sse-ip", "0.0.0.0", "IP SSE")
	cors := flag.String("cors", "*", "Значение Access-Control-Allow-Origin для CORS")
	filterRadius := flag.Float64("filter-radius", 0.5, "Радиус фильтрации точек у центра (0 - отключить фильтр)")
	voxelSize := flag.Float64("voxel-size", 0.05, "Размер вокселя для компрессора")
	flag.Parse()

	udpChan := make(chan deliveryUdp.Packet, 1024)
	pointsChan = make(chan [][]float32, 1024)
	// Запускаем UDP слушатель
	err := deliveryUdp.StartUDPListener(*listenIP, *listenPort, udpChan)
	if err != nil {
		log.Fatalf("Ошибка запуска UDP: %v\n", err)
	}
	byteChan := make(chan []byte, 1024)
	processor := usecase.NewPointCloudProcessor(float32(*filterRadius))

	// сначала voxel, потом gzip
	processor.SetCompressors(
		voxelCompressor.NewVoxelCompressor(float32(*voxelSize)),
		gzipCompressor.NewGzipCompressor(),
	)
	go processor.Tx(udpChan, byteChan)
	//-----------------------------------
	go processor.Rx(byteChan, pointsChan)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	deliveryHttp.RegisterSSEHandler(e, deliveryHttp.SSEConfig{CORS: *cors}, pointsChan)

	// Добавляем отдачу статических файлов Vite
	e.GET("/*", echo.WrapHandler(deliveryHttp.StaticHandler()))

	// Стартуем HTTP сервер
	fmt.Printf("HTTP сервер запущен на %s:%d\n", *sseIP, *ssePort)
	err = e.Start(fmt.Sprintf("%s:%d", *sseIP, *ssePort))
	if err != nil {
		log.Fatalf("Ошибка запуска HTTP сервера: %v\n", err)
	}
}
