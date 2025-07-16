package usecase

import (
	"dispatcher/internal/delivery/udp"
	"encoding/binary"
	"fmt"
	"github.com/chewxy/math32"
	"log"
)

type PointCloudProcessor struct {
	FilterRadius float32
	compressors  []PointCloudCompressor
}

func NewPointCloudProcessor(filterRadius float32) *PointCloudProcessor {
	return &PointCloudProcessor{FilterRadius: filterRadius}
}

func (p *PointCloudProcessor) SetCompressors(compressors ...PointCloudCompressor) {
	p.compressors = compressors
}

// Tx отвечает за сериализацию и прямой проход по pipeline компрессоров
func (p *PointCloudProcessor) Tx(in <-chan udp.Packet, out chan<- []byte) {
	frameBuf := make([][]float32, 0, 40000)
	prevAzimuth := float32(-1.0)
	frameCount := 0

	for packet := range in {
		buf := packet.RawData
		azimuth := float32(uint16(buf[2])|uint16(buf[3])<<8) / 100.0
		if prevAzimuth >= 0 && azimuth < prevAzimuth {
			if len(frameBuf) > 0 {
				frameCount++
				log.Printf("Processor: Обработка кадра #%d: %d точек", frameCount, len(frameBuf))
				data, err := serializePoints(frameBuf)
				if err != nil {
					log.Printf("Processor: Ошибка сериализации кадра #%d: %v", frameCount, err)
					continue
				}
				for i, compressor := range p.compressors {
					data, err = compressor.Compress(data)
					if err != nil {
						log.Printf("Processor: Ошибка компрессии #%d для кадра #%d: %v", i, frameCount, err)
						continue
					}
				}
				select {
				case out <- data:
					log.Printf("Processor: Кадр #%d отправлен в канал byteChan (размер: %d байт)", frameCount, len(data))
				default:
					log.Printf("Processor: Канал byteChan заполнен, кадр #%d пропущен", frameCount)
				}
			}
			frameBuf = make([][]float32, 0, 40000)
		}
		prevAzimuth = azimuth
		for block := 0; block < 12; block++ {
			start := block * 100
			azimuthBlock := float32(uint16(buf[start+2])|uint16(buf[start+3])<<8) / 100.0
			for laser := 0; laser < 32; laser++ {
				offset := start + 4 + laser*3
				dist := float32(uint16(buf[offset])|uint16(buf[offset+1])<<8) * 0.002
				vertAngle := vlp16VerticalAngle(laser)
				azimuthRad := azimuthBlock * math32.Pi / 180.0
				vertRad := vertAngle * math32.Pi / 180.0
				x := dist * math32.Cos(vertRad) * math32.Sin(azimuthRad)
				y := dist * math32.Cos(vertRad) * math32.Cos(azimuthRad)
				z := dist * math32.Sin(vertRad)
				if p.FilterRadius == 0 ||
					math32.Abs(x) > p.FilterRadius ||
					math32.Abs(y) > p.FilterRadius ||
					math32.Abs(z) > p.FilterRadius {
					frameBuf = append(frameBuf, []float32{x, y, z})
				}
			}
		}
	}
}

// Rx отвечает за обратный проход по pipeline компрессоров и десериализацию
func (p *PointCloudProcessor) Rx(in <-chan []byte, out chan<- [][]float32) {
	frameCount := 0
	for data := range in {
		frameCount++
		log.Printf("Rx: получен кадр #%d (размер: %d байт)", frameCount, len(data))

		var err error
		for i := len(p.compressors) - 1; i >= 0; i-- {
			log.Printf("Rx: декомпрессия #%d для кадра #%d, тип компрессора: %T", i, frameCount, p.compressors[i])
			before := len(data)
			data, err = p.compressors[i].Decompress(data)
			if err != nil {
				log.Printf("Rx: ошибка декомпрессии #%d для кадра #%d: %v", i, frameCount, err)
				break
			}
			log.Printf("Rx: декомпрессия #%d успешна, размер до: %d, после: %d байт", i, before, len(data))
		}

		if err != nil {
			log.Printf("Rx: пропуск кадра #%d из-за ошибки декомпрессии", frameCount)
			continue
		}

		pts, err := deserializePoints(data)
		if err != nil {
			log.Printf("Rx: ошибка десериализации точек для кадра #%d: %v", frameCount, err)
			continue
		}

		log.Printf("Rx: десериализовано %d точек для кадра #%d", len(pts), frameCount)

		select {
		case out <- pts:
			log.Printf("Rx: кадр #%d отправлен в канал pointsChan", frameCount)
		default:
			log.Printf("Rx: канал pointsChan заполнен, кадр #%d пропущен", frameCount)
		}
	}
}

func serializePoints(points [][]float32) ([]byte, error) {
	buf := make([]byte, 0, len(points)*12)
	for _, pt := range points {
		for i := 0; i < 3; i++ {
			b := make([]byte, 4)
			binary.LittleEndian.PutUint32(b, math32.Float32bits(pt[i]))
			buf = append(buf, b...)
		}
	}
	return buf, nil
}

func deserializePoints(data []byte) ([][]float32, error) {
	if len(data)%12 != 0 {
		return nil, fmt.Errorf("количество байт должно быть кратно 12")
	}
	pts := make([][]float32, 0, len(data)/12)
	for i := 0; i+12 <= len(data); i += 12 {
		x := math32.Float32frombits(binary.LittleEndian.Uint32(data[i : i+4]))
		y := math32.Float32frombits(binary.LittleEndian.Uint32(data[i+4 : i+8]))
		z := math32.Float32frombits(binary.LittleEndian.Uint32(data[i+8 : i+12]))
		pts = append(pts, []float32{x, y, z})
	}
	return pts, nil
}

func vlp16VerticalAngle(laser int) float32 {
	angles := []float32{
		-15, 1, -13, 3, -11, 5, -9, 7,
		-7, 9, -5, 11, -3, 13, -1, 15,
	}
	return angles[laser%16]
}
