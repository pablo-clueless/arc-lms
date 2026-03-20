package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"strings"
	"time"

	"arc-lms/internal/config"
)

// EmailService handles email sending via SMTP (Brevo)
type EmailService struct {
	host     string
	port     int
	user     string
	password string
	from     string
	logger   *log.Logger
}

// NewEmailService creates a new email service
func NewEmailService(cfg config.SMTPConfig, logger *log.Logger) *EmailService {
	if logger == nil {
		logger = log.Default()
	}
	return &EmailService{
		host:     cfg.Host,
		port:     cfg.Port,
		user:     cfg.User,
		password: cfg.Password,
		from:     cfg.From,
		logger:   logger,
	}
}

// Email represents an email to be sent
type Email struct {
	To          []string
	CC          []string
	BCC         []string
	Subject     string
	Body        string     // Plain text body
	HTMLBody    string     // HTML body
	ReplyTo     string
	Attachments []Attachment
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// SendResult represents the result of sending an email
type SendResult struct {
	Recipient string
	Success   bool
	Error     string
	SentAt    time.Time
}

// IsConfigured returns true if SMTP is configured
func (s *EmailService) IsConfigured() bool {
	return s.host != "" && s.user != "" && s.password != ""
}

// Send sends an email to a single recipient
func (s *EmailService) Send(to, subject, textBody, htmlBody string) error {
	return s.SendEmail(&Email{
		To:       []string{to},
		Subject:  subject,
		Body:     textBody,
		HTMLBody: htmlBody,
	})
}

// SendEmail sends an email with full options
func (s *EmailService) SendEmail(email *Email) error {
	if !s.IsConfigured() {
		s.logger.Println("[EmailService] SMTP not configured, skipping email send")
		return nil
	}

	if len(email.To) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	// Build the email message
	msg := s.buildMessage(email)

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	// Use TLS for port 465, STARTTLS for 587
	var err error
	if s.port == 465 {
		err = s.sendWithTLS(addr, email, msg)
	} else {
		err = s.sendWithStartTLS(addr, email, msg)
	}

	if err != nil {
		s.logger.Printf("[EmailService] Failed to send email to %v: %v", email.To, err)
		return err
	}

	s.logger.Printf("[EmailService] Email sent successfully to %v", email.To)
	return nil
}

// SendBatch sends emails to multiple recipients individually
func (s *EmailService) SendBatch(recipients []string, subject, textBody, htmlBody string) []SendResult {
	results := make([]SendResult, len(recipients))

	for i, recipient := range recipients {
		err := s.Send(recipient, subject, textBody, htmlBody)
		results[i] = SendResult{
			Recipient: recipient,
			Success:   err == nil,
			SentAt:    time.Now(),
		}
		if err != nil {
			results[i].Error = err.Error()
		}
	}

	return results
}

// sendWithStartTLS sends email using STARTTLS (port 587)
func (s *EmailService) sendWithStartTLS(addr string, email *Email, msg []byte) error {
	// Connect to the server
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Send EHLO/HELO
	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("EHLO failed: %w", err)
	}

	// Start TLS
	tlsConfig := &tls.Config{
		ServerName: s.host,
		MinVersion: tls.VersionTLS12,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS failed: %w", err)
	}

	// Authenticate
	auth := smtp.PlainAuth("", s.user, s.password, s.host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Set the sender
	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}

	// Set all recipients (To, CC, BCC)
	allRecipients := append(append(email.To, email.CC...), email.BCC...)
	for _, recipient := range allRecipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("RCPT TO failed for %s: %w", recipient, err)
		}
	}

	// Send the email body
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA failed: %w", err)
	}

	_, err = writer.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return client.Quit()
}

// sendWithTLS sends email using direct TLS (port 465)
func (s *EmailService) sendWithTLS(addr string, email *Email, msg []byte) error {
	tlsConfig := &tls.Config{
		ServerName: s.host,
		MinVersion: tls.VersionTLS12,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server with TLS: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Authenticate
	auth := smtp.PlainAuth("", s.user, s.password, s.host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Set the sender
	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}

	// Set all recipients
	allRecipients := append(append(email.To, email.CC...), email.BCC...)
	for _, recipient := range allRecipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("RCPT TO failed for %s: %w", recipient, err)
		}
	}

	// Send the email body
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA failed: %w", err)
	}

	_, err = writer.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return client.Quit()
}

// buildMessage constructs the email message with headers and body
func (s *EmailService) buildMessage(email *Email) []byte {
	var buf bytes.Buffer

	// Headers
	buf.WriteString(fmt.Sprintf("From: %s\r\n", s.from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ", ")))

	if len(email.CC) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(email.CC, ", ")))
	}

	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	buf.WriteString("MIME-Version: 1.0\r\n")

	if email.ReplyTo != "" {
		buf.WriteString(fmt.Sprintf("Reply-To: %s\r\n", email.ReplyTo))
	}

	// Determine content type
	if email.HTMLBody != "" && email.Body != "" {
		// Multipart message with both plain text and HTML
		boundary := "----=_Part_0_" + fmt.Sprintf("%d", time.Now().UnixNano())
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		buf.WriteString("\r\n")

		// Plain text part
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(email.Body)
		buf.WriteString("\r\n")

		// HTML part
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(email.HTMLBody)
		buf.WriteString("\r\n")

		// End boundary
		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if email.HTMLBody != "" {
		// HTML only
		buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(email.HTMLBody)
	} else {
		// Plain text only
		buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		buf.WriteString("\r\n")
		buf.WriteString(email.Body)
	}

	return buf.Bytes()
}

