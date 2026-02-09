package mail

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// POP3Config holds connection parameters for a POP3 mail server.
type POP3Config struct {
	Host   string
	Port   int
	User   string
	Pass   string
	UseSSL bool
}

// Message is a lightweight representation of an email, used both for inbox
// previews (From/Subject/Date only) and full retrieval (Body populated).
type Message struct {
	ID      int    `json:"id"`
	Size    int    `json:"size"`
	From    string `json:"from,omitempty"`
	Subject string `json:"subject,omitempty"`
	Date    string `json:"date,omitempty"`
	Body    string `json:"body,omitempty"`
}

// POP3Client speaks the POP3 protocol over a single TCP connection.
type POP3Client struct {
	cfg    POP3Config
	conn   net.Conn
	reader *bufio.Reader
}

func NewPOP3Client(cfg POP3Config) *POP3Client {
	return &POP3Client{cfg: cfg}
}

// Connect opens the TCP (or TLS) connection and reads the server greeting.
func (c *POP3Client) Connect() error {
	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	var err error

	if c.cfg.UseSSL {
		c.conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: c.cfg.Host})
	} else {
		c.conn, err = net.DialTimeout("tcp", addr, 30*time.Second)
	}
	if err != nil {
		return fmt.Errorf("pop3 connect %s: %w", addr, err)
	}
	c.reader = bufio.NewReader(c.conn)

	// Consume server greeting line.
	if _, err := c.readResponse(); err != nil {
		c.conn.Close()
		return fmt.Errorf("pop3 greeting: %w", err)
	}
	return nil
}

// Auth performs USER/PASS authentication.
func (c *POP3Client) Auth() error {
	if _, err := c.cmd("USER " + c.cfg.User); err != nil {
		return fmt.Errorf("pop3 USER: %w", err)
	}
	if _, err := c.cmd("PASS " + c.cfg.Pass); err != nil {
		return fmt.Errorf("pop3 PASS: %w", err)
	}
	return nil
}

// List returns every message in the mailbox with its index and size.
func (c *POP3Client) List() ([]Message, error) {
	if _, err := c.cmd("LIST"); err != nil {
		return nil, err
	}
	lines, err := c.readDot()
	if err != nil {
		return nil, err
	}

	msgs := make([]Message, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
		if len(parts) != 2 {
			continue
		}
		id, _ := strconv.Atoi(parts[0])
		size, _ := strconv.Atoi(parts[1])
		msgs = append(msgs, Message{ID: id, Size: size})
	}
	return msgs, nil
}

// Top fetches the headers (and optionally the first bodyLines lines) of a
// message without downloading the whole thing.  It returns a Message with
// From/Subject/Date parsed out of the headers.
func (c *POP3Client) Top(id, bodyLines int) (*Message, error) {
	if _, err := c.cmd(fmt.Sprintf("TOP %d %d", id, bodyLines)); err != nil {
		return nil, err
	}
	lines, err := c.readDot()
	if err != nil {
		return nil, err
	}
	content := strings.Join(lines, "\r\n")
	h := parseHeaders(content)

	msg := &Message{
		ID:      id,
		From:    h["from"],
		Subject: h["subject"],
		Date:    h["date"],
	}
	if bodyLines > 0 {
		if parts := strings.SplitN(content, "\r\n\r\n", 2); len(parts) == 2 {
			msg.Body = parts[1]
		}
	}
	return msg, nil
}

// Retrieve downloads the complete raw message.
func (c *POP3Client) Retrieve(id int) (string, error) {
	if _, err := c.cmd(fmt.Sprintf("RETR %d", id)); err != nil {
		return "", err
	}
	lines, err := c.readDot()
	if err != nil {
		return "", err
	}
	return strings.Join(lines, "\r\n"), nil
}

// Close sends QUIT and tears down the connection.
func (c *POP3Client) Close() error {
	if c.conn == nil {
		return nil
	}
	c.cmd("QUIT") //nolint:errcheck
	return c.conn.Close()
}

// ---------- low-level protocol helpers ----------

func (c *POP3Client) cmd(command string) (string, error) {
	if _, err := fmt.Fprintf(c.conn, "%s\r\n", command); err != nil {
		return "", err
	}
	return c.readResponse()
}

// readResponse reads a single status line.  Returns an error if the server
// replied with -ERR.
func (c *POP3Client) readResponse() (string, error) {
	line, err := c.readLine()
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(line, "-ERR") {
		return "", fmt.Errorf("pop3: %s", line)
	}
	return line, nil
}

func (c *POP3Client) readLine() (string, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// readDot reads a dot-terminated multi-line body, handling dot-unstuffing.
func (c *POP3Client) readDot() ([]string, error) {
	var lines []string
	for {
		line, err := c.readLine()
		if err != nil {
			return nil, err
		}
		if line == "." {
			break
		}
		if strings.HasPrefix(line, "..") {
			line = line[1:] // dot-unstuff
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// ---------- header parsing ----------

// parseHeaders does a best-effort extraction of common headers from the raw
// header block.  Folded (continuation) headers are skipped for simplicity.
func parseHeaders(raw string) map[string]string {
	h := make(map[string]string)
	for _, line := range strings.Split(raw, "\r\n") {
		if line == "" {
			break // end of headers
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue // skip folded continuation
		}
		k, v, ok := strings.Cut(line, ": ")
		if ok {
			h[strings.ToLower(k)] = v
		}
	}
	return h
}
