# 打包前最终检测报告（2026-03-26）

## 检测范围
- 后端：`services/backend`
- 后端：`bili-auth-backend`
- 后端：`services/bili-auth-backend`
- 前端：`frontend/vue-e`

## 检测命令与结果

1. `go test ./...`（`services/backend`）
   - 结果：通过
   - 输出：`?    paiduiji/backend    [no test files]`

2. `go test ./...`（`bili-auth-backend`）
   - 结果：失败
   - 原因：缺少 `go.sum` 条目，涉及 `gin`、`uuid`、`qrcode`、`godotenv` 等依赖。

3. `go test ./...`（`services/bili-auth-backend`）
   - 结果：失败
   - 原因：同上，缺少 `go.sum` 条目。

4. `go mod tidy`（`bili-auth-backend` 和 `services/bili-auth-backend`）
   - 结果：失败
   - 原因：当前环境访问 Go 代理 `https://proxy.golang.org` 返回 `403 Forbidden`，依赖无法下载。

5. `timeout 60s npm ci`（`frontend/vue-e`）
   - 结果：失败（超时退出码 124）
   - 原因：当前环境访问 npm registry 大量返回 `403 Forbidden`，依赖元数据无法获取，安装过程无法完成。

## 本次补充排查（拉取 Go 依赖失败）

### 现象复现
1. 在 `bili-auth-backend` 执行 `go mod tidy`：
   - `GOPROXY` 默认值为 `https://proxy.golang.org,direct`。
   - 下载多个模块时全部报错：`Get "https://proxy.golang.org/...": Forbidden`。

2. 直接探测代理站点：
   - 执行 `curl -I https://proxy.golang.org/github.com/gin-contrib/sse/@v/v0.1.0.zip`。
   - 返回 `CONNECT tunnel failed, response 403`，并伴随 `HTTP/1.1 403 Forbidden`。

3. 检查环境代理变量：
   - 存在 `HTTP_PROXY=http://proxy:8080`、`HTTPS_PROXY=http://proxy:8080`（及小写同名变量）。

4. 尝试绕过 Go 模块代理：
   - 执行 `GOPROXY=direct GOSUMDB=off go mod tidy`。
   - 改为直接访问 `github.com` / `golang.org`，依然报错：
     - `fatal: unable to access 'https://github.com/...': CONNECT tunnel failed, response 403`
     - `https fetch: Get "https://golang.org/x/net?go-get=1": Forbidden`

### 根因结论
- **根因不是 go.mod 或 go.sum 配置错误**，而是当前运行环境的上游 HTTP(S) 代理（`proxy:8080`）拒绝了对 Go 依赖源的 CONNECT/HTTPS 请求。
- 因为代理层返回 `403 Forbidden`，导致无论走 `proxy.golang.org` 还是 `direct`，都无法拉取外部依赖。

### 建议处理
1. 在可访问外网依赖源（至少 `proxy.golang.org`、`github.com`、`golang.org`）的网络环境执行：
   - `go mod tidy`
   - `go test ./...`
2. 若必须经过企业代理，请将上述域名加入代理放行白名单（CONNECT 允许名单）。
3. 若公司内部有私有 Go 模块镜像，建议统一设置：
   - `GOPROXY=<内网镜像>,direct`
   - 并确保镜像可访问 `sumdb` 或配套内网校验策略。

## 打包前结论
- 当前环境下仅 `services/backend` 可完成基础测试。
- 鉴于依赖仓库访问受限，前端与认证后端无法完成完整构建链路验证。
- 建议在可访问 `proxy.golang.org` 与 `registry.npmjs.org` 的网络环境中重新执行：
  - `go mod tidy && go test ./...`
  - `npm ci && npm run lint && npm run build`
