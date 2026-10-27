package config

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

const smtpAttemptTimeout = 20 * time.Second

// MailConfig 邮件配置，对应 Python Config 中的 MAIL_* 配置
type MailConfig struct {
	SMTPServer string
	SMTPPort   int
	Sender     string
	Password   string
	// VerifyCodeExpire 验证码有效期（秒），对应 Python VERIFY_CODE_EXPIRE
	VerifyCodeExpire int
}

// Mail 全局邮件配置实例
var Mail *MailConfig

// LoadMailConfig 从环境变量加载邮件配置
func LoadMailConfig() *MailConfig {
	port := 465
	if v := os.Getenv("MAIL_SMTP_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			port = n
		}
	}

	expire := 180
	if v := os.Getenv("VERIFY_CODE_EXPIRE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			expire = n
		}
	}

	password := os.Getenv("MAIL_PASSWORD")
	if password == "" {
		password = "dummy_password_for_dev"
		log.Println("[mail] 警告: MAIL_PASSWORD 未设置，邮件功能将不可用")
	}

	Mail = &MailConfig{
		SMTPServer:       getEnvOrDefault("MAIL_SMTP_SERVER", "smtp.163.com"),
		SMTPPort:         port,
		Sender:           getEnvOrDefault("MAIL_SENDER", "buct_course_remind@163.com"),
		Password:         password,
		VerifyCodeExpire: expire,
	}
	return Mail
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// IsReady 检查邮件服务是否可用（密码已配置）
func (m *MailConfig) IsReady() bool {
	return m.Password != "" && m.Password != "dummy_password_for_dev"
}

// smtpCandidate 单个 SMTP 尝试配置
type smtpCandidate struct {
	server string
	port   int
	ssl    bool
}

// SendMail 发送邮件，自动尝试多种 SMTP 配置
// 对应 Python send_email_notification / send_verification_email_enhanced 的多配置重试逻辑
func (m *MailConfig) SendMail(to, subject, htmlBody string) error {
	if !m.IsReady() {
		return fmt.Errorf("邮件服务未配置（MAIL_PASSWORD 未设置）")
	}

	// 候选配置：SSL 优先，其次 StartTLS，对应 Python smtp_candidates
	candidates := []smtpCandidate{
		{m.SMTPServer, m.SMTPPort, true},
		{m.SMTPServer, 587, false},
		{m.SMTPServer, 25, false},
	}

	// 拼装 RFC 2822 邮件报文
	now := time.Now().Format("2006-01-02 15:04:05")
	rawMsg := strings.Join([]string{
		"From: " + m.Sender,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		`Content-Type: text/html; charset="utf-8"`,
		"",
		htmlBody,
		"",
		"发送时间: " + now,
	}, "\r\n")

	auth := smtp.PlainAuth("", m.Sender, m.Password, m.SMTPServer)

	var lastErr error
	for i, c := range candidates {
		addr := fmt.Sprintf("%s:%d", c.server, c.port)
		log.Printf("[mail] 尝试 #%d: %s (SSL=%v)", i+1, addr, c.ssl)

		if c.ssl {
			lastErr = sendSSL(addr, auth, m.Sender, to, rawMsg)
		} else {
			lastErr = sendStartTLS(addr, c.server, auth, m.Sender, to, rawMsg)
		}

		if lastErr == nil {
			log.Printf("[mail] 邮件发送成功: %s (配置 #%d)", to, i+1)
			return nil
		}
		log.Printf("[mail] 配置 #%d 失败: %v", i+1, lastErr)
	}
	return fmt.Errorf("所有 SMTP 配置均失败，最后错误: %w", lastErr)
}

// sendSSL 使用 SMTP over SSL（端口 465）发送
func sendSSL(addr string, auth smtp.Auth, from, to, msg string) error {
	host := strings.Split(addr, ":")[0]
	deadline := time.Now().Add(smtpAttemptTimeout)
	dialer := &net.Dialer{Timeout: 10 * time.Second, Deadline: deadline}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	if err := conn.SetDeadline(deadline); err != nil {
		conn.Close()
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if err = c.Auth(auth); err != nil {
		return err
	}
	if err = c.Mail(from); err != nil {
		return err
	}
	if err = c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, msg)
	if err != nil {
		return err
	}
	return w.Close()
}

// sendStartTLS 使用 SMTP + STARTTLS（端口 587/25）发送
func sendStartTLS(addr, host string, auth smtp.Auth, from, to, msg string) error {
	deadline := time.Now().Add(smtpAttemptTimeout)
	conn, err := (&net.Dialer{Timeout: 10 * time.Second, Deadline: deadline}).Dial("tcp", addr)
	if err != nil {
		return err
	}
	if err := conn.SetDeadline(deadline); err != nil {
		conn.Close()
		return err
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return err
	}
	defer c.Close()
	if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
		return err
	}
	if err = c.Auth(auth); err != nil {
		return err
	}
	if err = c.Mail(from); err != nil {
		return err
	}
	if err = c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, msg)
	if err != nil {
		return err
	}
	return w.Close()
}
