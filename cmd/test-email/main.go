package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"arc-lms/internal/config"
	"arc-lms/internal/pkg/email"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Get recipient from command line or use default
	recipient := "smsnmicheal@gmail.com" // Default to SMTP user
	if len(os.Args) > 1 {
		recipient = os.Args[1]
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create email service
	emailService := email.NewEmailService(cfg.SMTP, nil)

	if !emailService.IsConfigured() {
		log.Fatal("Email service is not configured. Check SMTP environment variables.")
	}

	fmt.Printf("📧 Sending test email to: %s\n", recipient)
	fmt.Printf("   From: %s\n", cfg.SMTP.From)
	fmt.Printf("   SMTP Host: %s:%d\n", cfg.SMTP.Host, cfg.SMTP.Port)

	// Send test email
	err = emailService.Send(
		recipient,
		"Arc LMS - Test Email",
		"This is a test email from Arc LMS to verify the email service is working correctly.",
		`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #4A90D9; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
        .content { padding: 20px; background: #f9f9f9; border: 1px solid #ddd; }
        .footer { text-align: center; padding: 20px; font-size: 12px; color: #666; }
        .success { background: #d4edda; border: 1px solid #c3e6cb; padding: 15px; border-radius: 4px; margin: 15px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎉 Email Test Successful!</h1>
        </div>
        <div class="content">
            <div class="success">
                <strong>✅ Your Arc LMS email service is working correctly!</strong>
            </div>
            <p>This is a test email sent from the Arc LMS application to verify that the Brevo SMTP integration is functioning properly.</p>
            <p><strong>Configuration Details:</strong></p>
            <ul>
                <li>SMTP Host: smtp-relay.brevo.com</li>
                <li>Port: 587 (STARTTLS)</li>
                <li>Status: Connected and authenticated</li>
            </ul>
            <p>You can now use the email service for:</p>
            <ul>
                <li>Scheduled email broadcasts</li>
                <li>Meeting reminders</li>
                <li>Deadline notifications</li>
                <li>Invoice notifications</li>
                <li>Welcome emails</li>
                <li>Password reset emails</li>
            </ul>
        </div>
        <div class="footer">
            <p>© 2024 Arc LMS. All rights reserved.</p>
            <p>This is an automated test email.</p>
        </div>
    </div>
</body>
</html>`,
	)

	if err != nil {
		log.Fatalf("❌ Failed to send email: %v", err)
	}

	fmt.Println("✅ Test email sent successfully!")
	fmt.Println("   Check your inbox (and spam folder) for the test email.")
}
