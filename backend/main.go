package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	localWSAddr   = "127.0.0.1:23333"
	localAPIAddr  = "127.0.0.1:23334"
	upstreamWSURL = "wss://broadcastlv.chat.bilibili.com/sub"
)

var configPath = filepath.Join("..", "pdj", "dograin", "pdj_config.json")

type Config struct {
	RoomID int    `json:"roomid"`
	UID    int    `json:"uid"`
	Cookie string `json:"cookie"`
}

type ConfigStore struct{ mu sync.Mutex }

func (s *ConfigStore) Load() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := Config{RoomID: 26714219}
	b, err := os.ReadFile(configPath)
	if err != nil {
		_ = s.saveLocked(cfg)
		return cfg
	}
	_ = json.Unmarshal(b, &cfg)
	if cfg.RoomID == 0 {
		cfg.RoomID = 26714219
	}
	return cfg
}

func (s *ConfigStore) Save(cfg Config) (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cfg.RoomID == 0 {
		cfg.RoomID = 26714219
	}
	return cfg, s.saveLocked(cfg)
}

func (s *ConfigStore) saveLocked(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, b, 0o644)
}

type wsConn struct {
	conn       net.Conn
	rd         *bufio.Reader
	wrMu       sync.Mutex
	isClient   bool // client->server frames must be masked
	closed     atomic.Bool
	closeOnce  sync.Once
	closeCh    chan struct{}
	readDeadMu sync.Mutex
}

func newWSConn(c net.Conn, isClient bool) *wsConn {
	return &wsConn{conn: c, rd: bufio.NewReader(c), isClient: isClient, closeCh: make(chan struct{})}
}

func (w *wsConn) Close() error {
	var err error
	w.closeOnce.Do(func() {
		w.closed.Store(true)
		close(w.closeCh)
		err = w.conn.Close()
	})
	return err
}

func (w *wsConn) SetReadDeadline(t time.Time) error {
	w.readDeadMu.Lock()
	defer w.readDeadMu.Unlock()
	return w.conn.SetReadDeadline(t)
}

func (w *wsConn) WriteText(data []byte) error   { return w.writeFrame(0x1, data) }
func (w *wsConn) WriteBinary(data []byte) error { return w.writeFrame(0x2, data) }
func (w *wsConn) WritePong(data []byte) error   { return w.writeFrame(0xA, data) }

func (w *wsConn) writeFrame(opcode byte, payload []byte) error {
	if w.closed.Load() {
		return io.EOF
	}
	w.wrMu.Lock()
	defer w.wrMu.Unlock()

	finOpcode := byte(0x80) | (opcode & 0x0F)
	head := []byte{finOpcode}
	maskBit := byte(0)
	if w.isClient {
		maskBit = 0x80
	}
	plen := len(payload)
	switch {
	case plen <= 125:
		head = append(head, maskBit|byte(plen))
	case plen <= math.MaxUint16:
		head = append(head, maskBit|126, byte(plen>>8), byte(plen))
	default:
		head = append(head, maskBit|127)
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(plen))
		head = append(head, b...)
	}

	if w.isClient {
		mask := make([]byte, 4)
		if _, err := rand.Read(mask); err != nil {
			return err
		}
		head = append(head, mask...)
		masked := make([]byte, plen)
		for i := 0; i < plen; i++ {
			masked[i] = payload[i] ^ mask[i%4]
		}
		if _, err := w.conn.Write(head); err != nil {
			return err
		}
		_, err := w.conn.Write(masked)
		return err
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
	fin := first&0x80 != 0
	opcode := first & 0x0F
	if !fin {
		return 0, nil, errors.New("fragmented frames unsupported")
	}
	masked := second&0x80 != 0
	plen := int(second & 0x7F)
	if plen == 126 {
		len2 := make([]byte, 2)
		if _, err = io.ReadFull(w.rd, len2); err != nil {
			return 0, nil, err
		}
		plen = int(binary.BigEndian.Uint16(len2))
	} else if plen == 127 {
		len8 := make([]byte, 8)
		if _, err = io.ReadFull(w.rd, len8); err != nil {
			return 0, nil, err
		}
		u := binary.BigEndian.Uint64(len8)
		if u > 32*1024*1024 {
			return 0, nil, errors.New("frame too large")
		}
		plen = int(u)
	}
	var maskKey [4]byte
	if masked {
		if _, err = io.ReadFull(w.rd, maskKey[:]); err != nil {
			return 0, nil, err
		}
	}
	payload := make([]byte, plen)
	if plen > 0 {
		if _, err = io.ReadFull(w.rd, payload); err != nil {
			return 0, nil, err
		}
	}
	if masked {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}
	return opcode, payload, nil
}