// SendTemplated sends an email using a template
func (s *EmailService) SendTemplated(to, subject, templateName string, data interface{}) error {
	htmlBody, err := s.renderTemplate(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	return s.Send(to, subject, "", htmlBody)
}

// renderTemplate renders an HTML email template
func (s *EmailService) renderTemplate(templateName string, data interface{}) (string, error) {
	// In production, you would load templates from files
	// For now, use built-in templates
	tmpl, exists := emailTemplates[templateName]
	if !exists {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	t, err := template.New(templateName).Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// Built-in email templates
var emailTemplates = map[string]string{
	"welcome": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #4A90D9; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9f9f9; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
        .button { display: inline-block; padding: 12px 24px; background: #4A90D9; color: white; text-decoration: none; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to Arc LMS!</h1>
        </div>
        <div class="content">
            <p>Hello {{.Name}},</p>
            <p>Welcome to Arc LMS! Your account has been created successfully.</p>
            <p>You can now log in using your credentials and start exploring the platform.</p>
            <p><a href="{{.LoginURL}}" class="button">Log In Now</a></p>
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Arc LMS. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

	"password_reset": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #E74C3C; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9f9f9; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
        .button { display: inline-block; padding: 12px 24px; background: #E74C3C; color: white; text-decoration: none; border-radius: 4px; }
        .code { background: #eee; padding: 10px 20px; font-size: 24px; letter-spacing: 4px; text-align: center; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Password Reset</h1>
        </div>
        <div class="content">
            <p>Hello {{.Name}},</p>
            <p>We received a request to reset your password. Use the code below to proceed:</p>
            <div class="code">{{.ResetCode}}</div>
            <p>This code will expire in {{.ExpiryMinutes}} minutes.</p>
            <p>If you didn't request this, please ignore this email or contact support.</p>
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Arc LMS. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

	"notification": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #27AE60; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9f9f9; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
        .button { display: inline-block; padding: 12px 24px; background: #27AE60; color: white; text-decoration: none; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{{.Title}}</h1>
        </div>
        <div class="content">
            <p>Hello {{.Name}},</p>
            <p>{{.Message}}</p>
            {{if .ActionURL}}
            <p><a href="{{.ActionURL}}" class="button">{{.ActionText}}</a></p>
            {{end}}
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Arc LMS. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

	"invoice": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #2C3E50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9f9f9; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
        .invoice-details { background: white; padding: 15px; border: 1px solid #ddd; margin: 15px 0; }
        .amount { font-size: 24px; font-weight: bold; color: #27AE60; }
        .button { display: inline-block; padding: 12px 24px; background: #27AE60; color: white; text-decoration: none; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Invoice #{{.InvoiceNumber}}</h1>
        </div>
        <div class="content">
            <p>Hello {{.TenantName}},</p>
            <p>Your invoice for {{.Period}} is now available.</p>
            <div class="invoice-details">
                <p><strong>Invoice Number:</strong> {{.InvoiceNumber}}</p>
                <p><strong>Period:</strong> {{.Period}}</p>
                <p><strong>Students:</strong> {{.StudentCount}}</p>
                <p><strong>Due Date:</strong> {{.DueDate}}</p>
                <p class="amount">Total: ₦{{.Amount}}</p>
            </div>
            <p><a href="{{.PaymentURL}}" class="button">Pay Now</a></p>
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Arc LMS. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

	"deadline_reminder": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #F39C12; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9f9f9; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
        .deadline { background: #FFF3CD; border: 1px solid #F39C12; padding: 15px; margin: 15px 0; border-radius: 4px; }
        .button { display: inline-block; padding: 12px 24px; background: #F39C12; color: white; text-decoration: none; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Deadline Reminder</h1>
        </div>
        <div class="content">
            <p>Hello {{.StudentName}},</p>
            <div class="deadline">
                <p><strong>{{.AssessmentType}}:</strong> {{.Title}}</p>
                <p><strong>Course:</strong> {{.CourseName}}</p>
                <p><strong>Due:</strong> {{.Deadline}}</p>
            </div>
            <p>Don't forget to submit before the deadline!</p>
            <p><a href="{{.ActionURL}}" class="button">View {{.AssessmentType}}</a></p>
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Arc LMS. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,

	"meeting_reminder": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #9B59B6; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9f9f9; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
        .meeting-info { background: white; border: 1px solid #9B59B6; padding: 15px; margin: 15px 0; border-radius: 4px; }
        .button { display: inline-block; padding: 12px 24px; background: #9B59B6; color: white; text-decoration: none; border-radius: 4px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Meeting Starting Soon!</h1>
        </div>
        <div class="content">
            <p>Hello {{.Name}},</p>
            <div class="meeting-info">
                <p><strong>Meeting:</strong> {{.MeetingTitle}}</p>
                <p><strong>Starts in:</strong> {{.MinutesBefore}} minutes</p>
                <p><strong>Host:</strong> {{.HostName}}</p>
            </div>
            <p><a href="{{.JoinURL}}" class="button">Join Meeting</a></p>
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Arc LMS. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`,
}
