#!/usr/bin/env bash
# 生成 .env 所需的全部密钥（SECRET_KEY / AES_KEY / ECC 密钥对）
#
# 用法：
#   bash scripts/gen-secrets.sh            # 输出到 stdout，供人工填入 .env
#   bash scripts/gen-secrets.sh > .env     # 直接生成完整 .env（会覆盖！）
#
# 依赖：openssl（用于随机数）、python3 或 go（用于 ECC 密钥对）
set -euo pipefail

rand_hex() { openssl rand -hex "$1"; }

echo "# ──────────────────────────────────────────────────────────────"
echo "# 由 scripts/gen-secrets.sh 生成于 $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "# ⚠️ 本文件含真实密钥，不要提交到仓库"
echo "# ──────────────────────────────────────────────────────────────"
echo
echo "# ── MongoDB ──"
echo "MONGO_PORT=27017"
echo "MONGO_INITDB_ROOT_USERNAME=change_me"
echo "MONGO_INITDB_ROOT_PASSWORD=$(rand_hex 16)"
echo "MONGO_INITDB_DATABASE=buct-course"
echo
echo "# ── 后端应用 ──"
echo "GO_PORT=5000"
echo "# JWT 签名密钥（64 字节随机）"
echo "SECRET_KEY=$(rand_hex 32)"
echo
echo "# ── 邮件（需自行填写真实值，脚本无法生成）──"
echo "MAIL_SMTP_SERVER=smtp.163.com"
echo "MAIL_SMTP_PORT=465"
echo "MAIL_SENDER=your_email@163.com"
echo "# 注意是邮箱「授权码」，不是登录密码"
echo "MAIL_PASSWORD=change_me"
echo "VERIFY_CODE_EXPIRE=180"
echo
echo "# ── AES 密钥（对称加密，任意长度，内部会做 SHA-256）──"
echo "AES_KEY=$(rand_hex 32)"
echo
echo "# ── ECC 密钥对（P-256，用于加密存储学生学校密码）──"

# 优先用 python3（cryptography），退回到 go
if command -v python3 >/dev/null 2>&1 && python3 -c 'import cryptography' 2>/dev/null; then
  python3 - <<'PY'
from cryptography.hazmat.primitives.asymmetric import ec
import secrets

priv = ec.generate_private_key(ec.SECP256R1())
n = priv.private_numbers().private_value
pub = priv.public_key().public_numbers().x
print(f"ECC_PRIVATE_KEY={n}")
print(f"ECC_PUBLIC_KEY={pub}")
PY
elif command -v go >/dev/null 2>&1; then
  # 用 Go 生成，避免依赖 python cryptography
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT
  cat > "$tmp/main.go" <<'GO'
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
	fmt.Printf("ECC_PRIVATE_KEY=%s\n", n.String())
	fmt.Printf("ECC_PUBLIC_KEY=%s\n", x.String())
}
GO
  (cd "$tmp" && go mod init genkey >/dev/null 2>&1 && go run main.go)
else
  echo "# 无法生成 ECC 密钥：需要 python3+cryptography 或 go" >&2
  echo "ECC_PRIVATE_KEY=generate_manually"
  echo "ECC_PUBLIC_KEY=generate_manually"
fi

echo
echo "# ── RSA（防爬虫请求加密）──"
echo "RSA_KEY_SIZE=2048"
echo "RSA_KEY_EXPIRE_MINUTES=30"
echo "RSA_ENABLE=true"
echo
echo "# ── Snowflake（多实例部署时每个实例设不同值）──"
echo "SNOWFLAKE_NODE=1"
echo
echo "# ── 前端 / Nginx ──"
echo "FRONTEND_PORT=3033"
echo "NGINX_HTTP_PORT=80"
echo "NGINX_HTTPS_PORT=443"
echo
echo "# ── 其他 ──"
echo "TEST_MODE=false"
