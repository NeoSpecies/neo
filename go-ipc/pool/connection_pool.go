package pool

import (
	"context"
	"net"
	"sync"
	"time"
)

// Connection wraps a net.Conn with additional metadata
type Connection struct {
	conn      net.Conn
	lastUsed  time.Time
	inUse     bool
	errorCount int
}

// ConnectionPool manages a pool of connections
type ConnectionPool struct {
	mu          sync.RWMutex
	connections []*Connection
	maxSize     int
	minSize     int
	timeout     time.Duration
	factory     func() (net.Conn, error)
}

// Config represents connection pool configuration
type Config struct {
	MaxSize         int
	MinSize         int
	ConnectTimeout  time.Duration
	IdleTimeout     time.Duration
	MaxRetries      int
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config Config, factory func() (net.Conn, error)) *ConnectionPool {
	pool := &ConnectionPool{
		maxSize: config.MaxSize,
		minSize: config.MinSize,
		timeout: config.ConnectTimeout,
		factory: factory,
	}

	// Initialize minimum connections
	for i := 0; i < config.MinSize; i++ {
		if conn, err := factory(); err == nil {
			pool.connections = append(pool.connections, &Connection{
				conn:     conn,
				lastUsed: time.Now(),
			})
		}
	}

	// Start background maintenance
	go pool.maintain(config.IdleTimeout)

	return pool
}

// Acquire gets a connection from the pool
func (p *ConnectionPool) Acquire(ctx context.Context) (*Connection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Try to find an available connection
	for _, conn := range p.connections {
		if !conn.inUse {
			conn.inUse = true
			conn.lastUsed = time.Now()
			return conn, nil
		}
	}

	// Create new connection if pool is not full
	if len(p.connections) < p.maxSize {
		conn, err := p.factory()
		if err != nil {
			return nil, err
		}

		connection := &Connection{
			conn:     conn,
			lastUsed: time.Now(),
			inUse:    true,
		}
		p.connections = append(p.connections, connection)
		return connection, nil
	}

	// Wait for available connection
	return nil, ErrPoolExhausted
}

// Release returns a connection to the pool
func (p *ConnectionPool) Release(conn *Connection) {
	p.mu.Lock()
	defer p.mu.Unlock()

	conn.inUse = false
	conn.lastUsed = time.Now()
}

// maintain performs periodic maintenance on the connection pool
func (p *ConnectionPool) maintain(idleTimeout time.Duration) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		now := time.Now()

		// Remove idle connections exceeding minimum pool size
		i := 0
		for _, conn := range p.connections {
			if !conn.inUse && now.Sub(conn.lastUsed) > idleTimeout && len(p.connections) > p.minSize {
				conn.conn.Close()
				continue
			}
			p.connections[i] = conn
			i++
		}
		p.connections = p.connections[:i]

		// Health check on remaining connections
		for _, conn := range p.connections {
			if !conn.inUse {
				if err := p.healthCheck(conn); err != nil {
					conn.errorCount++
					if conn.errorCount > 3 {
						// Replace unhealthy connection
						if newConn, err := p.factory(); err == nil {
							conn.conn.Close()
							conn.conn = newConn
							conn.errorCount = 0
						}
					}
				}
			}
		}
		p.mu.Unlock()
	}
}

// healthCheck performs a health check on a connection
func (p *ConnectionPool) healthCheck(conn *Connection) error {
	// Implementation of connection health check
	// This could include sending a ping message or checking socket state
	return nil
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.connections {
		conn.conn.Close()
	}
	p.connections = nil
} 