package main

import (
	"flag"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"log"
	"net"
	"time"
)

// Дефолтные значения конфигурации
const (
	defaultPcapFile     = "static/sample1.pcap"
	defaultVelodyneIP   = "192.168.1.2"
	defaultVelodynePort = 2368
	packetSize          = 1206
)

func main() {
	// Парсим флаги командной строки
	pcapFile := flag.String("pcap", defaultPcapFile, "Путь к PCAP-файлу")
	velodyneIP := flag.String("ip", defaultVelodyneIP, "IP-адрес для отправки данных")
	velodynePort := flag.Int("port", defaultVelodynePort, "Порт для отправки данных")
	flag.Parse()

	log.Printf("Используем конфигурацию: pcap=%s, ip=%s, port=%d", *pcapFile, *velodyneIP, *velodynePort)

	for {
		handle, err := pcap.OpenOffline(*pcapFile)
		if err != nil {
			log.Fatalf("Ошибка открытия pcap: %v", err)
		}
		defer handle.Close()

		conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
			IP:   net.ParseIP(*velodyneIP),
			Port: *velodynePort,
		})
		if err != nil {
			log.Fatalf("Ошибка открытия UDP: %v", err)
		}
		defer conn.Close()

		packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		count := 0
		for packet := range packetSource.Packets() {
			udpLayer := packet.Layer(layers.LayerTypeUDP)
			if udpLayer == nil {
				continue
			}
			udp, _ := udpLayer.(*layers.UDP)
			if len(udp.Payload) == packetSize {
				_, err := conn.Write(udp.Payload)
				if err != nil {
					log.Printf("Ошибка отправки пакета: %v", err)
				}
				count++
				fmt.Printf("Отправлен пакет #%d\n", count)
				time.Sleep(1330 * time.Microsecond) // 1.33 мс задержка
			}
		}
		fmt.Println("Достигнут конец pcap, начинаем заново...")
		// handle.Close() и conn.Close() вызовутся через defer
		// Новый цикл начнёт воспроизведение сначала
	}
}
