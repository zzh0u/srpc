package tools

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// IDGenerator 是一个简单的ID生成器接口
type IDGenerator interface {
	Generate() string
}

// SimpleIDGenerator 基于时间戳和序列号的简单ID生成器
type SimpleIDGenerator struct {
	mu       sync.Mutex
	lastTime int64
	sequence uint16
	nodeID   uint16
}

// NewSimpleIDGenerator 创建新的简单ID生成器
// nodeID 用于区分不同节点（0-1023）
func NewSimpleIDGenerator(nodeID uint16) *SimpleIDGenerator {
	if nodeID > 1023 {
		nodeID = nodeID % 1024
	}
	return &SimpleIDGenerator{
		nodeID: nodeID,
	}
}

// Generate 生成一个基于时间戳的唯一ID
// 格式：时间戳(41位) + 节点ID(10位) + 序列号(13位)
// 总共64位，适合int64存储
func (g *SimpleIDGenerator) Generate() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now().UnixMilli()

	if now == g.lastTime {
		g.sequence++
		if g.sequence >= 8192 { // 13位序列号最大8191
			// 等待下一毫秒
			for now <= g.lastTime {
				time.Sleep(time.Microsecond)
				now = time.Now().UnixMilli()
			}
			g.sequence = 0
		}
	} else {
		g.sequence = 0
	}

	g.lastTime = now

	// 组合ID：时间戳(41位) | 节点ID(10位) | 序列号(13位)
	id := (int64(now) << 23) | (int64(g.nodeID) << 13) | int64(g.sequence)
	return fmt.Sprintf("%d", id)
}

// GenerateUUID 生成一个随机的UUID（版本4）
func GenerateUUID() (string, error) {
	var uuid [16]byte
	_, err := rand.Read(uuid[:])
	if err != nil {
		return "", err
	}

	// 设置版本为4（随机）
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	// 设置变体为RFC 4122
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:16]), nil
}

// MustGenerateUUID 生成UUID，如果失败则panic
func MustGenerateUUID() string {
	uuid, err := GenerateUUID()
	if err != nil {
		panic(err)
	}
	return uuid
}

// ShortID 生成一个简短的随机ID（16字符）
func ShortID() (string, error) {
	var bytes [8]byte
	_, err := rand.Read(bytes[:])
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}

// DefaultIDGenerator 默认的ID生成器实例
var (
	defaultGenerator *SimpleIDGenerator
	once             sync.Once
)

// GetDefaultIDGenerator 获取默认的ID生成器（单例）
func GetDefaultIDGenerator() *SimpleIDGenerator {
	once.Do(func() {
		// 使用一个随机的节点ID
		var nodeID uint16
		// 读取2个随机字节并转换为uint16
		var b [2]byte
		rand.Read(b[:])
		// 使用小端字节序转换
		nodeID = uint16(b[0]) | uint16(b[1])<<8
		defaultGenerator = NewSimpleIDGenerator(nodeID % 1024)
	})
	return defaultGenerator
}

// GenerateID 使用默认生成器生成ID
func GenerateID() string {
	return GetDefaultIDGenerator().Generate()
}
