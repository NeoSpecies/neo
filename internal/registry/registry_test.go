package registry_test

import (
	"context"
	"fmt"
	"neo/internal/registry"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试服务实例
func TestServiceInstance(t *testing.T) {
	t.Run("GetFullAddress", func(t *testing.T) {
		// 有端口
		instance := &registry.ServiceInstance{
			Address: "127.0.0.1",
			Port:    8080,
		}
		assert.Equal(t, "127.0.0.1:8080", instance.GetFullAddress())

		// 无端口
		instance.Port = 0
		assert.Equal(t, "127.0.0.1", instance.GetFullAddress())
	})

	t.Run("IsHealthy", func(t *testing.T) {
		instance := &registry.ServiceInstance{
			Status: registry.StatusHealthy,
		}
		assert.True(t, instance.IsHealthy())

		instance.Status = registry.StatusUnhealthy
		assert.False(t, instance.IsHealthy())
	})

	t.Run("Clone", func(t *testing.T) {
		instance := &registry.ServiceInstance{
			ID:      "test-1",
			Name:    "test-service",
			Address: "127.0.0.1",
			Port:    8080,
			Metadata: map[string]string{
				"version": "1.0",
				"region":  "us-east",
			},
			Status: registry.StatusHealthy,
			Weight: 10,
		}

		cloned := instance.Clone()
		assert.Equal(t, instance.ID, cloned.ID)
		assert.Equal(t, instance.Name, cloned.Name)
		assert.Equal(t, instance.Metadata, cloned.Metadata)

		// 修改克隆的元数据不影响原实例
		cloned.Metadata["new"] = "value"
		assert.NotContains(t, instance.Metadata, "new")
	})
}

// 测试服务注册中心
func TestServiceRegistry(t *testing.T) {
	ctx := context.Background()

	t.Run("基本注册和发现", func(t *testing.T) {
		reg := registry.NewServiceRegistry()

		// 注册服务
		instance := &registry.ServiceInstance{
			ID:      "test-1",
			Name:    "test-service",
			Address: "127.0.0.1",
			Port:    8080,
			Metadata: map[string]string{
				"version": "1.0",
			},
		}

		err := reg.Register(ctx, instance)
		require.NoError(t, err)

		// 发现服务
		instances, err := reg.Discover(ctx, "test-service")
		require.NoError(t, err)
		assert.Len(t, instances, 1)
		assert.Equal(t, "test-1", instances[0].ID)
		assert.Equal(t, registry.StatusHealthy, instances[0].Status)
	})

	t.Run("注销服务", func(t *testing.T) {
		reg := registry.NewServiceRegistry()

		// 注册服务
		instance := &registry.ServiceInstance{
			ID:      "test-1",
			Name:    "test-service",
			Address: "127.0.0.1",
			Port:    8080,
		}

		err := reg.Register(ctx, instance)
		require.NoError(t, err)

		// 注销服务
		err = reg.Deregister(ctx, "test-1")
		require.NoError(t, err)

		// 再次发现应该为空
		instances, err := reg.Discover(ctx, "test-service")
		require.NoError(t, err)
		assert.Len(t, instances, 0)
	})

	t.Run("参数验证", func(t *testing.T) {
		reg := registry.NewServiceRegistry()

		// nil实例
		err := reg.Register(ctx, nil)
		assert.Error(t, err)

		// 空ID
		err = reg.Register(ctx, &registry.ServiceInstance{
			Name:    "test",
			Address: "127.0.0.1",
		})
		assert.Error(t, err)

		// 空名称
		err = reg.Register(ctx, &registry.ServiceInstance{
			ID:      "test-1",
			Address: "127.0.0.1",
		})
		assert.Error(t, err)

		// 空地址
		err = reg.Register(ctx, &registry.ServiceInstance{
			ID:   "test-1",
			Name: "test",
		})
		assert.Error(t, err)
	})

	t.Run("更新实例", func(t *testing.T) {
		reg := registry.NewServiceRegistry()

		// 注册初始实例
		instance := &registry.ServiceInstance{
			ID:      "test-1",
			Name:    "test-service",
			Address: "127.0.0.1",
			Port:    8080,
			Weight:  10,
		}

		err := reg.Register(ctx, instance)
		require.NoError(t, err)

		// 更新实例
		instance.Weight = 20
		instance.Metadata = map[string]string{"version": "2.0"}
		err = reg.UpdateInstance(ctx, instance)
		require.NoError(t, err)

		// 验证更新
		updated, err := reg.GetInstance(ctx, "test-1")
		require.NoError(t, err)
		assert.Equal(t, 20, updated.Weight)
		assert.Equal(t, "2.0", updated.Metadata["version"])
	})

	t.Run("列出所有服务", func(t *testing.T) {
		reg := registry.NewServiceRegistry()

		// 注册多个服务
		services := []string{"service-a", "service-b", "service-c"}
		for i, name := range services {
			err := reg.Register(ctx, &registry.ServiceInstance{
				ID:      fmt.Sprintf("instance-%d", i),
				Name:    name,
				Address: "127.0.0.1",
				Port:    8080 + i,
			})
			require.NoError(t, err)
		}

		// 列出所有服务
		list, err := reg.ListServices(ctx)
		require.NoError(t, err)
		assert.Len(t, list, 3)
		
		// 验证所有服务都在列表中
		for _, svc := range services {
			assert.Contains(t, list, svc)
		}
	})
}

// 测试Watch机制
func TestServiceWatch(t *testing.T) {
	ctx := context.Background()

	t.Run("基本Watch功能", func(t *testing.T) {
		reg := registry.NewServiceRegistry()

		// 开始监听
		watchCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		eventCh, err := reg.Watch(watchCtx, "test-service")
		require.NoError(t, err)

		// 收集事件
		events := make([]registry.ServiceEvent, 0)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for event := range eventCh {
				events = append(events, event)
			}
		}()

		// 注册服务
		instance := &registry.ServiceInstance{
			ID:      "test-1",
			Name:    "test-service",
			Address: "127.0.0.1",
			Port:    8080,
		}
		err = reg.Register(ctx, instance)
		require.NoError(t, err)

		// 注销服务
		err = reg.Deregister(ctx, "test-1")
		require.NoError(t, err)

		// 等待事件处理
		time.Sleep(100 * time.Millisecond)
		cancel()
		wg.Wait()

		// 验证事件
		require.Len(t, events, 2)
		assert.Equal(t, registry.EventRegister, events[0].Type)
		assert.Equal(t, registry.EventDeregister, events[1].Type)
	})

	t.Run("多个Watcher", func(t *testing.T) {
		reg := registry.NewServiceRegistry()

		// 创建多个watcher
		watchCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		ch1, err := reg.Watch(watchCtx, "test-service")
		require.NoError(t, err)

		ch2, err := reg.Watch(watchCtx, "test-service")
		require.NoError(t, err)

		// 收集事件
		var events1, events2 []registry.ServiceEvent
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			for event := range ch1 {
				events1 = append(events1, event)
			}
		}()

		go func() {
			defer wg.Done()
			for event := range ch2 {
				events2 = append(events2, event)
			}
		}()

		// 注册服务
		err = reg.Register(ctx, &registry.ServiceInstance{
			ID:      "test-1",
			Name:    "test-service",
			Address: "127.0.0.1",
		})
		require.NoError(t, err)

		// 等待事件处理
		time.Sleep(100 * time.Millisecond)
		cancel()
		wg.Wait()

		// 两个watcher都应该收到事件
		assert.Len(t, events1, 1)
		assert.Len(t, events2, 1)
	})
}

