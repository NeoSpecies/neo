package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	fmt.Println("Testing connection to Neo IPC server...")
	
	// Try to connect to the IPC server
	conn, err := net.DialTimeout("tcp", "localhost:9999", 5*time.Second)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		// Try with explicit IP
		conn, err = net.DialTimeout("tcp", "127.0.0.1:9999", 5*time.Second)
		if err != nil {
			fmt.Printf("Failed to connect with IP: %v\n", err)
			return
		}
	}
	defer conn.Close()
	
	fmt.Printf("Successfully connected to %s\n", conn.RemoteAddr())
	
	// Test if we can write
	testMsg := []byte{0x00, 0x00, 0x00, 0x01, 0x04} // Length 1, type 4 (heartbeat)
	n, err := conn.Write(testMsg)
	if err != nil {
		fmt.Printf("Failed to write: %v\n", err)
		return
	}
	fmt.Printf("Wrote %d bytes\n", n)
}