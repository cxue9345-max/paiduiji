# paiduiji

支持抖音（预备）、Bilibili 平台的弹幕排队姬。

## 教学内容（仅保留这一份）

### 交流群
- QQ 群：`932531758`

### 观看入口
- 一纸轻予梦 → 主页 → 排队姬视频＆安装教程合辑

### 基础功能（不变）
# paiduiji 教学整合文档

支持抖音（预备）、Bilibili 平台的弹幕排队姬。

> 你提到“把所有教学内容整合到 README.md”，本文件已把仓库内可读取的教程/使用说明集中整理到一个地方，便于直接查阅。

---

## 1. 教程入口与交流群

- QQ 交流群：`932531758`
- 观看途径：
  - 进入「一纸轻予梦」主页
  - 查看「排队姬视频＆安装教程合辑」
- 如果找不到解决办法：
  - 可私信
  - 可在群里提问（更容易及时看到）
  - 直播间有时挂游戏，可能无法及时回复

---

## 2. 基础功能（长期不变）

视频版本可能有差异，但以下基础指令/功能不变：

- 排队
- 排队 + 内容
- 取消排队
- 删除：`del + 数字` / `删除 + 数字`
- 添加：`add + 内容` / `添加 + 内容`

### 教学视频
- 演示视频：
  - https://www.bilibili.com/video/BV1De411a7np/?spm_id_from=333.999.0.0&vd_source=6029afe07c3a593114d348ef03e9291b
- 安装教程：
  - https://www.bilibili.com/video/BV1wV411Q7sc/?spm_id_from=333.788&vd_source=6029afe07c3a593114d348ef03e9291b

### 安装步骤
1. 下载整合包。
2. 打开 `run.bat` 和 `nginx.exe`（路径不能有中文）。
3. 浏览器打开 `xxxx:23333` 登录，填写直播间号并连接。
4. 添加浏览器插件。
5. 自测（无需开播）。
6. 可按需修改 `index.html` 调整样式。
---

## 3. 安装教程（核心步骤）

安装教程视频：
- https://www.bilibili.com/video/BV1wV411Q7sc/?spm_id_from=333.788&vd_source=6029afe07c3a593114d348ef03e9291b

演示视频：
- https://www.bilibili.com/video/BV1De411a7np/?spm_id_from=333.999.0.0&vd_source=6029afe07c3a593114d348ef03e9291b

安装步骤：

1. 下载整合包。
2. 打开 `run.bat` 和 `nginx.exe`。
   - 有些压缩包已附带 bat 脚本，直接运行即可。
   - **注意：路径不能有中文。**
3. 浏览器会弹出类似 `xxxx:23333` 的网页，登录。
   - 登录后就不会有 `***`。
   - 填写直播间号并连接。
4. 添加浏览器插件。
5. 先自行测试（无需开播）。
6. 觉得好用再继续使用；可修改 `index.html` 调整样式。

---

## 4. 常用地址（本地）

- 主页面（脚本会自动打开）：
  - http://localhost:23333/
- 添加直播源：
  - http://localhost:9816/

群信息（原文）：
- 安装交流群：932531758
- 提意见群：932531758
- BUG 提交群：932531758

---

## 5. myjs 基础配置（基础功能）

> 来自“myjs 基本配置填写方法(基础功能)”文档要点。

### 5.1 插件管理设置

- 把“写一下你的名字”改成你的 B 站 ID。
- 说明：`2024-04-29` 版本及以后通常无需调整。

### 5.2 黑名单成员

- 黑名单含义：对方可在直播间说话，但不能参与排队。
- 重要限制：
  - 由于插件缓存机制，**必须在开播前**添加黑名单。
  - 如果开播后修改黑名单，可能导致当前排队信息被清空。

### 5.3 舰长插队（预设中）

- 当前说明为“预设中，未实装”。
- 可先了解：
  - 若舰长在职，通常无需填写。
  - 该配置用于主播自定义“在职/过期舰长插队权限”策略。

---

## 6. myjs 进阶功能（可选）

> 原文提示：**“能用再试试新功能”**。基础功能稳定后再配置。

### 6.1 云湖通知器（可选）

- 不使用可留空。
- 作用：排队列表每次更新会触发一次推送。
- 云湖开放文档：
  - https://www.yhchat.com/document/400-410

示例变量：

```js
var YHbotId = ""; // 群号或个人ID；群需先申请机器人并拉群
var YHbot_msg_type = ""; // 推送到群填 group，个人填 user
var YHbot_webhook_token = ""; // 机器人的 webhook token
```

