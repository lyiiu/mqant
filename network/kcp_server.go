// Package network tcp服务器
package network

import (
	"crypto/sha1"
	"net"
	"sync"
	"time"

	"github.com/liangdas/mqant/log"

	"github.com/xtaci/kcp-go/v5"
	"golang.org/x/crypto/pbkdf2"
)

const (
	// SALT is use for pbkdf2 key expansion
	SALT = "kcp-go"
	// maximum supported smux version
	maxSmuxVer = 2
	// stream copy buffer size
	bufSize = 4096
)

// KCPServer tcp服务器
type KCPServer struct {
	Addr         string
	Key          string
	Crypt        string
	Mtu          int
	NoDelay      int
	Interval     int
	Resend       int
	NC           int
	DataShards   int
	ParityShards int
	NewAgent     func(*KCPConn) Agent

	ln         *kcp.Listener
	mutexConns sync.Mutex
	wgLn       sync.WaitGroup
	wgConns    sync.WaitGroup
}

// Start 开始kcp监听
func (server *KCPServer) Start() {
	server.init()
	log.Info("KCP Listen :%s", server.Addr)
	go server.run()
}

func (server *KCPServer) init() {
	var block kcp.BlockCrypt
	if server.Crypt != "" {
		pass := pbkdf2.Key([]byte(server.Key), []byte(SALT), 4096, 32, sha1.New)
		block, _ = kcp.NewAESBlockCrypt(pass)
	}

	ln, err := kcp.ListenWithOptions(server.Addr, block, server.DataShards, server.ParityShards)

	// ln, err := net.Listen("tcp", server.Addr)

	if err != nil {
		log.Warning("%v", err)
	}

	if server.NewAgent == nil {
		log.Warning("NewAgent must not be nil")
	}

	server.ln = ln
}

func (server *KCPServer) run() {
	server.wgLn.Add(1)
	defer server.wgLn.Done()

	var tempDelay time.Duration
	for {
		conn, err := server.ln.AcceptKCP()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Info("accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return
		}
		tempDelay = 0
		conn.SetMtu(server.Mtu)
		conn.SetNoDelay(server.NoDelay, server.Interval, server.Resend, server.NC)
		conn.SetACKNoDelay(true)
		kcpConn := newKCPConn(conn)
		agent := server.NewAgent(kcpConn)
		go func() {
			server.wgConns.Add(1)
			agent.Run()

			// cleanup
			kcpConn.Close()
			agent.OnClose()

			server.wgConns.Done()
		}()
	}
}

// Close 关闭TCP监听
func (server *KCPServer) Close() {
	server.ln.Close()
	server.wgLn.Wait()
	server.wgConns.Wait()
}
