@echo off
setlocal
cd /d %~dp0

if not exist danmu_server.exe (
  echo [INFO] 未检测到 danmu_server.exe，开始构建...
  go build -o danmu_server.exe .
  if errorlevel 1 (
    echo [ERROR] Go 构建失败，请确认已安装 Go 1.22+
    pause
    exit /b 1
  )
)

title PDJ-Go-Danmu-%date%-%time%-%cd%
danmu_server.exe
