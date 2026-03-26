@echo off
setlocal

cd /d %~dp0

echo [PDJ] Building backend...
go build -o pdj-backend.exe .
if errorlevel 1 (
  echo [PDJ] Build failed.
  pause
  exit /b 1
)

echo [PDJ] Starting backend...
pdj-backend.exe