说明：webhook 形式示例（原文）
- `https://chat-go.jwzhd.com/open-apis/v1/bot/send?token=xxxxx`

### 6.2 企业微信通知器（可选）

- 填 webhook 完整地址。
- 原文备注：该功能“好像坏了”。

```js
var WX_webhook = ""; // https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxxxx
```

---

## 7. myjs 修改器使用方法

说明：

1. 当前处于测试阶段，出现 bug 属正常现象。
2. 界面不强调美观，以可用为主。

操作流程：

1. 点击“选择 myjs 查找”。
2. 找不到则手动选择 myjs 文件。
3. 调整人员权限（多个值用英文逗号分隔）。
4. 点击“写入”。
5. 右上角关闭即可。

注意：
- **下播后再修改**，直播中修改通常不会生效。

---

## 8. 样式替换教程（index.html）

核心说明：

- `myjs.js` 和 `index.html` 同目录的文件夹是主要工作区。
- 在样式目录中选择喜欢的 `xxx.html`。
- 把该文件改名为 `index.html`。
- 替换上级目录原有 `index.html` 即可。

补充：

- 直接替换后颜色不一定完全符合预期，可继续微调。
- 参考视频：
  - https://www.bilibili.com/video/BV17Z4y1Y7Rc/?spm_id_from=333.788&vd_source=6029afe07c3a593114d348ef03e9291b

---

## 9. 自建后端（backend）运行说明

当前版本为自建后端。

### 9.1 Windows 运行

- 直接运行 `run.bat`。
- 记得允许网络权限。

### 9.2 其他方式运行

- 解压后在当前目录打开控制台，执行：

```bash
go run ./backend
```

- 默认端口：WebSocket `23333`、配置 API `23334`。
- 如需改端口，请修改 `backend/main.go` 中 `localWSAddr` / `localAPIAddr`。

### 9.3 配置与退出

- 浏览器打开：`http://127.0.0.1:23333`
- 退出方式：直接关闭命令行窗口。

### 9.4 更新与反馈

- 后端逻辑参考：https://github.com/BanqiJane/Bilibili_Danmuji/
- 当前仓库使用自建 Go 后端，不再依赖 danmuji-green 绿色包。

---

## 10. 插件更新/扩展视频合集

- 免费插件更新（2024-04-13：黑名单 + 排队成功提示）：
  - https://www.bilibili.com/video/BV1gt421E7nm/?spm_id_from=333.788&vd_source=6029afe07c3a593114d348ef03e9291b
- 更新排队姬颜色文件 index.html：
  - https://www.bilibili.com/video/BV1KK421a7zy/?spm_id_from=333.788&vd_source=6029afe07c3a593114d348ef03e9291b
  - https://www.bilibili.com/video/BV1cx42117AW/?spm_id_from=333.788&vd_source=6029afe07c3a593114d348ef03e9291b
- 推送功能（原文：暂不建议使用）：
  - https://www.bilibili.com/video/BV1TD421L7pr/?spm_id_from=333.788&vd_source=6029afe07c3a593114d348ef03e9291b
- 弹幕姬更新办法（“大黑框”）：
  - https://www.bilibili.com/video/BV1jF4m1c7xd/?spm_id_from=333.788&vd_source=6029afe07c3a593114d348ef03e9291b

---

## 11. 其他补充说明

- `WebCT` 目录中的源码说明原文：
  - “这是建立一个 ws 服务器，然后被响应，然后才可以的。”
- `新的myjs替换掉旧的myjs即可完成更新.txt` 与 `改成index点html替换源文件即可.txt` 当前文件内容为空（文件名即说明）。

---

## 12. 教学内容来源（已整合）

- `教程.docx`
- `教程视频链接.txt`
- `myjs修改器使用方法.txt`
- `安装方法以及更新记录/教程视频链接.txt`
- `安装方法以及更新记录/myjs基本配置填写方法(基础功能).docx`
- `安装方法以及更新记录/myjs进阶功能填写方法(只用基础功能不用看).txt`
- `安装方法以及更新记录/两个网址.txt`
- `安装方法以及更新记录/新的myjs替换掉旧的myjs即可完成更新.txt`
- `pdj/dograin/排队姬样式常考/替换教程.txt`
- `pdj/dograin/排队姬样式常考/改成index点html替换源文件即可.txt`
- `backend/main.go`
- `WebCT/源码说明.txt`

