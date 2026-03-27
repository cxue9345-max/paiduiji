package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultListenAddr = ":9816"
	defaultStaticDir  = "../../frontend/dograin"
	configFileName    = "config.yaml"
	dateTimeLayout    = "2006-01-02 15:04:05"
	monthFileLayout   = "2006_01"
	runLogNameLayout  = "06-01-02-15-04-05"
	runLogFolder      = "log"
)

type AppConfig struct {
	ListenAddr  string
	ProxyTarget string
	DataDir     string
	QueueDir    string
	LogDir      string
	LogLevel    string
	Panel       PanelConfig
	MyJS        MyJSConfig
}

type PanelConfig struct {
	RoomID int
	UID    int
	Cookie string
}

type MyJSConfig struct {
	Admins                 []string `json:"admins"`
	BanAdmins              []string `json:"ban_admins"`
	Jianzhang              []string `json:"jianzhang"`
	Fankui                 bool     `json:"fankui"`
	GuanliFankui           bool     `json:"guanli_fankui"`
	PaiduiListLengthMax    int      `json:"paidui_list_length_max"`
	Jianzhangchadui        bool     `json:"jianzhangchadui"`
	JianzhangCDKind        int      `json:"jianzhang_cd_kind"`
	JianzhangCDCishu       int      `json:"jianzhang_cd_cishu"`
	FangguanCanDoing       bool     `json:"fangguan_can_doing"`
	AllSuoyourenbukepaidui bool     `json:"all_suoyourenbukepaidui"`
	YHbotKaiguan           bool     `json:"yhbot_kaiguan"`
	YHbotID                string   `json:"yhbotid"`
	YHbotMsgType           string   `json:"yhbot_msg_type"`
	YHbotWebhookToken      string   `json:"yhbot_webhook_token"`
	WsZbtoolKaiguan        bool     `json:"ws_zbtool_kaiguan"`
	QYWXKaiguan            bool     `json:"qywx_kaiguan"`
	WXWebhook              string   `json:"wx_webhook"`
	OnlyMyfunsPaidui       bool     `json:"only_myfuns_paidui"`
	LiwuChaduiKg           bool     `json:"liwu_chadui_kg"`
	LiwuPaiduiKg           bool     `json:"liwu_paidui_kg"`
	LiwuChaduiKind         int      `json:"liwu_chadui_kind"`
	LiwuPaiduiKind         int      `json:"liwu_paidui_kind"`
}

type QueueItem struct {
	Seq       int
	ID        string
	Remark    string
	CreatedAt time.Time
}

type QueueStore struct {
	mu sync.Mutex

	queueDir  string
	logsDir   string
	current   string
	logFile   string
	monthExpr *regexp.Regexp
}

type ConfigManager struct {
	mu      sync.RWMutex
	path    string
	cfg     AppConfig
	modTime time.Time
	size    int64
}

type ProxyManager struct {
	mu    sync.RWMutex
	proxy *httputil.ReverseProxy
}

type wsConn struct {
	conn      net.Conn
	rd        *bufio.Reader
	wmu       sync.Mutex
	closed    bool
	closeOnce sync.Once
}

type Hub struct {
	mu      sync.Mutex
	clients map[*wsConn]struct{}
}

type Server struct {
	cfgMgr    *ConfigManager
	store     *QueueStore
	proxyMgr  *ProxyManager
	hub       *Hub
	static    http.Handler
	staticDir string
}

var activeRunLogFilter *levelFilterWriter

