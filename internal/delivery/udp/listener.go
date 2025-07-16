package udp

import (
	"log"
	"net"
	"time"
)

const (
	packetSize = 1206 // VLP-16 UDP размер пакета (без учёта заголовка UDP!)
)

type Packet struct {
	RawData []byte
}

func StartUDPListener(ip string, port int, out chan<- Packet) error {
	packetCount := 0

	go func() {
		for {
			addr := net.UDPAddr{
				IP:   net.ParseIP(ip),
				Port: port,
			}
			conn, err := net.ListenUDP("udp", &addr)
			if err != nil {
				log.Printf("UDP: Ошибка открытия сокета: %v\n", err)
				time.Sleep(1 * time.Second)
				continue
			}
			log.Printf("UDP: сервер запущен на %s:%d\n", ip, port)
			buf := make([]byte, packetSize)

			// Логируем каждые 100 пакетов
			lastLogTime := time.Now()
			localPacketCount := 0

			for {
				n, _, err := conn.ReadFromUDP(buf)
				if err != nil {
					log.Printf("UDP: Ошибка чтения: %v\n", err)
					_ = conn.Close()
					break // Перезапуск сервера
				}

				localPacketCount++
				packetCount++

				// Выводим статистику каждые 5 секунд
				if time.Since(lastLogTime) > 5*time.Second {
					log.Printf("UDP: получено %d пакетов за последние 5 секунд (всего: %d)",
						localPacketCount, packetCount)
					lastLogTime = time.Now()
					localPacketCount = 0
				}

				if n == packetSize {
					packet := make([]byte, packetSize)
					copy(packet, buf[:packetSize])
					select {
					case out <- Packet{RawData: packet}:
						// успешно отправлено
					default:
						log.Printf("UDP: [WARN] Канал UDP перегружен, пакет отброшен\n")
						// пакет отброшен
					}
				} else {
					log.Printf("UDP: получен пакет неверного размера: %d байт\n", n)
				}
			}
			time.Sleep(1 * time.Second) // Пауза перед повторным запуском
		}
	}()
	return nil
}
