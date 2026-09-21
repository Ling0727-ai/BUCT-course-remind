# 把 .env 中的变量批量推送到 GitHub Actions Secrets
#
# 用法：
#   pwsh web/scripts/set-github-secrets.ps1 -EnvFile web/.env -Repo owner/repo
#   pwsh web/scripts/set-github-secrets.ps1 -EnvFile web/.env -Repo owner/repo -WhatIf
#
# 前置：gh CLI 已登录（gh auth login），且对目标仓库有 admin 权限。
# 注意：Secrets 是单向的，写入后无法读回明文，只能覆盖或删除。

param(
    [Parameter(Mandatory = $true)][string]$EnvFile,
    [Parameter(Mandatory = $true)][string]$Repo,
    # 只打印将写入哪些键，不实际写入
    [switch]$WhatIf,
    # 同时写入 GitHub Environment（用于审批/环境保护），留空则写仓库级 Secrets
    [string]$Environment
)

$ErrorActionPreference = 'Stop'

if (-not (Test-Path $EnvFile)) {
    throw "找不到文件: $EnvFile"
}

$gh = Get-Command gh -ErrorAction SilentlyContinue
if (-not $gh) { throw "未找到 gh CLI，请先安装：https://cli.github.com/" }

# 检查登录状态
& gh auth status 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) { throw "gh 未登录，请先执行 gh auth login" }

# 解析 .env：忽略空行与注释，支持 KEY=VALUE（值可含 = ）
$entries = @()
foreach ($line in Get-Content $EnvFile) {
    $trimmed = $line.Trim()
    if ($trimmed -eq '' -or $trimmed.StartsWith('#')) { continue }
    $idx = $trimmed.IndexOf('=')
    if ($idx -lt 1) { continue }
    $key = $trimmed.Substring(0, $idx).Trim()
    $val = $trimmed.Substring($idx + 1).Trim()

    # 去掉可能的引号
    if ($val.Length -ge 2 -and (
        ($val.StartsWith('"') -and $val.EndsWith('"')) -or
        ($val.StartsWith("'") -and $val.EndsWith("'")))) {
        $val = $val.Substring(1, $val.Length - 2)
    }

    if ($key -notmatch '^[A-Za-z_][A-Za-z0-9_]*$') {
        Write-Warning "跳过非法变量名: $key"
        continue
    }
    $entries += [pscustomobject]@{ Key = $key; Value = $val }
}

if ($entries.Count -eq 0) { throw "未从 $EnvFile 解析到任何变量" }

Write-Host "从 $EnvFile 解析到 $($entries.Count) 个变量" -ForegroundColor Cyan
Write-Host "目标仓库: $Repo" -ForegroundColor Cyan
if ($Environment) { Write-Host "目标 Environment: $Environment" -ForegroundColor Cyan }
Write-Host ""

# 敏感项：值看起来是占位符时提醒（避免把 your_xxx 推上去）
$placeholderPattern = '^(change_me|your_|generate_manually|xxx|<.*>)$'

$ok = 0
$failed = @()
foreach ($e in $entries) {
    $isPlaceholder = $e.Value -match $placeholderPattern
    $mark = if ($isPlaceholder) { '⚠ 占位符' } else { '✓' }

    if ($WhatIf) {
        Write-Host ("  {0} {1,-32} {2}" -f $mark, $e.Key, $(if ($isPlaceholder) { $e.Value } else { '(已设置，值不显示)' }))
        continue
    }

    # 通过 stdin 传值，避免出现在命令行历史/进程列表中
    $ghArgs = @('secret', 'set', $e.Key, '--repo', $Repo)
    if ($Environment) { $ghArgs += @('--env', $Environment) }

    $e.Value | & gh @ghArgs 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Host ("  {0} {1}" -f $mark, $e.Key)
        $ok++
    } else {
        Write-Host ("  ✗ {0}" -f $e.Key) -ForegroundColor Red
        $failed += $e.Key
    }
}

Write-Host ""
if ($WhatIf) {
    Write-Host "（WhatIf 模式，未实际写入）" -ForegroundColor Yellow
} else {
    Write-Host "完成：成功 $ok / $($entries.Count)" -ForegroundColor Green
    if ($failed.Count -gt 0) {
        Write-Host "失败: $($failed -join ', ')" -ForegroundColor Red
        exit 1
    }
}
