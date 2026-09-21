# 生成 .env 所需的全部密钥（Windows / PowerShell 版）
#
# 用法：
#   pwsh web/scripts/gen-secrets.ps1              # 打印到屏幕
#   pwsh web/scripts/gen-secrets.ps1 -OutFile .env  # 写入文件（会覆盖）
#
# ECC 密钥对通过临时 Go 程序生成（需要本机有 go）。

param(
    [string]$OutFile,
    [string]$MongoUser = "change_me",
    [string]$MailSender = "your_email@163.com"
)

$ErrorActionPreference = 'Stop'

function New-HexSecret([int]$bytes) {
    $buf = New-Object byte[] $bytes
    [System.Security.Cryptography.RandomNumberGenerator]::Fill($buf)
    return ($buf | ForEach-Object { $_.ToString('x2') }) -join ''
}

function New-EccKeyPair {
    $go = Get-Command go -ErrorAction SilentlyContinue
    if (-not $go) {
        Write-Warning "未找到 go，无法生成 ECC 密钥对，请手动生成"
        return @{ Private = 'generate_manually'; Public = 'generate_manually' }
    }

    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("genkey-" + [guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Force -Path $tmp | Out-Null
    try {
        $src = @'
package main

import (
	"crypto/ecdh"
	"crypto/rand"
	"fmt"
	"math/big"
)

func main() {
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	n := new(big.Int).SetBytes(priv.Bytes())
	pub := priv.PublicKey().Bytes()
	x := new(big.Int).SetBytes(pub[1:33])
	fmt.Printf("%s\n%s\n", n.String(), x.String())
}
'@
        Set-Content -Path (Join-Path $tmp 'main.go') -Value $src -Encoding UTF8
        Push-Location $tmp
        try {
            & go mod init genkey 2>$null | Out-Null
            $out = & go run main.go 2>&1
            if ($LASTEXITCODE -ne 0) { throw "go run 失败: $out" }
            $lines = $out | Where-Object { $_ -match '^\d+$' }
            return @{ Private = $lines[0]; Public = $lines[1] }
        } finally {
            Pop-Location
        }
    } finally {
        Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
    }
}

Write-Host "正在生成密钥..." -ForegroundColor Cyan
$ecc = New-EccKeyPair

$lines = @(
    "# ──────────────────────────────────────────────────────────────"
    "# 由 web/scripts/gen-secrets.ps1 生成于 $(Get-Date -Format 'yyyy-MM-ddTHH:mm:ssZ')"
    "# ⚠️ 本文件含真实密钥，不要提交到仓库（.env 已在 .gitignore 中忽略）"
    "# ──────────────────────────────────────────────────────────────"
    ""
    "# ── MongoDB ──"
    "MONGO_PORT=27017"
    "MONGO_INITDB_ROOT_USERNAME=$MongoUser"
    "MONGO_INITDB_ROOT_PASSWORD=$(New-HexSecret 16)"
    "MONGO_INITDB_DATABASE=buct-course"
    ""
    "# ── 后端应用 ──"
    "GO_PORT=5000"
    "# JWT 签名密钥（32 字节随机）"
    "SECRET_KEY=$(New-HexSecret 32)"
    ""
    "# ── 邮件（需自行填写真实值，脚本无法生成）──"
    "MAIL_SMTP_SERVER=smtp.163.com"
    "MAIL_SMTP_PORT=465"
    "MAIL_SENDER=$MailSender"
    "# 注意是邮箱「授权码」，不是登录密码"
    "MAIL_PASSWORD=change_me"
    "VERIFY_CODE_EXPIRE=180"
    ""
    "# ── AES 密钥（对称加密，内部会做 SHA-256 派生）──"
    "AES_KEY=$(New-HexSecret 32)"
    ""
    "# ── ECC 密钥对（P-256，加密存储学生学校密码）──"
    "# ⚠️ 更换此密钥会导致库中已加密的 s_password 无法解密，必须同步迁移"
    "ECC_PRIVATE_KEY=$($ecc.Private)"
    "ECC_PUBLIC_KEY=$($ecc.Public)"
    ""
    "# ── RSA（防爬虫请求加密）──"
    "RSA_KEY_SIZE=2048"
    "RSA_KEY_EXPIRE_MINUTES=30"
    "RSA_ENABLE=true"
    ""
    "# ── Snowflake（多实例部署时每个实例设不同值）──"
    "SNOWFLAKE_NODE=1"
    ""
    "# ── 前端 / Nginx ──"
    "FRONTEND_PORT=3033"
    "NGINX_HTTP_PORT=80"
    "NGINX_HTTPS_PORT=443"
    ""
    "# ── 其他 ──"
    "TEST_MODE=false"
)

if ($OutFile) {
    $full = [System.IO.Path]::GetFullPath($OutFile)
    if (Test-Path $full) {
        Write-Warning "文件已存在，将被覆盖: $full"
    }
    $lines -join "`n" | Set-Content -Path $full -Encoding UTF8 -NoNewline
    Write-Host "已写入: $full" -ForegroundColor Green
    Write-Host "请手动填写 MAIL_SENDER 与 MAIL_PASSWORD 后使用。" -ForegroundColor Yellow
} else {
    $lines -join "`n"
}
