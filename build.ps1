param(
  # 输出 exe 名称（默认与 README 一致）
  [string]$Out = "yseren.exe",
  # 显式版本号。留空时只有精确 vX.Y.Z tag 构建会显示正式版本。
  [string]$Version = "",
  # 跳过前端构建（仅用于排查）
  [switch]$NoFrontend,
  # 跳过依赖安装（如果你确信 node_modules 已经是最新的）
  [switch]$NoNpmInstall
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

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

$version = $Version.Trim()
if ($version -eq "") {
  $version = "dev"
  try {
    $exactTag = git describe --tags --exact-match 2>$null
    if ($exactTag -match "^v(?<ver>(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*))$") {
      $version = $Matches["ver"]
    }
  } catch {
    # 非 tag 或无 git 环境时保持 dev。
  }
} elseif ($version -ne "dev" -and $version -notmatch "^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$") {
  throw "Version 必须是 dev 或不带 v 前缀的 MAJOR.MINOR.PATCH：$version"
}

Write-Host "Version: $version" -ForegroundColor DarkGray
go build -trimpath -ldflags="-X main.Version=$version" -o $Out

Write-Host "Done: $Out" -ForegroundColor Green