func main() {
	cfgMgr, err := NewConfigManager(resolveConfigPath())
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	cfg := cfgMgr.Get()
	if err := setupRunLogger(cfg.LogLevel); err != nil {
		log.Fatalf("初始化运行日志失败: %v", err)
	}

	store, err := NewQueueStore(cfg)
	if err != nil {
		log.Fatalf("初始化存储失败: %v", err)
	}

	staticDir := resolveStaticDir()
	s := &Server{
		cfgMgr:    cfgMgr,
		store:     store,
		proxyMgr:  NewProxyManager(cfg.ProxyTarget),
		hub:       NewHub(),
		static:    newStaticFileHandler(staticDir),
		staticDir: staticDir,
	}
	go s.watchConfigLoop()

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/dograin/", s.handleDograinStatic)
	mux.HandleFunc("/danmu/sub", s.handleDanmuSub)
	mux.HandleFunc("/api/config", s.handleAPIConfig)
	mux.HandleFunc("/config", s.handleConfigPage)
	mux.HandleFunc("/config/new-queue", s.handleCreateNewQueue)
	mux.HandleFunc("/config/add", s.handleAddQueueItem)
	mux.HandleFunc("/b", s.handleQueueBoard)

	log.Printf("服务启动: http://%s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, mux); err != nil {
		log.Fatalf("服务异常: %v", err)
	}
}

func resolveConfigPath() string {
	if path := strings.TrimSpace(os.Getenv("PDJ_CONFIG_PATH")); path != "" {
		return path
	}

	if _, err := os.Stat(configFileName); err == nil {
		return configFileName
	}

	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), configFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return configFileName
}

func resolveStaticDir() string {
	if path := strings.TrimSpace(os.Getenv("PDJ_STATIC_DIR")); path != "" {
		return path
	}

	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "frontend", "dograin")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return defaultStaticDir
}

func newStaticFileHandler(staticDir string) http.Handler {
	return http.FileServer(http.Dir(staticDir))
}

func DefaultConfig() AppConfig {
	return AppConfig{
		ListenAddr: defaultListenAddr,
		DataDir:    "data",
		QueueDir:   "data/queues",
		LogDir:     "data/logs",
		LogLevel:   "info",
		Panel: PanelConfig{
			RoomID: 0,
			UID:    0,
			Cookie: "",
		},
		MyJS: MyJSConfig{
			Admins:              []string{"迷糊的迷糊菇", "一纸轻予梦", "写一下你的名字哦"},
			BanAdmins:           []string{"黑名单成员", "黑名单成员2", "黑猫静止"},
			Jianzhang:           []string{""},
			PaiduiListLengthMax: 100,
			JianzhangCDKind:     1,
			JianzhangCDCishu:    1,
			LiwuChaduiKind:      50,
			LiwuPaiduiKind:      50,
		},
	}
}

func normalizeConfig(cfg AppConfig) AppConfig {
	def := DefaultConfig()
	if strings.TrimSpace(cfg.ListenAddr) == "" {
		cfg.ListenAddr = def.ListenAddr
	}
	if strings.TrimSpace(cfg.DataDir) == "" {
		cfg.DataDir = def.DataDir
	}
	if strings.TrimSpace(cfg.QueueDir) == "" {
		cfg.QueueDir = filepath.Join(cfg.DataDir, "queues")
	}
	if strings.TrimSpace(cfg.LogDir) == "" {
		cfg.LogDir = filepath.Join(cfg.DataDir, "logs")
	}
	cfg.LogLevel = normalizeLogLevel(cfg.LogLevel)
	cfg.ProxyTarget = strings.TrimSpace(cfg.ProxyTarget)
	return cfg
}

