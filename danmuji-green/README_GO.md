# danmuji-green（Go + JavaScript 替代版）

该目录新增了基于 **Go + JavaScript** 的本地弹幕流服务，用于替代原先的 Java Jar 运行方式。

## 启动

1. 安装 Go 1.22+
2. 双击 `run.bat`（首次会自动 `go build`）
3. 打开浏览器：
   - 弹幕流与网页：`http://127.0.0.1:23333`
   - 配置 API：`http://127.0.0.1:23334/api/config`

## 接口兼容

- WebSocket：`ws://127.0.0.1:23333/danmu/sub`
- HTTP 配置：`GET/POST http://127.0.0.1:23334/api/config`
- 配置文件：`../pdj/dograin/pdj_config.json`

## 参考逻辑

弹幕协议封包/解包、心跳与鉴权流程参考：
`https://github.com/BanqiJane/Bilibili_Danmuji`