func serverUpgrade(w http.ResponseWriter, req *http.Request) (*wsConn, error) {
	if !strings.EqualFold(req.Header.Get("Connection"), "Upgrade") && !strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade") {
		return nil, errors.New("invalid Connection header")
	}
	if !strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
		return nil, errors.New("invalid Upgrade header")
	}
	key := strings.TrimSpace(req.Header.Get("Sec-WebSocket-Key"))
	if key == "" {
		return nil, errors.New("missing websocket key")
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("hijacking not supported")
	}
	conn, buf, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	accept := computeAcceptKey(key)
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + accept + "\r\n\r\n"
	if _, err = buf.WriteString(resp); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err = buf.Flush(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return newWSConn(conn, false), nil
}

func clientDial(rawURL string, headers map[string]string) (*wsConn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	host := u.Host
	if !strings.Contains(host, ":") {
		if u.Scheme == "wss" {
			host += ":443"
		} else {
			host += ":80"
		}
	}
	var conn net.Conn
	if u.Scheme == "wss" {
		conn, err = tls.Dial("tcp", host, &tls.Config{ServerName: strings.Split(u.Host, ":")[0], MinVersion: tls.VersionTLS12})
	} else {
		conn, err = net.Dial("tcp", host)
	}
	if err != nil {
		return nil, err
	}

	keyRaw := make([]byte, 16)
	_, _ = rand.Read(keyRaw)
	wsKey := base64.StdEncoding.EncodeToString(keyRaw)
	path := u.RequestURI()
	if path == "" {
		path = "/"
	}

	var reqBuf bytes.Buffer
	fmt.Fprintf(&reqBuf, "GET %s HTTP/1.1\r\n", path)
	fmt.Fprintf(&reqBuf, "Host: %s\r\n", u.Host)
	fmt.Fprintf(&reqBuf, "Upgrade: websocket\r\n")
	fmt.Fprintf(&reqBuf, "Connection: Upgrade\r\n")
	fmt.Fprintf(&reqBuf, "Sec-WebSocket-Version: 13\r\n")
	fmt.Fprintf(&reqBuf, "Sec-WebSocket-Key: %s\r\n", wsKey)
	for k, v := range headers {
		fmt.Fprintf(&reqBuf, "%s: %s\r\n", k, v)
	}
	fmt.Fprintf(&reqBuf, "\r\n")

	if _, err = conn.Write(reqBuf.Bytes()); err != nil {
		_ = conn.Close()
		return nil, err
	}

	rd := bufio.NewReader(conn)
	status, err := rd.ReadString('\n')
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if !strings.Contains(status, "101") {
		_ = conn.Close()
		return nil, fmt.Errorf("websocket handshake failed: %s", strings.TrimSpace(status))
	}
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		if line == "\r\n" {
			break
		}
	}
	return &wsConn{conn: conn, rd: rd, isClient: true, closeCh: make(chan struct{})}, nil
}

func computeAcceptKey(key string) string {
	h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h[:])
}

type Relay struct {
	store         *ConfigStore
	clients       map[*wsConn]struct{}
	clientsMu     sync.RWMutex
	configVersion atomic.Int64
}

func NewRelay(store *ConfigStore) *Relay {
	return &Relay{store: store, clients: map[*wsConn]struct{}{}}
}
func (r *Relay) NotifyConfigChanged() { r.configVersion.Add(1) }

func (r *Relay) addClient(c *wsConn) {
	r.clientsMu.Lock()
	defer r.clientsMu.Unlock()
	r.clients[c] = struct{}{}
}

func (r *Relay) removeClient(c *wsConn) {
	r.clientsMu.Lock()
	defer r.clientsMu.Unlock()
	delete(r.clients, c)
}

func (r *Relay) broadcast(payload any) {
	msg, _ := json.Marshal(payload)
	r.clientsMu.RLock()
	clients := make([]*wsConn, 0, len(r.clients))
	for c := range r.clients {
		clients = append(clients, c)
	}
	r.clientsMu.RUnlock()
	for _, c := range clients {
		if err := c.WriteText(msg); err != nil {
			r.removeClient(c)
			_ = c.Close()
		}
	}
}

func (r *Relay) wsHandler(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/danmu/sub" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	conn, err := serverUpgrade(w, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	r.addClient(conn)
	r.broadcast(map[string]any{"type": "PDJ_STATUS", "status": "client_connected"})
	defer func() {
		r.removeClient(conn)
		_ = conn.Close()
	}()

	for {
		op, payload, err := conn.ReadFrame()
		if err != nil {
			return
		}
		switch op {
		case 0x8:
			return
		case 0x9:
			_ = conn.WritePong(payload)
		}
	}
}

func (r *Relay) apiHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if req.Method == http.MethodOptions {
		_, _ = w.Write([]byte(`{"ok":true}`))
		return
	}
	if req.URL.Path != "/api/config" {
		http.NotFound(w, req)
		return
	}

	switch req.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode(r.store.Load())
	case http.MethodPost:
		var cfg Config
		if err := json.NewDecoder(req.Body).Decode(&cfg); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		saved, err := r.store.Save(cfg)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}
		r.NotifyConfigChanged()
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "config": saved})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func encodePacket(payload map[string]any, op uint32) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	packetLen := uint32(16 + len(body))
	buf := make([]byte, packetLen)
	binary.BigEndian.PutUint32(buf[0:4], packetLen)
	binary.BigEndian.PutUint16(buf[4:6], 16)
	binary.BigEndian.PutUint16(buf[6:8], 1)
	binary.BigEndian.PutUint32(buf[8:12], op)
	binary.BigEndian.PutUint32(buf[12:16], 1)
	copy(buf[16:], body)
	return buf, nil
}

