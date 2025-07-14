package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

// IDGenerator ID生成器接口
type IDGenerator interface {
	Generate() string
}

// uuidGenerator UUID生成器
type uuidGenerator struct{}

// Generate 生成UUID
func (g *uuidGenerator) Generate() string {
	uuid := make([]byte, 16)
	rand.Read(uuid)
	
	// 设置版本 (4) 和变体位
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// NewUUIDGenerator 创建UUID生成器
func NewUUIDGenerator() IDGenerator {
	return &uuidGenerator{}
}

// sequentialIDGenerator 顺序ID生成器
type sequentialIDGenerator struct {
	counter uint64
	prefix  string
}

// Generate 生成顺序ID
func (g *sequentialIDGenerator) Generate() string {
	count := atomic.AddUint64(&g.counter, 1)
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s-%d-%d", g.prefix, timestamp, count)
}

// NewSequentialIDGenerator 创建顺序ID生成器
func NewSequentialIDGenerator(prefix string) IDGenerator {
	return &sequentialIDGenerator{
		prefix: prefix,
	}
}

// snowflakeIDGenerator 雪花算法ID生成器
type snowflakeIDGenerator struct {
	machineID      uint64
	datacenterID   uint64
	sequence       uint64
	lastTimestamp  int64
}

const (
	timestampBits  = 41
	datacenterBits = 5
	machineBits    = 5
	sequenceBits   = 12

	maxDatacenterID = -1 ^ (-1 << datacenterBits)
	maxMachineID    = -1 ^ (-1 << machineBits)
	maxSequence     = -1 ^ (-1 << sequenceBits)

	timestampShift  = sequenceBits + machineBits + datacenterBits
	datacenterShift = sequenceBits + machineBits
	machineShift    = sequenceBits

	epoch = int64(1609459200000) // 2021-01-01 00:00:00 UTC
)

// Generate 生成雪花ID
func (g *snowflakeIDGenerator) Generate() string {
	timestamp := time.Now().UnixMilli()
	
	if timestamp < g.lastTimestamp {
		panic("clock moved backwards")
	}
	
	if timestamp == g.lastTimestamp {
		g.sequence = (g.sequence + 1) & maxSequence
		if g.sequence == 0 {
			// 等待下一毫秒
			for timestamp <= g.lastTimestamp {
				timestamp = time.Now().UnixMilli()
			}
		}
	} else {
		g.sequence = 0
	}
	
	g.lastTimestamp = timestamp
	
	id := uint64((timestamp - epoch) << timestampShift) |
		(g.datacenterID << datacenterShift) |
		(g.machineID << machineShift) |
		g.sequence
		
	return fmt.Sprintf("%d", id)
}

// NewSnowflakeIDGenerator 创建雪花算法ID生成器
func NewSnowflakeIDGenerator(datacenterID, machineID uint64) IDGenerator {
	if datacenterID > maxDatacenterID {
		panic(fmt.Sprintf("datacenter ID must be between 0 and %d", maxDatacenterID))
	}
	if machineID > maxMachineID {
		panic(fmt.Sprintf("machine ID must be between 0 and %d", maxMachineID))
	}
	
	return &snowflakeIDGenerator{
		datacenterID: datacenterID,
		machineID:    machineID,
	}
}

// GenerateRequestID 生成请求ID（使用默认生成器）
var defaultIDGenerator = NewUUIDGenerator()

// GenerateRequestID 生成请求ID
func GenerateRequestID() string {
	return defaultIDGenerator.Generate()
}

// SetDefaultIDGenerator 设置默认ID生成器
func SetDefaultIDGenerator(generator IDGenerator) {
	defaultIDGenerator = generator
}

// GenerateTraceID 生成追踪ID（16字节hex）
func GenerateTraceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateSpanID 生成Span ID（8字节hex）
func GenerateSpanID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}