package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

type wsConn struct {
	conn net.Conn
	rd   *bufio.Reader
	mu   sync.Mutex
}

func newWSConn(c net.Conn) *wsConn { return &wsConn{conn: c, rd: bufio.NewReader(c)} }

func (w *wsConn) Close() error { return w.conn.Close() }

func (w *wsConn) WriteText(text string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	payload := []byte(text)
	head := []byte{0x81}
	sz := len(payload)
	switch {
	case sz <= 125:
		head = append(head, byte(sz))
	case sz <= math.MaxUint16:
		head = append(head, 126, byte(sz>>8), byte(sz))
	default:
		return errors.New("消息太大")
	}
	if _, err := w.conn.Write(head); err != nil {
		return err
	}
	_, err := w.conn.Write(payload)
	return err
}

func (w *wsConn) ReadFrame() (byte, []byte, error) {
	first, err := w.rd.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	second, err := w.rd.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	opcode := first & 0x0F
	masked := second&0x80 != 0
	plen := int(second & 0x7F)
	if plen == 126 {
		len2 := make([]byte, 2)
		if _, err = io.ReadFull(w.rd, len2); err != nil {
			return 0, nil, err
		}
		plen = int(binary.BigEndian.Uint16(len2))
	}
	var mask [4]byte
	if masked {
		if _, err = io.ReadFull(w.rd, mask[:]); err != nil {
			return 0, nil, err
		}
	}
	payload := make([]byte, plen)
	if _, err = io.ReadFull(w.rd, payload); err != nil {
		return 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return opcode, payload, nil
}

func computeAcceptKey(key string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, strings.TrimSpace(key)+"258EAFA5-E914-47DA-95CA-C5AB0DC85B11")
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func upgrade(w http.ResponseWriter, req *http.Request) (*wsConn, error) {
	if !strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade") || !strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
		return nil, errors.New("不是 websocket 升级请求")
	}
	key := req.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, errors.New("缺少 Sec-WebSocket-Key")
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("当前环境不支持 Hijack")
	}
	conn, buf, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + computeAcceptKey(key) + "\r\n\r\n"
	if _, err = buf.WriteString(resp); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err = buf.Flush(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return newWSConn(conn), nil
}

func main() {
	var (
		current *wsConn
		mu      sync.RWMutex
	)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrade(w, r)
		if err != nil {
			log.Printf("升级失败: %v", err)
			return
		}
		mu.Lock()
		if current != nil {
			_ = current.Close()
		}
		current = c
		mu.Unlock()
		log.Printf("客户端已连接: %s", r.RemoteAddr)

		for {
			op, msg, err := c.ReadFrame()
			if err != nil {
				log.Printf("客户端断开: %v", err)
				mu.Lock()
				if current == c {
					current = nil
				}
				mu.Unlock()
				_ = c.Close()
				return
			}
			if op == 0x1 {
				log.Printf("收到: %s", string(msg))
			}
		}
	})

	go func() {
		fmt.Println("输入内容回车即可发送；/hello 发送你好；/json 发送示例 JSON")
		scan := bufio.NewScanner(os.Stdin)
		for scan.Scan() {
			line := strings.TrimSpace(scan.Text())
			if line == "" {
				continue
			}
			send := line
			if line == "/hello" {
				send = "你好"
			} else if line == "/json" {
				send = `{"msg":"hello"}`
			}
			mu.RLock()
			c := current
			mu.RUnlock()
			if c == nil {
				log.Println("发送失败：当前没有客户端连接")
				continue
			}
			if err := c.WriteText(send); err != nil {
				log.Printf("发送失败: %v", err)
				continue
			}
			log.Printf("发送: %s", send)
		}
	}()

	log.Println("WebSocket 服务启动: ws://127.0.0.1:23223/")
	log.Fatal(http.ListenAndServe("127.0.0.1:23223", nil))
}