func decodePackets(blob []byte) []map[string]any {
	out := make([]map[string]any, 0)
	offset := 0
	for offset+16 <= len(blob) {
		packetLen := int(binary.BigEndian.Uint32(blob[offset : offset+4]))
		headerLen := int(binary.BigEndian.Uint16(blob[offset+4 : offset+6]))
		ver := int(binary.BigEndian.Uint16(blob[offset+6 : offset+8]))
		op := int(binary.BigEndian.Uint32(blob[offset+8 : offset+12]))
		if packetLen < 16 || offset+packetLen > len(blob) || headerLen < 16 {
			break
		}
		body := blob[offset+headerLen : offset+packetLen]
		offset += packetLen

		switch op {
		case 3:
			pop := 0
			if len(body) >= 4 {
				pop = int(binary.BigEndian.Uint32(body[:4]))
			}
			out = append(out, map[string]any{"type": "PDJ_STATUS", "status": "popularity", "popularity": pop})
		case 5:
			if ver == 2 {
				zr, err := zlib.NewReader(bytes.NewReader(body))
				if err != nil {
					continue
				}
				b, err := io.ReadAll(zr)
				_ = zr.Close()
				if err == nil {
					out = append(out, decodePackets(b)...)
				}
				continue
			}
			var item map[string]any
			if err := json.Unmarshal(bytes.Trim(body, "\x00"), &item); err == nil {
				out = append(out, item)
			}
		}
	}
	return out
}

func (r *Relay) runUpstream(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		cfg := r.store.Load()
		version := r.configVersion.Load()
		r.broadcast(map[string]any{"type": "PDJ_STATUS", "status": "connecting", "roomid": cfg.RoomID})

		headers := map[string]string{}
		if cfg.Cookie != "" {
			headers["Cookie"] = cfg.Cookie
		}
		up, err := clientDial(upstreamWSURL, headers)
		if err != nil {
			r.broadcast(map[string]any{"type": "PDJ_STATUS", "status": "upstream_error", "error": err.Error()})
			time.Sleep(2 * time.Second)
			continue
		}

		auth := map[string]any{"uid": cfg.UID, "roomid": cfg.RoomID, "protover": 2, "platform": "web", "type": 2, "clientver": "1.16.3"}
		pkt, _ := encodePacket(auth, 7)
		if err = up.WriteBinary(pkt); err != nil {
			_ = up.Close()
			time.Sleep(1500 * time.Millisecond)
			continue
		}
		r.broadcast(map[string]any{"type": "PDJ_STATUS", "status": "connected", "roomid": cfg.RoomID})

		hbTicker := time.NewTicker(30 * time.Second)
		for {
			if r.configVersion.Load() != version {
				r.broadcast(map[string]any{"type": "PDJ_STATUS", "status": "config_changed"})
				break
			}
			_ = up.SetReadDeadline(time.Now().Add(2 * time.Second))
			op, payload, err := up.ReadFrame()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					select {
					case <-hbTicker.C:
						hb, _ := encodePacket(map[string]any{}, 2)
						_ = up.WriteBinary(hb)
					default:
					}
					continue
				}
				r.broadcast(map[string]any{"type": "PDJ_STATUS", "status": "upstream_error", "error": err.Error()})
				break
			}
			switch op {
			case 0x8:
				break
			case 0x9:
				_ = up.WritePong(payload)
			case 0x2:
				for _, item := range decodePackets(payload) {
					r.broadcast(item)
				}
			}
		}
		hbTicker.Stop()
		_ = up.Close()
		time.Sleep(1500 * time.Millisecond)
	}
}

func main() {
	if v := os.Getenv("PDJ_CONFIG_PATH"); v != "" {
		configPath = v
	}
	store := &ConfigStore{}
	relay := NewRelay(store)

	wsMux := http.NewServeMux()
	wsMux.HandleFunc("/danmu/sub", relay.wsHandler)
	wsMux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("paiduiji backend running"))
	})

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/config", relay.apiHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go relay.runUpstream(ctx)

	go func() {
		log.Printf("HTTP API listening on http://%s/api/config", localAPIAddr)
		if err := http.ListenAndServe(localAPIAddr, apiMux); err != nil {
			log.Fatalf("api server failed: %v", err)
		}
	}()

	log.Printf("WS relay + web listening on http://%s", localWSAddr)
	if err := http.ListenAndServe(localWSAddr, wsMux); err != nil {
		log.Fatalf("ws server failed: %v", err)
	}
}
