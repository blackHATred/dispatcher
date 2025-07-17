package main

import (
	"context"
	"crypto/tls"
	deliveryUdp "dispatcher/internal/delivery/udp"
	"dispatcher/internal/usecase"
	gzipCompressor "dispatcher/internal/usecase/compressor/gzip"
	voxelCompressor "dispatcher/internal/usecase/compressor/voxel"
	"flag"
	"fmt"
	"github.com/quic-go/quic-go"
	"log"
	"sync"
	"time"
)

// Пул буферов для отправки данных
var sendBufferPool = sync.Pool{
	New: func() any {
		// Создаём буфер с начальной ёмкостью 256 КБ (типичный размер пакета)
		return make([]byte, 0, 262144)
	},
}

const (
	velodynePort = 2368
)

func main() {
	// Флаги
	serverIP := flag.String("server-ip", "localhost", "IP удалённого сервера для QUIC соединения")
	serverPort := flag.Int("server-port", 8081, "Порт удалённого сервера для QUIC соединения")
	listenPort := flag.Int("port", velodynePort, "Порт для HTTP и UDP сервера")
	listenIP := flag.String("ip", "0.0.0.0", "IP для прослушивания UDP и HTTP сервера")
	filterRadius := flag.Float64("filter-radius", 0.5, "Радиус фильтрации точек у центра (0 - отключить фильтр)")
	voxelSize := flag.Float64("voxel-size", 0.05, "Размер вокселя для компрессора")
	flag.Parse()

	// UDP слушатель для принятия точек от Velodyne
	// --------------------------------------------
	udpChan := make(chan deliveryUdp.Packet, 1024)
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

	// Подключение по QUIC к удалённому серверу
	// ------------------------------------------------
	tlsConf := &tls.Config{
		InsecureSkipVerify: true, // лучше использовать сертификаты
	}
	quicConf := &quic.Config{
		EnableDatagrams: true,
		KeepAlivePeriod: 1 * time.Second,
		MaxIdleTimeout:  600 * time.Second,
	}
	ctx := context.Background()
	conn, err := quic.DialAddr(ctx, fmt.Sprintf("%s:%d", *serverIP, *serverPort), tlsConf, quicConf)
	if err != nil {
		log.Fatalf("Ошибка подключения к QUIC серверу: %v", err)
	}

	for {
		// Принимаем данные от Velodyne
		data, ok := <-byteChan
		if !ok {
			log.Println("Канал byteChan закрыт, завершаем отправку")
			return
		}

		// Создаем новый поток для каждого пакета данных
		stream, err := conn.OpenUniStreamSync(ctx)
		if err != nil {
			log.Printf("Ошибка открытия потока QUIC: %v", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Получаем буфер для отправки из пула и копируем данные
		sendBuf := sendBufferPool.Get().([]byte)[:0]
		sendBuf = append(sendBuf, data...)

		log.Printf("Отправляем %d байт по QUIC", len(sendBuf))

		// Отправляем данные одним куском
		_, err = stream.Write(sendBuf)
		if err != nil {
			log.Printf("Ошибка отправки данных по QUIC: %v", err)
			_ = stream.Close()
			sendBufferPool.Put(sendBuf) // Возвращаем буфер в пул в случае ошибки
			continue
		}

		// Закрываем поток после отправки данных, чтобы сигнализировать серверу о конце пакета
		err = stream.Close()
		if err != nil {
			log.Printf("Ошибка закрытия потока QUIC: %v", err)
		}

		// Возвращаем буфер в пул для переиспользования
		sendBufferPool.Put(sendBuf)

		log.Printf("Отправлено %d байт по QUIC", len(sendBuf))
	}
}
