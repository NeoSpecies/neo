package main

import (
	"net"
	"sync"
	"time"
)

// WorkerPool 协程池实现
type WorkerPool struct {
	workers int
	tasks   chan func()
	wg      sync.WaitGroup
}

// NewWorkerPool 创建新的协程池
func NewWorkerPool(workers int) *WorkerPool {
	pool := &WorkerPool{
		workers: workers,
		tasks:   make(chan func()),
	}
	pool.start()
	return pool
}

// start 启动工作协程
func (p *WorkerPool) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for task := range p.tasks {
				task()
			}
		}()
	}
}

// Submit 提交任务到协程池
func (p *WorkerPool) Submit(task func()) {
	p.tasks <- task
}

// Close 关闭协程池
func (p *WorkerPool) Close() {
	close(p.tasks)
	p.wg.Wait()
}

// ConnPool 连接池实现
type ConnPool struct {
	mu       sync.Mutex
	conns    map[string][]net.Conn
	maxConns int
	timeout  time.Duration
}

// NewConnPool 创建新的连接池
func NewConnPool(maxConns int, timeout time.Duration) *ConnPool {
	return &ConnPool{
		conns:    make(map[string][]net.Conn),
		maxConns: maxConns,
		timeout:  timeout,
	}
}

// Get 从连接池获取连接
func (p *ConnPool) Get(addr string) (net.Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 检查是否有可用连接
	if conns := p.conns[addr]; len(conns) > 0 {
		conn := conns[len(conns)-1]
		p.conns[addr] = conns[:len(conns)-1]

		// 检查连接是否还有效
		if err := p.checkConn(conn); err != nil {
			conn.Close()
			return p.createConn(addr)
		}
		return conn, nil
	}

	return p.createConn(addr)
}

// Put 将连接放回连接池
func (p *ConnPool) Put(addr string, conn net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 检查连接池是否已满
	if len(p.conns[addr]) >= p.maxConns {
		conn.Close()
		return
	}

	// 检查连接是否还有效
	if err := p.checkConn(conn); err != nil {
		conn.Close()
		return
	}

	p.conns[addr] = append(p.conns[addr], conn)
}

// createConn 创建新连接
func (p *ConnPool) createConn(addr string) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", addr, p.timeout)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// checkConn 检查连接是否有效
func (p *ConnPool) checkConn(conn net.Conn) error {
	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(p.timeout))

	// 尝试读取一个字节
	one := []byte{0}
	_, err := conn.Read(one)
	if err != nil {
		return err
	}

	// 重置读取超时
	conn.SetReadDeadline(time.Time{})
	return nil
}

// Close 关闭连接池
func (p *ConnPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conns := range p.conns {
		for _, conn := range conns {
			conn.Close()
		}
	}
	p.conns = make(map[string][]net.Conn)
}
