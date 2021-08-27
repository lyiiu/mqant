// Package network kcp网络控制器
package network

import (
	"bytes"
	"io"
	"net"
	"sync"
	"time"

	"github.com/xtaci/kcp-go/v5"
)

// KCPConn kcp连接
type KCPConn struct {
	io.Reader //Read(p []byte) (n int, err error)
	io.Writer //Write(p []byte) (n int, err error)
	sync.Mutex
	bufLocks  chan bool // 当有写入一次数据设置一次
	buffer    bytes.Buffer
	conn      *kcp.UDPSession
	closeFlag bool
}

func newKCPConn(conn *kcp.UDPSession) *KCPConn {
	kcpConn := new(KCPConn)
	kcpConn.conn = conn

	return kcpConn
}

func (kcpConn *KCPConn) doDestroy() {
	// kcpConn.conn.(*net.KCPConn).SetLinger(0)
	kcpConn.conn.Close()

	if !kcpConn.closeFlag {
		kcpConn.closeFlag = true
	}
}

// Destroy 断连
func (kcpConn *KCPConn) Destroy() {
	kcpConn.Lock()
	defer kcpConn.Unlock()

	kcpConn.doDestroy()
}

// Close 关闭tcp连接
func (kcpConn *KCPConn) Close() error {
	kcpConn.Lock()
	defer kcpConn.Unlock()
	if kcpConn.closeFlag {
		return nil
	}

	kcpConn.closeFlag = true
	return kcpConn.conn.Close()
}

// Write b must not be modified by the others goroutines
func (kcpConn *KCPConn) Write(b []byte) (n int, err error) {
	kcpConn.Lock()
	defer kcpConn.Unlock()
	if kcpConn.closeFlag || b == nil {
		return
	}

	return kcpConn.conn.Write(b)
}

// Read read data
func (kcpConn *KCPConn) Read(b []byte) (int, error) {
	return kcpConn.conn.Read(b)
}

// LocalAddr 本地socket端口地址
func (kcpConn *KCPConn) LocalAddr() net.Addr {
	return kcpConn.conn.LocalAddr()
}

// RemoteAddr 远程socket端口地址
func (kcpConn *KCPConn) RemoteAddr() net.Addr {
	return kcpConn.conn.RemoteAddr()
}

// SetDeadline A zero value for t means I/O operations will not time out.
func (kcpConn *KCPConn) SetDeadline(t time.Time) error {
	return kcpConn.conn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls.
// A zero value for t means Read will not time out.
func (kcpConn *KCPConn) SetReadDeadline(t time.Time) error {
	return kcpConn.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (kcpConn *KCPConn) SetWriteDeadline(t time.Time) error {
	return kcpConn.conn.SetWriteDeadline(t)
}