// 测试健康检查
func TestHealthCheck(t *testing.T) {
	ctx := context.Background()

	t.Run("心跳更新", func(t *testing.T) {
		reg := registry.NewServiceRegistry()

		// 注册服务
		instance := &registry.ServiceInstance{
			ID:      "test-1",
			Name:    "test-service",
			Address: "127.0.0.1",
			Port:    8080,
		}
		err := reg.Register(ctx, instance)
		require.NoError(t, err)

		// 获取初始心跳时间
		inst1, err := reg.GetInstance(ctx, "test-1")
		require.NoError(t, err)
		firstHeartbeat := inst1.LastHeartbeat

		// 等待一段时间
		time.Sleep(100 * time.Millisecond)

		// 执行健康检查
		err = reg.HealthCheck(ctx, "test-1")
		require.NoError(t, err)

		// 验证心跳时间已更新
		inst2, err := reg.GetInstance(ctx, "test-1")
		require.NoError(t, err)
		assert.True(t, inst2.LastHeartbeat.After(firstHeartbeat))
	})

	t.Run("自定义健康检查函数", func(t *testing.T) {
		checkCalled := false
		healthCheckFunc := func(ctx context.Context, instance *registry.ServiceInstance) error {
			checkCalled = true
			return nil
		}

		reg := registry.NewServiceRegistry(
			registry.WithHealthCheckFunc(healthCheckFunc),
		)

		// 注册服务
		instance := &registry.ServiceInstance{
			ID:      "test-1",
			Name:    "test-service",
			Address: "127.0.0.1",
		}
		err := reg.Register(ctx, instance)
		require.NoError(t, err)

		// 执行健康检查
		err = reg.HealthCheck(ctx, "test-1")
		require.NoError(t, err)
		assert.True(t, checkCalled)
	})
}

