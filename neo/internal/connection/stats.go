package connection

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ConnectionStats 连接统计信息
type ConnectionStats struct {
	ActiveRequests int64         // 当前活跃请求数
	TotalRequests  int64         // 总请求数
	ErrorCount     int64         // 错误数
	LatencyStats   *LatencyStats // 延迟统计
	LastUsedTime   time.Time     // 最后使用时间
}

// NewConnectionStats 创建连接统计
func NewConnectionStats() *ConnectionStats {
	return &ConnectionStats{
		LatencyStats: NewLatencyStats(),
		LastUsedTime: time.Now(),
	}
}

// LatencyStats 延迟统计
type LatencyStats struct {
	mu          sync.RWMutex
	count       int64
	sum         time.Duration
	min         time.Duration
	max         time.Duration
	buckets     []int64    // 延迟分布桶
	boundaries  []float64  // 桶边界（毫秒）
	windowSize  int        // 滑动窗口大小
	samples     []float64  // 最近的样本
	currentPos  int       // 当前样本位置
}

// NewLatencyStats 创建延迟统计
func NewLatencyStats() *LatencyStats {
	return &LatencyStats{
		min:        time.Duration(1<<63 - 1),
		boundaries: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000}, // 毫秒
		buckets:    make([]int64, 10),                               // 9个边界产生10个桶
		windowSize: 1000,                                            // 保存最近1000个样本
		samples:    make([]float64, 1000),
	}
}

// Add 添加一个延迟样本
func (s *LatencyStats) Add(latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 更新基本统计信息
	s.count++
	s.sum += latency
	if latency < s.min {
		s.min = latency
	}
	if latency > s.max {
		s.max = latency
	}

	// 更新延迟分布
	ms := float64(latency.Milliseconds())
	bucketIndex := s.findBucket(ms)
	atomic.AddInt64(&s.buckets[bucketIndex], 1)

	// 更新滑动窗口
	s.samples[s.currentPos] = ms
	s.currentPos = (s.currentPos + 1) % s.windowSize
}

// findBucket 找到延迟值所属的桶
func (s *LatencyStats) findBucket(ms float64) int {
	for i, boundary := range s.boundaries {
		if ms <= boundary {
			return i
		}
	}
	return len(s.boundaries)
}

// Average 获取平均延迟
func (s *LatencyStats) Average() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.count == 0 {
		return 0
	}
	return time.Duration(float64(s.sum) / float64(s.count))
}

// Percentile 获取指定百分位的延迟
func (s *LatencyStats) Percentile(p float64) time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.count == 0 {
		return 0
	}

	// 复制并排序样本
	samples := make([]float64, s.count)
	copy(samples, s.samples[:s.count])
	quickSort(samples)

	// 计算百分位
	index := int(float64(s.count-1) * p)
	return time.Duration(samples[index] * float64(time.Millisecond))
}

// GetDistribution 获取延迟分布
func (s *LatencyStats) GetDistribution() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dist := make(map[string]int64)
	dist["0-1ms"] = s.buckets[0]
	
	for i := 1; i < len(s.boundaries); i++ {
		key := fmt.Sprintf("%.0f-%.0fms", s.boundaries[i-1], s.boundaries[i])
		dist[key] = s.buckets[i]
	}
	
	key := fmt.Sprintf(">%.0fms", s.boundaries[len(s.boundaries)-1])
	dist[key] = s.buckets[len(s.buckets)-1]

	return dist
}

// Reset 重置统计信息
func (s *LatencyStats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.count = 0
	s.sum = 0
	s.min = time.Duration(1<<63 - 1)
	s.max = 0
	for i := range s.buckets {
		s.buckets[i] = 0
	}
	s.currentPos = 0
}

// quickSort 快速排序实现
func quickSort(arr []float64) {
	if len(arr) <= 1 {
		return
	}

	pivot := arr[len(arr)/2]
	left, right := 0, len(arr)-1

	for left <= right {
		for arr[left] < pivot {
			left++
		}
		for arr[right] > pivot {
			right--
		}
		if left <= right {
			arr[left], arr[right] = arr[right], arr[left]
			left++
			right--
		}
	}

	if right > 0 {
		quickSort(arr[:right+1])
	}
	if left < len(arr) {
		quickSort(arr[left:])
	}
}