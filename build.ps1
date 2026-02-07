param(
    [string]$BaseOutputName = "RUN_LOG"
)

# 出错时立即停止
$ErrorActionPreference = "Stop"

# 构建输出目录
$BuildDir = "build"

# 若 build 目录不存在则创建
if (!(Test-Path $BuildDir)) {
    New-Item -ItemType Directory -Path $BuildDir | Out-Null
    Write-Host "Created build directory: $BuildDir" -ForegroundColor DarkGray
}

# 清空目录
Remove-Item -Path .\build\* -Recurse -Force

# 将配置文件复制一份到build便于打包
if (-not (Test-Path .\build)) {
    New-Item -ItemType Directory -Path .\build | Out-Null
}

Copy-Item -Path .\config.yaml -Destination .\build\ -Force

# 获取时间戳
$TimeStamp = Get-Date -Format "yyyyMMdd_HHmmss"

# 构建目标列表
$BuildTargets = @(
    @{ GOOS = "linux";   GOARCH = "amd64";  EXT = ""     },
    @{ GOOS = "linux";   GOARCH = "arm64";  EXT = ""     },
    @{ GOOS = "windows"; GOARCH = "amd64";  EXT = ".exe" }
)

Write-Host ""
Write-Host "Start Building Go Project (Multi-Target)" -ForegroundColor Cyan
Write-Host "Timestamp : $TimeStamp" -ForegroundColor DarkGray
Write-Host "Output Dir: $BuildDir" -ForegroundColor DarkGray
Write-Host ""

foreach ($Target in $BuildTargets) {

    $env:GOOS   = $Target.GOOS
    $env:GOARCH = $Target.GOARCH

    $OutputName = "${BaseOutputName}_${env:GOOS}_${env:GOARCH}_${TimeStamp}" + $Target.EXT
    $OutputPath = Join-Path $BuildDir $OutputName

    Write-Host "----------------------------------------" -ForegroundColor DarkGray
    Write-Host "Target OS   : $env:GOOS"   -ForegroundColor DarkBlue
    Write-Host "Target Arch : $env:GOARCH" -ForegroundColor DarkBlue
    Write-Host "Output File : $OutputPath" -ForegroundColor DarkBlue
    Write-Host ""

    # 删除旧文件
    if (Test-Path $OutputPath) {
        Remove-Item $OutputPath -Force
        Write-Host "Removed Old Binary: $OutputPath" -ForegroundColor DarkGray
    }

    # 构建
    Write-Host "Building..." -ForegroundColor Cyan
    go build -o $OutputPath

    if ($LASTEXITCODE -eq 0) {
        Write-Host "Build Succeeded: $OutputName" -ForegroundColor Green
    } else {
        Write-Host "Build Failed for $env:GOOS / $env:GOARCH" -ForegroundColor Red
        exit 1
    }

    Write-Host ""
}

Write-Host "========================================" -ForegroundColor DarkGray
Write-Host "All Builds Completed Successfully" -ForegroundColor Green
Write-Host ""
