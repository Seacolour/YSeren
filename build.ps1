Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

param(
  # 输出 exe 名称（默认与 README 一致）
  [string]$Out = "yseren.exe",
  # 跳过前端构建（仅用于排查）
  [switch]$NoFrontend,
  # 跳过依赖安装（如果你确信 node_modules 已经是最新的）
  [switch]$NoNpmInstall
)

# 确保从脚本所在目录运行（避免从别的 cwd 运行导致路径错）
Set-Location $PSScriptRoot

if (-not $NoFrontend) {
  Write-Host "==> Build frontend (Svelte/Vite)" -ForegroundColor Cyan
  Push-Location "frontend"

  if (-not $NoNpmInstall) {
    if (Test-Path "package-lock.json") {
      npm ci
    } else {
      npm install
    }
  } elseif (!(Test-Path "node_modules")) {
    throw "node_modules 不存在，但你传了 -NoNpmInstall。请去掉该参数或先安装依赖。"
  }

  npm run build
  Pop-Location
} else {
  Write-Host "==> Skip frontend build (-NoFrontend)" -ForegroundColor Yellow
}

Write-Host "==> Build backend (Go embed -> single exe)" -ForegroundColor Cyan
go build -o $Out

Write-Host "Done: $Out" -ForegroundColor Green