// 测试并发注册
func TestConcurrentRegistration(t *testing.T) {
	ctx := context.Background()
	reg := registry.NewServiceRegistry()

	// 并发注册多个服务
	var wg sync.WaitGroup
	instanceCount := 100
	
	for i := 0; i < instanceCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			instance := &registry.ServiceInstance{
				ID:      fmt.Sprintf("instance-%d", id),
				Name:    "concurrent-service",
				Address: "127.0.0.1",
				Port:    8080 + id,
			}
			
			err := reg.Register(ctx, instance)
			assert.NoError(t, err)
		}(i)
	}
	
	wg.Wait()

	// 验证所有实例都已注册
	instances, err := reg.Discover(ctx, "concurrent-service")
	require.NoError(t, err)
	assert.Len(t, instances, instanceCount)
}

// 测试负载均衡器
func TestLoadBalancers(t *testing.T) {
	instances := []*registry.ServiceInstance{
		{ID: "1", Name: "test", Address: "127.0.0.1", Port: 8081, Weight: 5},
		{ID: "2", Name: "test", Address: "127.0.0.1", Port: 8082, Weight: 3},
		{ID: "3", Name: "test", Address: "127.0.0.1", Port: 8083, Weight: 2},
	}

	t.Run("随机负载均衡", func(t *testing.T) {
		lb := registry.NewRandomLoadBalancer()
		assert.Equal(t, "random", lb.Name())

		// 多次选择，应该有不同的结果
		selected := make(map[string]int)
		for i := 0; i < 100; i++ {
			instance, err := lb.Select(instances)
			require.NoError(t, err)
			selected[instance.ID]++
		}

		// 每个实例都应该被选中过
		assert.Len(t, selected, 3)
		for _, count := range selected {
			assert.Greater(t, count, 0)
		}
	})

	t.Run("轮询负载均衡", func(t *testing.T) {
		lb := registry.NewRoundRobinLoadBalancer()
		assert.Equal(t, "round-robin", lb.Name())

		// 验证轮询顺序
		for i := 0; i < 6; i++ {
			instance, err := lb.Select(instances)
			require.NoError(t, err)
			expectedID := string(rune('1' + (i % 3)))
			assert.Equal(t, expectedID, instance.ID)
		}
	})

	t.Run("加权轮询负载均衡", func(t *testing.T) {
		lb := registry.NewWeightedRoundRobinLoadBalancer()
		assert.Equal(t, "weighted-round-robin", lb.Name())

		// 统计选择次数
		selected := make(map[string]int)
		totalSelections := 100
		for i := 0; i < totalSelections; i++ {
			instance, err := lb.Select(instances)
			require.NoError(t, err)
			selected[instance.ID]++
		}

		// 验证权重分布（允许一定误差）
		assert.InDelta(t, 50, selected["1"], 10) // 权重5，约50%
		assert.InDelta(t, 30, selected["2"], 10) // 权重3，约30%
		assert.InDelta(t, 20, selected["3"], 10) // 权重2，约20%
	})

	t.Run("最少连接负载均衡", func(t *testing.T) {
		lb := registry.NewLeastConnectionLoadBalancer()
		assert.Equal(t, "least-connection", lb.Name())

		// 第一次选择
		instance1, err := lb.Select(instances)
		require.NoError(t, err)

		// 模拟连接
		if lcLb, ok := lb.(*registry.LeastConnectionLoadBalancer); ok {
			lcLb.AddConnection(instance1.ID)
		}

		// 第二次应该选择不同的实例
		instance2, err := lb.Select(instances)
		require.NoError(t, err)
		assert.NotEqual(t, instance1.ID, instance2.ID)
	})

	t.Run("空实例列表", func(t *testing.T) {
		lbs := []registry.LoadBalancer{
			registry.NewRandomLoadBalancer(),
			registry.NewRoundRobinLoadBalancer(),
			registry.NewWeightedRoundRobinLoadBalancer(),
			registry.NewLeastConnectionLoadBalancer(),
		}

		for _, lb := range lbs {
			_, err := lb.Select([]*registry.ServiceInstance{})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "no available instances")
		}
	})

	t.Run("负载均衡器工厂", func(t *testing.T) {
		algorithms := []string{"random", "round-robin", "weighted-round-robin", "least-connection"}
		
		for _, algo := range algorithms {
			lb, err := registry.NewLoadBalancer(algo)
			require.NoError(t, err)
			assert.NotNil(t, lb)
			assert.Equal(t, algo, lb.Name())
		}

		// 未知算法
		_, err := registry.NewLoadBalancer("unknown")
		assert.Error(t, err)
	})
}