func parseSimpleYAML(b []byte) AppConfig {
	cfg := DefaultConfig()
	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		v = strings.Trim(v, `"'`)
		switch k {
		case "listen_addr":
			cfg.ListenAddr = v
		case "proxy_target":
			cfg.ProxyTarget = v
		case "data_dir":
			cfg.DataDir = v
		case "queue_dir":
			cfg.QueueDir = v
		case "log_dir":
			cfg.LogDir = v
		case "log_level":
			cfg.LogLevel = v
		case "roomid":
			if n, err := strconv.Atoi(v); err == nil {
				cfg.Panel.RoomID = n
			}
		case "uid":
			if n, err := strconv.Atoi(v); err == nil {
				cfg.Panel.UID = n
			}
		case "cookie":
			cfg.Panel.Cookie = v
		case "admins":
			cfg.MyJS.Admins = splitCSV(v)
		case "ban_admins":
			cfg.MyJS.BanAdmins = splitCSV(v)
		case "jianzhang":
			cfg.MyJS.Jianzhang = splitCSV(v)
		case "fankui":
			cfg.MyJS.Fankui = parseBool(v, cfg.MyJS.Fankui)
		case "guanli_fankui":
			cfg.MyJS.GuanliFankui = parseBool(v, cfg.MyJS.GuanliFankui)
		case "paidui_list_length_max":
			cfg.MyJS.PaiduiListLengthMax = parseInt(v, cfg.MyJS.PaiduiListLengthMax)
		case "jianzhangchadui":
			cfg.MyJS.Jianzhangchadui = parseBool(v, cfg.MyJS.Jianzhangchadui)
		case "jianzhang_cd_kind":
			cfg.MyJS.JianzhangCDKind = parseInt(v, cfg.MyJS.JianzhangCDKind)
		case "jianzhang_cd_cishu":
			cfg.MyJS.JianzhangCDCishu = parseInt(v, cfg.MyJS.JianzhangCDCishu)
		case "fangguan_can_doing":
			cfg.MyJS.FangguanCanDoing = parseBool(v, cfg.MyJS.FangguanCanDoing)
		case "all_suoyourenbukepaidui":
			cfg.MyJS.AllSuoyourenbukepaidui = parseBool(v, cfg.MyJS.AllSuoyourenbukepaidui)
		case "yhbot_kaiguan":
			cfg.MyJS.YHbotKaiguan = parseBool(v, cfg.MyJS.YHbotKaiguan)
		case "yhbotid":
			cfg.MyJS.YHbotID = v
		case "yhbot_msg_type":
			cfg.MyJS.YHbotMsgType = v
		case "yhbot_webhook_token":
			cfg.MyJS.YHbotWebhookToken = v
		case "ws_zbtool_kaiguan":
			cfg.MyJS.WsZbtoolKaiguan = parseBool(v, cfg.MyJS.WsZbtoolKaiguan)
		case "qywx_kaiguan":
			cfg.MyJS.QYWXKaiguan = parseBool(v, cfg.MyJS.QYWXKaiguan)
		case "wx_webhook":
			cfg.MyJS.WXWebhook = v
		case "only_myfuns_paidui":
			cfg.MyJS.OnlyMyfunsPaidui = parseBool(v, cfg.MyJS.OnlyMyfunsPaidui)
		case "liwu_chadui_kg":
			cfg.MyJS.LiwuChaduiKg = parseBool(v, cfg.MyJS.LiwuChaduiKg)
		case "liwu_paidui_kg":
			cfg.MyJS.LiwuPaiduiKg = parseBool(v, cfg.MyJS.LiwuPaiduiKg)
		case "liwu_chadui_kind":
			cfg.MyJS.LiwuChaduiKind = parseInt(v, cfg.MyJS.LiwuChaduiKind)
		case "liwu_paidui_kind":
			cfg.MyJS.LiwuPaiduiKind = parseInt(v, cfg.MyJS.LiwuPaiduiKind)
		}
	}
	return normalizeConfig(cfg)
}

func marshalSimpleYAML(cfg AppConfig) []byte {
	cfg = normalizeConfig(cfg)
	return []byte(fmt.Sprintf("listen_addr: %s\nproxy_target: %s\ndata_dir: %s\nqueue_dir: %s\nlog_dir: %s\nlog_level: %s\n", cfg.ListenAddr, cfg.ProxyTarget, cfg.DataDir, cfg.QueueDir, cfg.LogDir, cfg.LogLevel))
}

// 日志级别映射 / Log level mapping.
var logLevelPriority = map[string]int{
	"debug": 10,
	"info":  20,
	"warn":  30,
	"error": 40,
}

func normalizeLogLevel(raw string) string {
	level := strings.ToLower(strings.TrimSpace(raw))
	if _, ok := logLevelPriority[level]; ok {
		return level
	}
	return "info"
}

func shouldLog(level string, current string) bool {
	return logLevelPriority[level] >= logLevelPriority[current]
}

// 级别日志输出 / Leveled log output.
func logf(current, level, format string, args ...any) {
	if !shouldLog(level, current) {
		return
	}
	log.Printf("["+strings.ToUpper(level)+"] "+format, args...)
}

