package main

import (
	deliveryQuic "dispatcher/internal/delivery/quic"
	deliveryUdp "dispatcher/internal/delivery/udp"
	"dispatcher/internal/usecase"
	gzipCompressor "dispatcher/internal/usecase/compressor/gzip"
	voxelCompressor "dispatcher/internal/usecase/compressor/voxel"
	"flag"
	"log"
	"time"
)

const (
	velodynePort = 2368
)

func main() {
	// Флаги
	listenPort := flag.Int("port", velodynePort, "Порт для UDP сервера")
	listenIP := flag.String("ip", "0.0.0.0", "IP для прослушивания UDP сервера")
	serverAddr := flag.String("server", "127.0.0.1:8081", "Адрес dispatcher-server (QUIC)")
	filterRadius := flag.Float64("filter-radius", 0.5, "Радиус фильтрации точек у центра (0 - отключить фильтр)")
	voxelSize := flag.Float64("voxel-size", 0.05, "Размер вокселя для компрессора")
	flag.Parse()

	log.Printf("Запуск клиента, QUIC сервер: %s, UDP: %s:%d", *serverAddr, *listenIP, *listenPort)

	udpChan := make(chan deliveryUdp.Packet, 1024)
	byteChan := make(chan []byte, 1024)
	processor := usecase.NewPointCloudProcessor(float32(*filterRadius))
	processor.SetCompressors(
		voxelCompressor.NewVoxelCompressor(float32(*voxelSize)),
		gzipCompressor.NewGzipCompressor(),
	)
	// Запуск UDP listener для получения точек от лидара
	err := deliveryUdp.StartUDPListener(*listenIP, *listenPort, udpChan)
	if err != nil {
		log.Fatalf("Ошибка запуска UDP listener: %v", err)
	}

	log.Printf("UDP слушатель запущен на %s:%d", *listenIP, *listenPort)

	// Мониторинг каналов
	go func() {
		for {
			log.Printf("Статус каналов - UDP: %d/1024, byteChan: %d/1024",
				len(udpChan), len(byteChan))
			time.Sleep(5 * time.Second)
		}
	}()

	go processor.Tx(udpChan, byteChan)

	log.Printf("Процессор запущен, ожидаем данные от лидара...")

	go func() {
		frameCount := 0
		for data := range byteChan {
			frameCount++
			log.Printf("Отправка кадра #%d по QUIC на %s (размер: %d байт)",
				frameCount, *serverAddr, len(data))
			err := deliveryQuic.SendPointCloud(*serverAddr, data)
			if err != nil {
				log.Printf("Ошибка отправки QUIC: %v", err)
			} else {
				log.Printf("Кадр #%d успешно отправлен по QUIC", frameCount)
			}
		}
	}()

	select {}
}
