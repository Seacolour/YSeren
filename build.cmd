@echo off
setlocal

REM 一键构建入口（双击即可）
REM 等价于：powershell -ExecutionPolicy Bypass -File .\build.ps1

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0build.ps1" %*
exit /b %errorlevel%

