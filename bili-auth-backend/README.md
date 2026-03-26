# Bilibili QR Login Backend (Go + Gin)

> ⚠️ 仅用于“用户自己扫码授权登录自己的账号”场景。禁止用于绕过验证、批量滥用、钓鱼或伪造登录。

本项目实现一个独立后端服务：
- 向 Bilibili 申请二维码登录会话；
- 返回二维码内容与二维码图片；
- 轮询扫码状态；
- 登录成功时从 **轮询响应头 `Set-Cookie`** 提取 cookie；
- 保存 cookie map / cookie string 供后续业务使用。

**本后端获取 bilibili cookie 的方式，是在用户扫码并确认后，从轮询响应头的 Set-Cookie 中提取，而不是从二维码内容本身解析 cookie。**

## 技术栈

- Go 1.22+
- Gin
- net/http
- 内存会话存储（预留 Redis Store 接口）
- slog JSON 结构化日志

## 目录结构

```text
bili-auth-backend/
├── cmd/server/main.go
├── internal/handler/auth_handler.go
├── internal/httpclient/bilibili_auth_client.go
├── internal/model/config.go
├── internal/model/session.go
├── internal/service/auth/service.go
├── internal/store/memory_store.go
├── internal/store/redis_store.go
├── internal/store/store.go
├── internal/utils/cookiejar.go
├── internal/utils/logger.go
├── internal/utils/mask.go
├── internal/utils/response.go
├── internal/service/auth/service_test.go
├── internal/store/memory_store_test.go
├── internal/utils/cookiejar_test.go
├── .env.example
├── go.mod
└── README.md
```

## 启动

```bash
cd bili-auth-backend
cp .env.example .env
go mod tidy
go run ./cmd/server
```

默认监听：`http://localhost:8080`

## API

### 1) 创建登录会话

```bash
curl -X POST http://localhost:8080/api/auth/qrcode/start
```

### 2) 获取二维码图片（PNG）

```bash
curl -L "http://localhost:8080/api/auth/qrcode/image/<session_id>" --output qrcode.png
```

### 3) 轮询扫码状态

```bash
curl "http://localhost:8080/api/auth/qrcode/poll/<session_id>"
```

状态映射：
- `waiting_scan`：未扫码
- `waiting_confirm`：已扫码未确认
- `confirmed`：已确认（尝试提取 cookie）
- `expired`：二维码过期
- `failed`：其他错误

### 4) 查询会话状态

```bash
curl "http://localhost:8080/api/auth/session/<session_id>"
```

默认不会返回完整 cookie；只有 `DEBUG=true` 时返回 `cookie_string` 与 `cookie_map`。

### 5) 登出并清理本地会话

```bash
curl -X POST "http://localhost:8080/api/auth/logout/<session_id>"
```

## 单元测试

```bash
go test ./...
```

覆盖点：
- `Set-Cookie` 解析
- Cookie 合并/规范化
- 登录状态迁移映射
- 会话过期清理

## 关键说明

1. 关键 cookie 校验：
   - `SESSDATA`
   - `bili_jct`
   - `DedeUserID`
   - `DedeUserID__ckMd5`
   - `sid`

2. 补全能力：
   - 预留 `EnrichCookies(ctx, session)` 插件点；
   - 可后续补 `buvid3`/`buvid4`/`bili_ticket`；
   - 若不完整，会在状态中给出提示。

3. 安全：
   - 日志脱敏，不打印完整 cookie。
   - 默认 API 不回传完整 cookie。