func setupRunLogger(configLevel string) error {
	level := normalizeLogLevel(configLevel)
	if err := os.MkdirAll(runLogFolder, 0o755); err != nil {
		return err
	}
	if err := cleanupExpiredRunLogs(runLogFolder, 30*24*time.Hour); err != nil {
		return err
	}
	logFile := filepath.Join(runLogFolder, "pdj-"+time.Now().Format(runLogNameLayout)+".log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	log.SetOutput(io.MultiWriter(os.Stdout, f))
	log.Printf("[INFO] 运行日志已启用，等级=%s, 文件=%s", level, logFile)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetPrefix("")
	activeRunLogFilter = &levelFilterWriter{
		level:  level,
		writer: io.MultiWriter(os.Stdout, f),
	}
	log.SetOutput(activeRunLogFilter)
	return nil
}

type levelFilterWriter struct {
	level  string
	writer io.Writer
}

func (w *levelFilterWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	matchedLevel := ""
	for level := range logLevelPriority {
		tag := "[" + strings.ToUpper(level) + "]"
		if strings.Contains(msg, tag) {
			matchedLevel = level
			break
		}
	}
	if matchedLevel == "" {
		matchedLevel = "info"
	}
	if !shouldLog(matchedLevel, w.level) {
		return len(p), nil
	}
	return w.writer.Write(p)
}

func cleanupExpiredRunLogs(dir string, maxAge time.Duration) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	expireAt := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "pdj-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(expireAt) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	return nil
	return []byte(fmt.Sprintf("listen_addr: %s\nproxy_target: %s\ndata_dir: %s\nqueue_dir: %s\nlog_dir: %s\nroomid: %d\nuid: %d\ncookie: %s\nadmins: %s\nban_admins: %s\njianzhang: %s\nfankui: %t\nguanli_fankui: %t\npaidui_list_length_max: %d\njianzhangchadui: %t\njianzhang_cd_kind: %d\njianzhang_cd_cishu: %d\nfangguan_can_doing: %t\nall_suoyourenbukepaidui: %t\nyhbot_kaiguan: %t\nyhbotid: %s\nyhbot_msg_type: %s\nyhbot_webhook_token: %s\nws_zbtool_kaiguan: %t\nqywx_kaiguan: %t\nwx_webhook: %s\nonly_myfuns_paidui: %t\nliwu_chadui_kg: %t\nliwu_paidui_kg: %t\nliwu_chadui_kind: %d\nliwu_paidui_kind: %d\n",
		cfg.ListenAddr, cfg.ProxyTarget, cfg.DataDir, cfg.QueueDir, cfg.LogDir,
		cfg.Panel.RoomID, cfg.Panel.UID, cfg.Panel.Cookie,
		joinCSV(cfg.MyJS.Admins), joinCSV(cfg.MyJS.BanAdmins), joinCSV(cfg.MyJS.Jianzhang),
		cfg.MyJS.Fankui, cfg.MyJS.GuanliFankui, cfg.MyJS.PaiduiListLengthMax,
		cfg.MyJS.Jianzhangchadui, cfg.MyJS.JianzhangCDKind, cfg.MyJS.JianzhangCDCishu,
		cfg.MyJS.FangguanCanDoing, cfg.MyJS.AllSuoyourenbukepaidui, cfg.MyJS.YHbotKaiguan,
		cfg.MyJS.YHbotID, cfg.MyJS.YHbotMsgType, cfg.MyJS.YHbotWebhookToken,
		cfg.MyJS.WsZbtoolKaiguan, cfg.MyJS.QYWXKaiguan, cfg.MyJS.WXWebhook,
		cfg.MyJS.OnlyMyfunsPaidui, cfg.MyJS.LiwuChaduiKg, cfg.MyJS.LiwuPaiduiKg,
		cfg.MyJS.LiwuChaduiKind, cfg.MyJS.LiwuPaiduiKind))
}

func parseBool(val string, fallback bool) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(val))
	if err != nil {
		return fallback
	}
	return b
}

func parseInt(val string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		return fallback
	}
	return n
}

func splitCSV(val string) []string {
	parts := strings.Split(val, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		result = append(result, p)
	}
	return result
}

