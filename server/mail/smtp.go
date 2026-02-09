package mail

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"
)

// SMTPConfig holds connection parameters for an SMTP submission server.
type SMTPConfig struct {
	Host   string
	Port   int
	User   string
	Pass   string
	UseSSL bool // true = implicit TLS (port 465); false = STARTTLS (port 587/25)
}

// SendRequest is the payload passed to SMTPClient.Send.
type SendRequest struct {
	From    string
	To      []string
	Subject string
	Body    string
}

// SMTPClient speaks SMTP over a single TCP connection.
type SMTPClient struct {
	cfg    SMTPConfig
	conn   net.Conn
	reader *bufio.Reader
}

func NewSMTPClient(cfg SMTPConfig) *SMTPClient {
	return &SMTPClient{cfg: cfg}
}

// Connect opens the connection and reads the server greeting.
func (c *SMTPClient) Connect() error {
	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	var err error

	if c.cfg.UseSSL {
		c.conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: c.cfg.Host})
	} else {
		c.conn, err = net.DialTimeout("tcp", addr, 30*time.Second)
	}
	if err != nil {
		return fmt.Errorf("smtp connect %s: %w", addr, err)
	}
	c.reader = bufio.NewReader(c.conn)

	if _, err := c.readResponse(); err != nil {
		c.conn.Close()
		return fmt.Errorf("smtp greeting: %w", err)
	}
	return nil
}

// Handshake performs EHLO and upgrades to TLS via STARTTLS when the connection
// is not already encrypted.
func (c *SMTPClient) Handshake() error {
	if _, err := c.cmd("EHLO mulamail"); err != nil {
		if _, err := c.cmd("HELO mulamail"); err != nil {
			return fmt.Errorf("smtp EHLO/HELO: %w", err)
		}
	}

	if !c.cfg.UseSSL {
		if resp, err := c.cmd("STARTTLS"); err == nil && strings.HasPrefix(resp, "220") {
			tlsConn := tls.Client(c.conn, &tls.Config{ServerName: c.cfg.Host})
			if err := tlsConn.Handshake(); err != nil {
				return fmt.Errorf("smtp TLS handshake: %w", err)
			}
			c.conn = tlsConn
			c.reader = bufio.NewReader(tlsConn)
			c.cmd("EHLO mulamail") //nolint:errcheck // best-effort re-EHLO
		}
	}
	return nil
}

// Auth attempts AUTH PLAIN and falls back to AUTH LOGIN.
func (c *SMTPClient) Auth() error {
	creds := fmt.Sprintf("\x00%s\x00%s", c.cfg.User, c.cfg.Pass)
	encoded := base64.StdEncoding.EncodeToString([]byte(creds))

	if resp, err := c.cmd("AUTH PLAIN " + encoded); err == nil && strings.HasPrefix(resp, "235") {
		return nil
	}
	return c.authLogin()
}

func (c *SMTPClient) authLogin() error {
	if _, err := c.cmd("AUTH LOGIN"); err != nil {
		return fmt.Errorf("smtp AUTH LOGIN init: %w", err)
	}
	// Server sends base64("Username:") challenge – we just send the answer.
	if _, err := c.cmd(base64.StdEncoding.EncodeToString([]byte(c.cfg.User))); err != nil {
		return fmt.Errorf("smtp AUTH LOGIN user: %w", err)
	}
	if _, err := c.cmd(base64.StdEncoding.EncodeToString([]byte(c.cfg.Pass))); err != nil {
		return fmt.Errorf("smtp AUTH LOGIN pass: %w", err)
	}
	return nil
}

// Send transmits a single message.  The connection must already be
// authenticated.
func (c *SMTPClient) Send(req SendRequest) error {
	if _, err := c.cmd(fmt.Sprintf("MAIL FROM:<%s>", req.From)); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	for _, to := range req.To {
		if _, err := c.cmd(fmt.Sprintf("RCPT TO:<%s>", to)); err != nil {
			return fmt.Errorf("smtp RCPT TO %s: %w", to, err)
		}
	}
	if _, err := c.cmd("DATA"); err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}

	// Build a minimal RFC 5322 message.
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n",
		req.From,
		strings.Join(req.To, ", "),
		req.Subject,
		time.Now().Format(time.RFC1123Z),
		req.Body,
	)

	// Write with dot-stuffing.
	for _, line := range strings.Split(msg, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, ".") {
			line = "." + line
		}
		if _, err := fmt.Fprintf(c.conn, "%s\r\n", line); err != nil {
			return err
		}
	}
	// Terminate the DATA phase.
	if _, err := fmt.Fprintf(c.conn, ".\r\n"); err != nil {
		return err
	}
	if _, err := c.readResponse(); err != nil {
		return fmt.Errorf("smtp DATA end: %w", err)
	}
	return nil
}

// Close sends QUIT and tears down the connection.
func (c *SMTPClient) Close() error {
	if c.conn == nil {
		return nil
	}
	c.cmd("QUIT") //nolint:errcheck
	return c.conn.Close()
}

// ---------- low-level protocol helpers ----------

func (c *SMTPClient) cmd(command string) (string, error) {
	if _, err := fmt.Fprintf(c.conn, "%s\r\n", command); err != nil {
		return "", err
	}
	return c.readResponse()
}

// readResponse handles both single-line and multi-line SMTP replies.
// It returns an error for 4xx / 5xx status codes.
func (c *SMTPClient) readResponse() (string, error) {
	var last string
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		last = strings.TrimRight(line, "\r\n")
		// Multi-line reply continues while the 4th character is '-'.
		if len(last) < 4 || last[3] != '-' {
			break
		}
	}
	if len(last) >= 1 && (last[0] == '4' || last[0] == '5') {
		return last, fmt.Errorf("smtp: %s", last)
	}
	return last, nil
}
