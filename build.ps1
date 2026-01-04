Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Write-Host "==> Build frontend (Svelte/Vite)" -ForegroundColor Cyan
Push-Location "frontend"

if (!(Test-Path "node_modules")) {
  npm install
}

npm run build
Pop-Location

Write-Host "==> Build backend (Go embed -> single exe)" -ForegroundColor Cyan
go build -o "lv-link.exe"

Write-Host "Done: lv-link.exe" -ForegroundColor Green


