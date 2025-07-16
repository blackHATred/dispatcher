package udp

import (
	"fmt"
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
	go func() {
		for {
			addr := net.UDPAddr{
				IP:   net.ParseIP(ip),
				Port: port,
			}
			conn, err := net.ListenUDP("udp", &addr)
			if err != nil {
				fmt.Printf("Ошибка открытия сокета: %v\n", err)
				time.Sleep(1 * time.Second)
				continue
			}
			fmt.Printf("UDP сервер запущен на %s:%d\n", ip, port)
			buf := make([]byte, packetSize)
			for {
				n, _, err := conn.ReadFromUDP(buf)
				if err != nil {
					fmt.Printf("Ошибка чтения: %v\n", err)
					_ = conn.Close()
					break // Перезапуск сервера
				}
				if n == packetSize {
					packet := make([]byte, packetSize)
					copy(packet, buf[:packetSize])
					select {
					case out <- Packet{RawData: packet}:
						// успешно отправлено
					default:
						fmt.Printf("[WARN] Канал UDP перегружен, пакет отброшен\n")
						// пакет отброшен
					}
				}
			}
			time.Sleep(1 * time.Second) // Пауза перед повторным запуском
		}
	}()
	return nil
}
