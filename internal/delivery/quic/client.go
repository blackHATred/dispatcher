package quic

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"github.com/quic-go/quic-go"
	"log"
	"time"
)

func getTLSConfig() *tls.Config {
	return &tls.Config{InsecureSkipVerify: true}
}

func SendPointCloud(addr string, data []byte) error {
	log.Printf("QUIC: попытка подключения к серверу %s", addr)
	// Добавляем таймаут для подключения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := quic.DialAddr(ctx, addr, getTLSConfig(), nil)
	if err != nil {
		log.Printf("QUIC: ошибка подключения к серверу %s: %v", addr, err)
		return err
	}
	log.Printf("QUIC: подключение к серверу %s успешно установлено", addr)

	// Создаём новый контекст с таймаутом для открытия потока
	streamCtx, streamCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer streamCancel()

	stream, err := session.OpenStreamSync(streamCtx)
	if err != nil {
		log.Printf("QUIC: ошибка открытия stream: %v", err)
		return err
	}
	defer stream.Close()

	// Сначала отправляем размер данных (4 байта)
	sizeBuffer := make([]byte, 4)
	binary.BigEndian.PutUint32(sizeBuffer, uint32(len(data)))
	_, err = stream.Write(sizeBuffer)
	if err != nil {
		log.Printf("QUIC: ошибка отправки размера данных: %v", err)
		return err
	}

	// Отправляем данные небольшими блоками
	const chunkSize = 16 * 1024 // 16KB блоки
	totalWritten := 0

	for totalWritten < len(data) {
		endPos := totalWritten + chunkSize
		if endPos > len(data) {
			endPos = len(data)
		}

		chunk := data[totalWritten:endPos]
		written, err := stream.Write(chunk)
		if err != nil {
			log.Printf("QUIC: ошибка отправки блока данных: %v", err)
			return err
		}

		totalWritten += written
		if written < len(chunk) {
			// Если записано меньше, чем размер блока - дописываем остаток
			remaining := chunk[written:]
			for len(remaining) > 0 {
				n, err := stream.Write(remaining)
				if err != nil {
					log.Printf("QUIC: ошибка при дозаписи данных: %v", err)
					return err
				}
				remaining = remaining[n:]
				totalWritten += n
			}
		}
	}

	log.Printf("QUIC: успешно отправлено %d/%d байт на сервер %s", totalWritten, len(data), addr)
	return nil
}
