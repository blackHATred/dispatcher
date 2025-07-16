package main

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"log"
	"net"
	"time"
)

const (
	pcapFile = "static/sample1.pcap"
	// velodyneIP   = "127.0.0.1"
	velodyneIP   = "192.168.1.2"
	velodynePort = 2368
	packetSize   = 1206
)

func main() {
	for {
		handle, err := pcap.OpenOffline(pcapFile)
		if err != nil {
			log.Fatalf("Ошибка открытия pcap: %v", err)
		}
		defer handle.Close()

		conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
			IP:   net.ParseIP(velodyneIP),
			Port: velodynePort,
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