// 测试服务事件
func TestServiceEvent(t *testing.T) {
	t.Run("事件类型字符串", func(t *testing.T) {
		tests := []struct {
			eventType registry.EventType
			expected  string
		}{
			{registry.EventRegister, "REGISTER"},
			{registry.EventDeregister, "DEREGISTER"},
			{registry.EventUpdate, "UPDATE"},
			{registry.EventHealthChange, "HEALTH_CHANGE"},
			{registry.EventType(999), "UNKNOWN"},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.expected, tt.eventType.String())
		}
	})
}

// 性能测试
func BenchmarkServiceRegistry(b *testing.B) {
	ctx := context.Background()
	reg := registry.NewServiceRegistry()

	// 预先注册一些服务
	for i := 0; i < 100; i++ {
		instance := &registry.ServiceInstance{
			ID:      fmt.Sprintf("bench-%d", i),
			Name:    "bench-service",
			Address: "127.0.0.1",
			Port:    8080 + i,
		}
		reg.Register(ctx, instance)
	}

	b.Run("Register", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			instance := &registry.ServiceInstance{
				ID:      fmt.Sprintf("test-%d", i),
				Name:    "test-service",
				Address: "127.0.0.1",
				Port:    9000 + i,
			}
			reg.Register(ctx, instance)
		}
	})

	b.Run("Discover", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			reg.Discover(ctx, "bench-service")
		}
	})

	b.Run("HealthCheck", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			reg.HealthCheck(ctx, "bench-0")
		}
	})
}

func BenchmarkLoadBalancer(b *testing.B) {
	instances := make([]*registry.ServiceInstance, 10)
	for i := 0; i < 10; i++ {
		instances[i] = &registry.ServiceInstance{
			ID:     fmt.Sprintf("bench-%d", i),
			Weight: i + 1,
		}
	}

	b.Run("Random", func(b *testing.B) {
		lb := registry.NewRandomLoadBalancer()
		for i := 0; i < b.N; i++ {
			lb.Select(instances)
		}
	})

	b.Run("RoundRobin", func(b *testing.B) {
		lb := registry.NewRoundRobinLoadBalancer()
		for i := 0; i < b.N; i++ {
			lb.Select(instances)
		}
	})

	b.Run("WeightedRoundRobin", func(b *testing.B) {
		lb := registry.NewWeightedRoundRobinLoadBalancer()
		for i := 0; i < b.N; i++ {
			lb.Select(instances)
		}
	})
}