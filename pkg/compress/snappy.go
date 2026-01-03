package compress

import (
	"io"

	"github.com/golang/snappy"
	"google.golang.org/grpc/encoding"
)

// snappyCompressor 实现 gRPC 的 Compressor 接口
type snappyCompressor struct{}

// SnappyCompressor 是 snappy 压缩器的单例实例
var SnappyCompressor = &snappyCompressor{}

func init() {
	// 注册 snappy 压缩器到 gRPC
	encoding.RegisterCompressor(SnappyCompressor)
}

// Compress 返回一个 snappy 压缩的 WriteCloser
func (s *snappyCompressor) Compress(w io.Writer) (io.WriteCloser, error) {
	return snappy.NewWriter(w), nil
}

// Decompress 返回一个 snappy 解压缩的 Reader
func (s *snappyCompressor) Decompress(r io.Reader) (io.Reader, error) {
	return snappy.NewReader(r), nil
}

// Name 返回压缩器的名称，用于在 gRPC 调用中标识
func (s *snappyCompressor) Name() string {
	return "snappy"
}