func joinCSV(items []string) string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		it := strings.TrimSpace(item)
		if it == "" {
			continue
		}
		result = append(result, it)
	}
	return strings.Join(result, ",")
}

func NewConfigManager(path string) (*ConfigManager, error) {
	m := &ConfigManager{path: path}
	if err := m.loadOrInit(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *ConfigManager) loadOrInit() error {
	st, err := os.Stat(m.path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.WriteFile(m.path, marshalSimpleYAML(DefaultConfig()), 0o644); err != nil {
			return err
		}
		st, err = os.Stat(m.path)
		if err != nil {
			return err
		}
	}
	b, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.cfg = parseSimpleYAML(b)
	m.modTime = st.ModTime()
	m.size = st.Size()
	m.mu.Unlock()
	return nil
}

func (m *ConfigManager) ReloadIfChanged() (AppConfig, bool, error) {
	st, err := os.Stat(m.path)
	if err != nil {
		return AppConfig{}, false, err
	}
	m.mu.RLock()
	unchanged := st.ModTime().Equal(m.modTime) && st.Size() == m.size
	m.mu.RUnlock()
	if unchanged {
		return AppConfig{}, false, nil
	}
	b, err := os.ReadFile(m.path)
	if err != nil {
		return AppConfig{}, false, err
	}
	cfg := parseSimpleYAML(b)
	m.mu.Lock()
	m.cfg = cfg
	m.modTime = st.ModTime()
	m.size = st.Size()
	m.mu.Unlock()
	return cfg, true, nil
}

func (m *ConfigManager) Get() AppConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

func NewQueueStore(cfg AppConfig) (*QueueStore, error) {
	s := &QueueStore{monthExpr: regexp.MustCompile(`^queue_(\d{4})_(\d{2})\.csv$`)}
	if err := s.UpdatePaths(cfg); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *QueueStore) UpdatePaths(cfg AppConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queueDir = cfg.QueueDir
	s.logsDir = cfg.LogDir
	s.current = filepath.Join(cfg.QueueDir, "current_queue.csv")
	s.logFile = filepath.Join(cfg.LogDir, "queue.log")
	return s.ensureFilesLocked()
}

func (s *QueueStore) ensureFilesLocked() error {
	if err := os.MkdirAll(s.queueDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(s.logsDir, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(s.current); err != nil {
		if os.IsNotExist(err) {
			if err := writeCSVAtomic(s.current, nil); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	f, err := os.OpenFile(s.logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}

func (s *QueueStore) readCurrentLocked() ([]QueueItem, error) {
	f, err := os.Open(s.current)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	recs, err := r.ReadAll()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	items := make([]QueueItem, 0, len(recs))
	for _, rec := range recs {
		if len(rec) < 4 {
			continue
		}
		seq, err := strconv.Atoi(rec[0])
		if err != nil {
			continue
		}
		tm, err := time.ParseInLocation(dateTimeLayout, rec[3], time.Local)
		if err != nil {
			tm = time.Now()
		}
		items = append(items, QueueItem{Seq: seq, ID: rec[1], Remark: rec[2], CreatedAt: tm})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Seq < items[j].Seq })
	return items, nil
}

func writeCSVAtomic(path string, items []QueueItem) error {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	for _, item := range items {
		if err := w.Write([]string{strconv.Itoa(item.Seq), item.ID, item.Remark, item.CreatedAt.Format(dateTimeLayout)}); err != nil {
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func appendItemsToCSV(path string, items []QueueItem) error {
	if len(items) == 0 {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	for _, item := range items {
		if err := w.Write([]string{strconv.Itoa(item.Seq), item.ID, item.Remark, item.CreatedAt.Format(dateTimeLayout)}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func (s *QueueStore) appendLogLocked(msg string) {
	f, err := os.OpenFile(s.logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("日志写入失败: %v", err)
		return
	}
	defer f.Close()
	_, _ = f.WriteString(fmt.Sprintf("%s %s\n", time.Now().Format(dateTimeLayout), msg))
}

func (s *QueueStore) ListCurrent() ([]QueueItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readCurrentLocked()
}

func (s *QueueStore) Add(id, remark string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.readCurrentLocked()
	if err != nil {
		return err
	}
	items = append(items, QueueItem{Seq: len(items) + 1, ID: id, Remark: remark, CreatedAt: time.Now().Truncate(time.Second)})
	if err := writeCSVAtomic(s.current, items); err != nil {
		return err
	}
	s.appendLogLocked("ADD id=" + id)
	return nil
}

func (s *QueueStore) CreateNewQueue() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.readCurrentLocked()
	if err != nil {
		return err
	}
	if len(items) > 0 {
		monthFile := filepath.Join(s.queueDir, "queue_"+time.Now().Format(monthFileLayout)+".csv")
		if err := appendItemsToCSV(monthFile, items); err != nil {
			return err
		}
	}
	if err := writeCSVAtomic(s.current, nil); err != nil {
		return err
	}
	s.appendLogLocked(fmt.Sprintf("ROTATE merged=%d", len(items)))
	return s.cleanupOldMonthlyLocked(6)
}

func (s *QueueStore) cleanupOldMonthlyLocked(months int) error {
	entries, err := os.ReadDir(s.queueDir)
	if err != nil {
		return err
	}
	threshold := time.Now().AddDate(0, -months, 0)
	boundary := time.Date(threshold.Year(), threshold.Month(), 1, 0, 0, 0, 0, time.Local)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := s.monthExpr.FindStringSubmatch(e.Name())
		if len(m) != 3 {
			continue
		}
		y, _ := strconv.Atoi(m[1])
		mo, _ := strconv.Atoi(m[2])
		fm := time.Date(y, time.Month(mo), 1, 0, 0, 0, 0, time.Local)
		if fm.Before(boundary) {
			_ = os.Remove(filepath.Join(s.queueDir, e.Name()))
			s.appendLogLocked("CLEAN old=" + e.Name())
		}
	}
	return nil
}

func NewProxyManager(target string) *ProxyManager {
	p := &ProxyManager{}
	p.Update(target)
	return p
}

func (p *ProxyManager) Update(target string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	target = strings.TrimSpace(target)
	if target == "" {
		p.proxy = nil
		return
	}
	u, err := url.Parse(target)
	if err != nil || u.Scheme == "" || u.Host == "" {
		log.Printf("proxy_target 非法，已忽略: %q", target)
		p.proxy = nil
		return
	}
	p.proxy = httputil.NewSingleHostReverseProxy(u)
	p.proxy.ErrorLog = log.New(os.Stderr, "[proxy] ", log.LstdFlags)
}

func (p *ProxyManager) ServeHTTP(w http.ResponseWriter, r *http.Request) bool {
	p.mu.RLock()
	proxy := p.proxy
	p.mu.RUnlock()
	if proxy == nil {
		return false
	}
	proxy.ServeHTTP(w, r)
	return true
}

func NewHub() *Hub {
	return &Hub{clients: map[*wsConn]struct{}{}}
}

func (h *Hub) Add(c *wsConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}
func (h *Hub) Remove(c *wsConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
}
func (h *Hub) Broadcast(msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		if err := c.WriteText([]byte(msg)); err != nil {
			_ = c.Close()
			delete(h.clients, c)
		}
	}
}

func (s *Server) watchConfigLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		cfg, changed, err := s.cfgMgr.ReloadIfChanged()
		if err != nil {
			log.Printf("热更新失败: %v", err)
			continue
		}
		if changed {
			s.proxyMgr.Update(cfg.ProxyTarget)
			if err := s.store.UpdatePaths(cfg); err != nil {
				log.Printf("更新路径失败: %v", err)
			}
			if activeRunLogFilter != nil {
				activeRunLogFilter.level = normalizeLogLevel(cfg.LogLevel)
			}
		}
	}
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/danmu/sub" {
		s.handleWS(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/config") || strings.HasPrefix(r.URL.Path, "/b") {
		http.NotFound(w, r)
		return
	}
	s.static.ServeHTTP(w, r)
}

func (s *Server) handleDograinStatic(w http.ResponseWriter, r *http.Request) {
	orig := r.URL.Path
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/dograin")
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}
	s.static.ServeHTTP(w, r)
	r.URL.Path = orig
}

func (s *Server) handleDanmuSub(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/danmu/sub" {
		http.NotFound(w, r)
		return
	}
	s.handleWS(w, r)
}

type apiConfigPayload struct {
	RoomID int        `json:"roomid"`
	UID    int        `json:"uid"`
	Cookie string     `json:"cookie"`
	MyJS   MyJSConfig `json:"myjs"`
}

func (s *Server) handleAPIConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg := s.cfgMgr.Get()
		payload := apiConfigPayload{
			RoomID: cfg.Panel.RoomID,
			UID:    cfg.Panel.UID,
			Cookie: cfg.Panel.Cookie,
			MyJS:   cfg.MyJS,
		}
		b, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(b)
	case http.MethodPost:
		var payload apiConfigPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		cfg := s.cfgMgr.Get()
		cfg.Panel.RoomID = payload.RoomID
		cfg.Panel.UID = payload.UID
		cfg.Panel.Cookie = payload.Cookie
		if !isMyJSConfigZero(payload.MyJS) {
			cfg.MyJS = payload.MyJS
		}
		if err := os.WriteFile(s.cfgMgr.path, marshalSimpleYAML(cfg), 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _, _ = s.cfgMgr.ReloadIfChanged()
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func isMyJSConfigZero(cfg MyJSConfig) bool {
	return len(cfg.Admins) == 0 &&
		len(cfg.BanAdmins) == 0 &&
		len(cfg.Jianzhang) == 0 &&
		!cfg.Fankui &&
		!cfg.GuanliFankui &&
		cfg.PaiduiListLengthMax == 0 &&
		!cfg.Jianzhangchadui &&
		cfg.JianzhangCDKind == 0 &&
		cfg.JianzhangCDCishu == 0 &&
		!cfg.FangguanCanDoing &&
		!cfg.AllSuoyourenbukepaidui &&
		!cfg.YHbotKaiguan &&
		cfg.YHbotID == "" &&
		cfg.YHbotMsgType == "" &&
		cfg.YHbotWebhookToken == "" &&
		!cfg.WsZbtoolKaiguan &&
		!cfg.QYWXKaiguan &&
		cfg.WXWebhook == "" &&
		!cfg.OnlyMyfunsPaidui &&
		!cfg.LiwuChaduiKg &&
		!cfg.LiwuPaiduiKg &&
		cfg.LiwuChaduiKind == 0 &&
		cfg.LiwuPaiduiKind == 0
}

func (s *Server) handleConfigPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := s.cfgMgr.Get()
	items, _ := s.store.ListCurrent()
	_ = configPageTpl.Execute(w, map[string]any{"Config": cfg, "Count": len(items)})
}

func (s *Server) handleAddQueueItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	remark := strings.TrimSpace(r.FormValue("remark"))
	if id == "" {
		http.Error(w, "id 不能为空", http.StatusBadRequest)
		return
	}
	if err := s.store.Add(id, remark); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.hub.Broadcast("queue_updated")
	http.Redirect(w, r, "/config", http.StatusSeeOther)
}

func (s *Server) handleCreateNewQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.store.CreateNewQueue(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.hub.Broadcast("queue_rotated")
	http.Redirect(w, r, "/config", http.StatusSeeOther)
}

func (s *Server) handleQueueBoard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	items, err := s.store.ListCurrent()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = boardPageTpl.Execute(w, map[string]any{"Items": items})
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") && strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

func computeAcceptKey(key string) string {
	h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h[:])
}

func serverUpgrade(w http.ResponseWriter, req *http.Request) (*wsConn, error) {
	key := strings.TrimSpace(req.Header.Get("Sec-WebSocket-Key"))
	if key == "" {
		return nil, errors.New("missing websocket key")
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("hijacking unsupported")
	}
	c, buf, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + computeAcceptKey(key) + "\r\n\r\n"
	if _, err := buf.WriteString(resp); err != nil {
		_ = c.Close()
		return nil, err
	}
	if err := buf.Flush(); err != nil {
		_ = c.Close()
		return nil, err
	}
	return &wsConn{conn: c, rd: bufio.NewReader(c)}, nil
}

func (w *wsConn) Close() error {
	w.closeOnce.Do(func() {
		w.closed = true
		_ = w.conn.Close()
	})
	return nil
}
func (w *wsConn) WriteText(data []byte) error { return w.writeFrame(0x1, data) }
func (w *wsConn) writeFrame(opcode byte, payload []byte) error {
	w.wmu.Lock()
	defer w.wmu.Unlock()
	if w.closed {
		return io.EOF
	}
	head := []byte{0x80 | (opcode & 0x0F)}
	plen := len(payload)
	switch {
	case plen <= 125:
		head = append(head, byte(plen))
	case plen <= math.MaxUint16:
		head = append(head, 126, byte(plen>>8), byte(plen))
	default:
		head = append(head, 127)
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(plen))
		head = append(head, b...)
	}
	if _, err := w.conn.Write(head); err != nil {
		return err
	}
	_, err := w.conn.Write(payload)
	return err
}
func (w *wsConn) ReadFrame() (byte, []byte, error) {
	f, err := w.rd.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	s, err := w.rd.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	op := f & 0x0F
	masked := s&0x80 != 0
	plen := int(s & 0x7F)
	if plen == 126 {
		b := make([]byte, 2)
		if _, err := io.ReadFull(w.rd, b); err != nil {
			return 0, nil, err
		}
		plen = int(binary.BigEndian.Uint16(b))
	} else if plen == 127 {
		b := make([]byte, 8)
		if _, err := io.ReadFull(w.rd, b); err != nil {
			return 0, nil, err
		}
		plen = int(binary.BigEndian.Uint64(b))
	}
	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(w.rd, mask[:]); err != nil {
			return 0, nil, err
		}
	}
	payload := make([]byte, plen)
	if _, err := io.ReadFull(w.rd, payload); err != nil {
		return 0, nil, err
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return op, payload, nil
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	c, err := serverUpgrade(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.hub.Add(c)
	defer func() {
		s.hub.Remove(c)
		_ = c.Close()
	}()
	_ = c.WriteText([]byte("connected"))
	for {
		op, payload, err := c.ReadFrame()
		if err != nil {
			return
		}
		switch op {
		case 0x8:
			return
		case 0x1:
			s.hub.Broadcast(string(payload))
		}
	}
}

var configPageTpl = template.Must(template.New("cfg").Parse(`<!doctype html><html lang="zh-CN"><head><meta charset="UTF-8"><title>配置管理</title><style>body{font-family:Arial;margin:24px;max-width:880px}.card{border:1px solid #ddd;border-radius:8px;padding:16px;margin-bottom:12px}input,button{padding:8px;margin:4px}</style></head><body><h1>配置管理页面</h1><div class="card"><p><b>proxy_target:</b> {{.Config.ProxyTarget}}</p><p><b>queue_dir:</b> {{.Config.QueueDir}}</p><p><b>当前队列数量:</b> {{.Count}}</p><form method="post" action="/config/new-queue"><button type="submit">创建新队列</button></form></div><div class="card"><h3>新增队列项</h3><form method="post" action="/config/add"><input name="id" placeholder="id" required><input name="remark" placeholder="备注"><button type="submit">保存</button></form></div><p><a href="/b">查看展示页</a></p></body></html>`))

var boardPageTpl = template.Must(template.New("b").Parse(`<!doctype html><html lang="zh-CN"><head><meta charset="UTF-8"><title>队列展示</title><style>body{font-family:Arial;margin:24px}table{border-collapse:collapse;width:100%;max-width:860px}th,td{border:1px solid #ccc;padding:8px}th{background:#f5f5f5}</style></head><body><h1>当前队列</h1><table><thead><tr><th>序号</th><th>id</th><th>备注</th></tr></thead><tbody>{{range .Items}}<tr><td>{{.Seq}}</td><td>{{.ID}}</td><td>{{.Remark}}</td></tr>{{else}}<tr><td colspan="3">暂无数据</td></tr>{{end}}</tbody></table></body></html>`))
