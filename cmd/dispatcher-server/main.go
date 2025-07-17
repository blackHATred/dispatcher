package main

import (
	"context"
	"crypto/tls"
	deliveryHttp "dispatcher/internal/delivery/http"
	"dispatcher/internal/usecase"
	gzipCompressor "dispatcher/internal/usecase/compressor/gzip"
	"errors"
	"flag"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/quic-go/quic-go"
	"io"
	"log"
	"sync"
)

// Создаём пулы буферов для переиспользования
var bufferPool = sync.Pool{
	New: func() any {
		// Создаём буфер размером 64 КБ
		return make([]byte, 65536)
	},
}

// Пул для накопления данных
var dataBufferPool = sync.Pool{
	New: func() any {
		// Начальный размер буфера для собираемых данных - 256 КБ
		return make([]byte, 0, 262144)
	},
}

func main() {
	listenIP := flag.String("ip", "0.0.0.0", "IP для прослушивания QUIC")
	listenPort := flag.Int("port", 8081, "Порт для прослушивания QUIC")
	ssePort := flag.Int("sse-port", 8080, "Порт SSE")
	sseIP := flag.String("sse-ip", "0.0.0.0", "IP SSE")
	cors := flag.String("cors", "*", "Значение Access-Control-Allow-Origin для CORS")
	filterRadius := flag.Float64("filter-radius", 0.05, "Радиус фильтрации точек у центра (0 - отключить фильтр)")
	flag.Parse()

	// SSE сервер для операторов
	// -------------------------
	pointsChan := make(chan [][]float32, 1024)
	byteChan := make(chan []byte, 1024)
	processor := usecase.NewPointCloudProcessor(float32(*filterRadius))
	processor.SetCompressors(
		gzipCompressor.NewGzipCompressor(),
	)
	// обработка приходящих данных
	go processor.Rx(byteChan, pointsChan)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	deliveryHttp.RegisterSSEHandler(e, deliveryHttp.SSEConfig{CORS: *cors}, pointsChan)

	// Добавляем отдачу статических файлов Vite
	e.GET("/*", echo.WrapHandler(deliveryHttp.StaticHandler()))

	// Стартуем HTTP сервер
	log.Printf("HTTP сервер запущен на %s:%d\n", *sseIP, *ssePort)
	go func() {
		err := e.Start(fmt.Sprintf("%s:%d", *sseIP, *ssePort))
		if err != nil {
			log.Fatalf("Ошибка запуска HTTP сервера: %v\n", err)
		}
	}()

	// QUIC для принятия данных от ТС
	// ------------------------------
	certFile := "config/localhost.pem"
	keyFile := "config/localhost-key.pem"
	tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatal("Ошибка загрузки сертификатов:", err)
	}
	tlsConf := &tls.Config{
		Certificates:       []tls.Certificate{tlsCert},
		InsecureSkipVerify: true, // лучше использовать сертификаты
	}
	quicConf := &quic.Config{
		EnableDatagrams: true,
	}
	ln, err := quic.ListenAddr(fmt.Sprintf("%s:%d", *listenIP, *listenPort), tlsConf, quicConf)
	if err != nil {
		log.Fatalf("Ошибка запуска QUIC сервера: %v", err)
	}
	log.Printf("QUIC сервер запущен на %s:%d", *listenIP, *listenPort)
	for {
		conn, err := ln.Accept(context.Background())
		if err != nil {
			log.Printf("Ошибка при принятии соединения: %v", err)
			continue
		}
		// Запускаем обработку соединения в отдельной горутине
		go handleConnection(conn, byteChan)
	}
}

func handleConnection(conn *quic.Conn, byteChan chan<- []byte) {
	log.Printf("Новое соединение: %s", conn.RemoteAddr().String())

	// Обрабатываем все потоки от соединения
	for {
		stream, err := conn.AcceptUniStream(context.Background())
		if err != nil {
			log.Printf("Ошибка при принятии потока: %v", err)
			return
		}

		log.Printf("Принят поток: %d", stream.StreamID())

		// Обрабатываем каждый поток в отдельной горутине
		go func(s *quic.ReceiveStream) {
			// Получаем буферы из пулов
			buf := bufferPool.Get().([]byte)
			dataBuf := dataBufferPool.Get().([]byte)
			dataBuf = dataBuf[:0] // Очищаем буфер, но сохраняем ёмкость

			var totalBytes int

			defer func() {
				// Возвращаем временный буфер в пул
				bufferPool.Put(buf)
				// dataBuf не возвращаем в пул сразу, т.к. он отправляется в канал
				// и будет использоваться в другой части программы
			}()

			for {
				n, err := s.Read(buf)
				if err != nil && !errors.Is(err, io.EOF) {
					log.Printf("Ошибка чтения из потока %d: %v", s.StreamID(), err)
					dataBufferPool.Put(dataBuf) // Возвращаем буфер в пул при ошибке
					return
				}

				if n > 0 {
					dataBuf = append(dataBuf, buf[:n]...)
					totalBytes += n
					log.Printf("Прочитано %d байт из потока %d (всего: %d)", n, s.StreamID(), totalBytes)
				}

				// Если достигли конца потока или ошибка - выходим из цикла
				if errors.Is(err, io.EOF) || n == 0 {
					break
				}
			}

			// Если собрали данные, отправляем в канал обработки
			if totalBytes > 0 {
				log.Printf("Собран полный пакет: %d байт из потока %d", totalBytes, s.StreamID())

				// Создаем копию данных для отправки в канал
				result := make([]byte, len(dataBuf))
				copy(result, dataBuf)

				// Теперь можем вернуть dataBuf в пул
				dataBufferPool.Put(dataBuf)

				// Отправляем копию данных в канал
				byteChan <- result
			} else {
				// Возвращаем пустой буфер в пул
				dataBufferPool.Put(dataBuf)
			}
		}(stream)
	}
}
