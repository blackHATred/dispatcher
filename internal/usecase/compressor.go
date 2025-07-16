package usecase

// PointCloudCompressor описывает методы для сжатия/разжатия облака точек
// Compress принимает []byte (например, сериализованные точки), возвращает []byte (сжатые данные)
// Decompress принимает []byte (сжатые данные), возвращает []byte (десериализованные точки)
type PointCloudCompressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
}
