package quic

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"github.com/quic-go/quic-go"
	"io"
	"log"
	"math/big"
	"time"
)

func generateTLSConfig() *tls.Config {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certDER, _ := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &key.PublicKey, key)
	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}
}

func StartQUICServer(addr string, handler func([]byte), onClientConnect func(string)) error {
	tlsConfig := generateTLSConfig()
	log.Printf("QUIC: запуск сервера на адресе %s", addr)
	listener, err := quic.ListenAddr(addr, tlsConfig, nil)
	if err != nil {
		log.Printf("QUIC: ошибка запуска сервера на %s: %v", addr, err)
		return err
	}
	log.Printf("QUIC: сервер успешно запущен на %s", addr)

	for {
		acceptCtx, cancelAccept := context.WithTimeout(context.Background(), 10*time.Second)
		session, err := listener.Accept(acceptCtx)
		cancelAccept()

		if err != nil {
			if err.Error() == "context deadline exceeded" {
				// Тайм-аут истёк, но это нормально для Accept
				continue
			}
			log.Printf("QUIC: ошибка при Accept: %v", err)
			continue
		}

		clientAddr := session.RemoteAddr().String()
		log.Printf("QUIC: клиент %s подклю��ился", clientAddr)
		onClientConnect(clientAddr)

		go func() {
			streamCtx, cancelStream := context.WithTimeout(context.Background(), 5*time.Second)
			stream, err := session.AcceptStream(streamCtx)
			cancelStream()

			if err != nil {
				log.Printf("QUIC: ошибка AcceptStream от клиента %s: %v", clientAddr, err)
				return
			}

			// Читаем размер данных (4 байта)
			sizeBuffer := make([]byte, 4)
			_, err = io.ReadFull(stream, sizeBuffer)
			if err != nil {
				log.Printf("QUIC: ошибка чтения размера данных от клиента %s: %v", clientAddr, err)
				return
			}

			dataSize := binary.BigEndian.Uint32(sizeBuffer)
			log.Printf("QUIC: ожидаемый размер данных от клиента %s: %d байт", clientAddr, dataSize)

			// Читаем все данные в буфер
			buffer := make([]byte, dataSize)
			bytesRead, err := io.ReadFull(stream, buffer)
			if err != nil {
				log.Printf("QUIC: ошибка чтения данных от клиента %s: %v (прочитано %d из %d байт)",
					clientAddr, err, bytesRead, dataSize)
				return
			}

			log.Printf("QUIC: получено %d байт от клиента %s", bytesRead, clientAddr)
			handler(buffer)
		}()
	}
}
