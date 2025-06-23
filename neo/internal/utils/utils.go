package utils

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"os"
	"syscall"
)

// CompressData 使用gzip压缩数据
// 适配场景：协议传输前的数据压缩
func CompressData(data []byte) ([]byte, error) {
	buf := &bytes.Buffer{}
	zw := gzip.NewWriter(buf)

	if _, err := zw.Write(data); err != nil {
		zw.Close()
		return nil, fmt.Errorf("压缩数据失败: %w", err)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("关闭gzip写入器失败: %w", err)
	}

	return buf.Bytes(), nil
}

// DecompressData 使用gzip解压缩数据
// 适配场景：接收到压缩数据后的解压处理
func DecompressData(data []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("创建gzip读取器失败: %w", err)
	}
	defer zr.Close()

	result, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("解压缩数据失败: %w", err)
	}

	return result, nil
}

// ZeroCopySendFile 通过零拷贝方式发送文件内容
// 适配场景：大文件传输优化（如日志、配置文件分发）
// 参数说明：
//   dst: neo连接池中的net.Conn类型连接
//   src: 待发送的文件句柄
func ZeroCopySendFile(dst net.Conn, src *os.File) error {
	// 获取连接的底层文件描述符
	connFile, err := dst.(*net.TCPConn).File()
	if err != nil {
		return fmt.Errorf("获取连接文件描述符失败: %w", err)
	}
	defer connFile.Close()

	// 获取文件信息
	fileInfo, err := src.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	var totalSent int64 = 0
	fileSize := fileInfo.Size()

	// 循环发送直到所有数据传输完成
	for totalSent < fileSize {
		sent, err := syscall.Sendfile(int(connFile.Fd()), int(src.Fd()), &totalSent, int(fileSize-totalSent))
		if err != nil && err != syscall.EINTR {
			return fmt.Errorf("零拷贝发送失败: %w", err)
		}
		if sent == 0 {
			break // EOF
		}
	}

	if totalSent != fileSize {
		return fmt.Errorf("文件发送不完整，已发送 %d/%d 字节", totalSent, fileSize)
	}

	return nil
}

// ShouldCompress 判断数据是否需要压缩
// 决策依据：数据大小超过1KB且压缩率预估收益>10%
func ShouldCompress(data []byte) bool {
	// 小数据不压缩（小于1KB）
	if len(data) < 1024 {
		return false
	}

	// 预压缩测试（取前1KB数据预估压缩率）
	testSize := len(data)
	if testSize > 1024 {
		testSize = 1024
	}

	compressed, err := CompressData(data[:testSize])
	if err != nil || len(compressed) >= testSize*9/10 {
		return false // 压缩失败或压缩率低于10%
	}

	return true
}