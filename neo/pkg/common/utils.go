package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"net"
	"os"
	"syscall"
)

// CompressData 压缩数据
func CompressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecompressData 解压数据
func DecompressData(data []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

// ZeroCopySendFile 使用零拷贝技术发送文件
func ZeroCopySendFile(dst net.Conn, src *os.File) error {
	stat, err := src.Stat()
	if err != nil {
		return err
	}

	// 获取文件描述符
	srcFd := int(src.Fd())
	dstFile, err := dst.(*net.TCPConn).File()
	if err != nil {
		return err
	}
	defer dstFile.Close()
	dstFd := int(dstFile.Fd())

	// 使用sendfile系统调用
	_, err = syscall.Sendfile(dstFd, srcFd, nil, int(stat.Size()))
	return err
}

// ShouldCompress 判断是否应该压缩数据
func ShouldCompress(data []byte) bool {
	// 当数据大于1MB时进行压缩
	return len(data) > 1024*1024
}
