package mail

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/apperr"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
)

type CodeSender interface {
	SendEmailCode(ctx context.Context, email string, purpose string, code string, ttl time.Duration) error
}

func NewSender(cfg config.MailConfig, log *slog.Logger) (CodeSender, error) {
	switch cfg.Provider {
	case "log":
		return &LogSender{log: log}, nil
	case "smtp":
		return NewSMTPSender(cfg.SMTP), nil
	default:
		return nil, fmt.Errorf("unsupported mail provider %q", cfg.Provider)
	}
}

type LogSender struct {
	log *slog.Logger
}

func (sender *LogSender) SendEmailCode(ctx context.Context, email string, purpose string, code string, ttl time.Duration) error {
	if sender.log != nil {
		sender.log.InfoContext(ctx, "email verification code generated", "email", email, "purpose", purpose, "code", code, "ttl_seconds", int64(ttl.Seconds()))
	}
	return nil
}

type SMTPSender struct {
	cfg config.SMTPConfig
}

func NewSMTPSender(cfg config.SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (sender *SMTPSender) SendEmailCode(ctx context.Context, email string, purpose string, code string, ttl time.Duration) error {
	headerFrom, envelopeFrom, err := parseFromAddress(sender.cfg.From)
	if err != nil {
		return err
	}

	subject := mime.QEncoding.Encode("UTF-8", "CUMT Nexus 验证码")
	body := fmt.Sprintf("你的 CUMT Nexus 验证码是：%s\n\n用途：%s\n有效期：%d 分钟。\n如果不是你本人操作，请忽略这封邮件。\n", code, purpose, int(ttl.Minutes()))
	message := "From: " + headerFrom + "\r\n" +
		"To: " + email + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" + body

	addr := fmt.Sprintf("%s:%d", sender.cfg.Host, sender.cfg.Port)
	auth := smtp.Auth(nil)
	if strings.TrimSpace(sender.cfg.Username) != "" {
		auth = smtp.PlainAuth("", sender.cfg.Username, sender.cfg.Password, sender.cfg.Host)
	}

	sendCtx := ctx
	var cancel context.CancelFunc
	if _, ok := sendCtx.Deadline(); !ok {
		sendCtx, cancel = context.WithTimeout(sendCtx, 10*time.Second)
		defer cancel()
	}

	switch sender.cfg.TLSMode {
	case "ssl":
		dialer := tls.Dialer{
			NetDialer: &net.Dialer{Timeout: 10 * time.Second},
			Config:    &tls.Config{ServerName: sender.cfg.Host, MinVersion: tls.VersionTLS12},
		}
		conn, err := dialer.DialContext(sendCtx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("connect smtp ssl: %w", err)
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, sender.cfg.Host)
		if err != nil {
			return fmt.Errorf("create smtp ssl client: %w", err)
		}
		defer client.Close()
		return sendWithClient(client, auth, envelopeFrom, email, []byte(message))
	case "starttls":
		client, err := dialSMTP(sendCtx, addr, sender.cfg.Host)
		if err != nil {
			return fmt.Errorf("connect smtp: %w", err)
		}
		defer client.Close()
		if err := client.StartTLS(&tls.Config{ServerName: sender.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("start smtp tls: %w", err)
		}
		return sendWithClient(client, auth, envelopeFrom, email, []byte(message))
	default:
		client, err := dialSMTP(sendCtx, addr, sender.cfg.Host)
		if err != nil {
			return fmt.Errorf("connect smtp: %w", err)
		}
		defer client.Close()
		return sendWithClient(client, auth, envelopeFrom, email, []byte(message))
	}
}

func parseFromAddress(raw string) (string, string, error) {
	address, err := mail.ParseAddress(strings.TrimSpace(raw))
	if err != nil {
		return "", "", fmt.Errorf("parse smtp from: %w", err)
	}
	return address.String(), address.Address, nil
}

func dialSMTP(ctx context.Context, addr string, host string) (*smtp.Client, error) {
	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return client, nil
}

func sendWithClient(client *smtp.Client, auth smtp.Auth, from string, to string, msg []byte) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return mapSMTPRecipientError(err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := writer.Write(msg); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}

func mapSMTPRecipientError(err error) error {
	var textErr *textproto.Error
	if errors.As(err, &textErr) && textErr.Code >= 500 && textErr.Code < 600 {
		return apperr.New(apperr.CodeInvalidArgument, "email cannot receive verification code")
	}
	return fmt.Errorf("smtp rcpt: %w", err)
}
