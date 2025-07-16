package compressor

import (
	"dispatcher/internal/usecase"
	"encoding/binary"
	"fmt"
	"github.com/chewxy/math32"
)

type VoxelCompressor struct {
	VoxelSize float32
}

func NewVoxelCompressor(voxelSize float32) usecase.PointCloudCompressor {
	return &VoxelCompressor{VoxelSize: voxelSize}
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

func (c *VoxelCompressor) Compress(data []byte) ([]byte, error) {
	pts, err := deserializePoints(data)
	if err != nil {
		return nil, err
	}
	voxelMap := make(map[[3]int][]float32)
	countMap := make(map[[3]int]int)
	for _, pt := range pts {
		vx := int(math32.Floor(pt[0] / c.VoxelSize))
		vy := int(math32.Floor(pt[1] / c.VoxelSize))
		vz := int(math32.Floor(pt[2] / c.VoxelSize))
		key := [3]int{vx, vy, vz}
		if _, ok := voxelMap[key]; !ok {
			voxelMap[key] = make([]float32, 3)
		}
		voxelMap[key][0] += pt[0]
		voxelMap[key][1] += pt[1]
		voxelMap[key][2] += pt[2]
		countMap[key]++
	}
	averaged := make([][]float32, 0, len(voxelMap))
	for key, sum := range voxelMap {
		cnt := float32(countMap[key])
		averaged = append(averaged, []float32{sum[0] / cnt, sum[1] / cnt, sum[2] / cnt})
	}
	return serializePoints(averaged)
}

func (c *VoxelCompressor) Decompress(data []byte) ([]byte, error) {
	return data, nil
}
