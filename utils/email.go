package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const brevoAPIURL = "https://api.brevo.com/v3/smtp/email"

type brevoSender struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type brevoRecipient struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type brevoEmailRequest struct {
	Sender      brevoSender      `json:"sender"`
	To          []brevoRecipient `json:"to"`
	Subject     string           `json:"subject"`
	HTMLContent string           `json:"htmlContent"`
}

func sendBrevoEmail(to, toName, subject, htmlContent string) error {
	apiKey := os.Getenv("BREVO_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("BREVO_API_KEY is not set")
	}

	senderEmail := os.Getenv("BREVO_SENDER_EMAIL")
	senderName := os.Getenv("BREVO_SENDER_NAME")
	if senderEmail == "" {
		senderEmail = os.Getenv("SMTP_EMAIL") // fallback
	}
	if senderName == "" {
		senderName = os.Getenv("APP_NAME")
	}

	payload := brevoEmailRequest{
		Sender: brevoSender{Name: senderName, Email: senderEmail},
		To:     []brevoRecipient{{Email: to, Name: toName}},
		Subject:     subject,
		HTMLContent: htmlContent,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, brevoAPIURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Brevo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return fmt.Errorf("brevo API error (status %d): %v", resp.StatusCode, errBody)
	}

	return nil
}

// SendOTPEmail sends a registration OTP email to the user.
func SendOTPEmail(toEmail, toName, otp string) error {
	appName := os.Getenv("APP_NAME")
	if appName == "" {
		appName = "Potensia"
	}

	subject := fmt.Sprintf("Kode OTP Verifikasi Akun %s", appName)
	html := fmt.Sprintf(`<!DOCTYPE html>
			<html>
			<head><meta charset="UTF-8"></head>
			<body style="font-family: Arial, sans-serif; background:#f4f4f4; margin:0; padding:0;">
			<table width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f4f4; padding:40px 0;">
				<tr><td align="center">
				<table width="520" cellpadding="0" cellspacing="0" style="background:#ffffff; border-radius:8px; overflow:hidden; box-shadow:0 2px 8px rgba(0,0,0,0.08);">
					<tr><td style="background:#4F46E5; padding:28px 40px;">
					<h1 style="color:#ffffff; margin:0; font-size:22px;">%s</h1>
					</td></tr>
					<tr><td style="padding:32px 40px;">
					<h2 style="color:#1a1a1a; margin:0 0 12px;">Verifikasi Akun Kamu</h2>
					<p style="color:#555; line-height:1.6; margin:0 0 24px;">
						Halo <strong>%s</strong>,<br>
						Terima kasih sudah mendaftar di %s. Gunakan kode OTP berikut untuk memverifikasi akun kamu:
					</p>
					<div style="background:#F5F3FF; border:2px dashed #4F46E5; border-radius:8px; padding:20px; text-align:center; margin-bottom:24px;">
						<span style="font-size:40px; font-weight:bold; letter-spacing:10px; color:#4F46E5;">%s</span>
					</div>
					<p style="color:#888; font-size:13px; margin:0;">
						Kode ini berlaku selama <strong>5 menit</strong>. Jangan bagikan kode ini kepada siapapun.
					</p>
					</td></tr>
					<tr><td style="background:#f9f9f9; padding:16px 40px; text-align:center;">
					<p style="color:#aaa; font-size:12px; margin:0;">Email ini dikirim otomatis oleh %s. Abaikan jika kamu tidak merasa mendaftar.</p>
					</td></tr>
				</table>
				</td></tr>
			</table>
			</body>
			</html>`, appName, toName, appName, otp, appName)

	return sendBrevoEmail(toEmail, toName, subject, html)
}

// SendForgotPasswordEmail sends a password reset link email.
func SendForgotPasswordEmail(toEmail, toName, resetToken string) error {
	appName := os.Getenv("APP_NAME")
	if appName == "" {
		appName = "Potensia"
	}
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:3000"
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", appURL, resetToken)
	subject := fmt.Sprintf("Reset Password Akun %s", appName)

	html := fmt.Sprintf(`<!DOCTYPE html>
			<html>
			<head><meta charset="UTF-8"></head>
			<body style="font-family: Arial, sans-serif; background:#f4f4f4; margin:0; padding:0;">
			<table width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f4f4; padding:40px 0;">
				<tr><td align="center">
				<table width="520" cellpadding="0" cellspacing="0" style="background:#ffffff; border-radius:8px; overflow:hidden; box-shadow:0 2px 8px rgba(0,0,0,0.08);">
					<tr><td style="background:#4F46E5; padding:28px 40px;">
					<h1 style="color:#ffffff; margin:0; font-size:22px;">%s</h1>
					</td></tr>
					<tr><td style="padding:32px 40px;">
					<h2 style="color:#1a1a1a; margin:0 0 12px;">Reset Password</h2>
					<p style="color:#555; line-height:1.6; margin:0 0 24px;">
						Halo <strong>%s</strong>,<br>
						Kami menerima permintaan reset password untuk akun kamu. Klik tombol di bawah untuk membuat password baru:
					</p>
					<div style="text-align:center; margin-bottom:28px;">
						<a href="%s" style="display:inline-block; background:#4F46E5; color:#ffffff; text-decoration:none; padding:14px 36px; border-radius:6px; font-size:16px; font-weight:bold;">
						Reset Password
						</a>
					</div>
					<p style="color:#888; font-size:13px; margin:0 0 8px;">
						Atau copy link berikut ke browser kamu:
					</p>
					<p style="color:#4F46E5; font-size:12px; word-break:break-all; margin:0 0 24px;">%s</p>
					<p style="color:#888; font-size:13px; margin:0;">
						Link ini berlaku selama <strong>15 menit</strong> dan hanya bisa digunakan sekali.
						Jika kamu tidak meminta reset password, abaikan email ini — akun kamu tetap aman.
					</p>
					</td></tr>
					<tr><td style="background:#f9f9f9; padding:16px 40px; text-align:center;">
					<p style="color:#aaa; font-size:12px; margin:0;">Email ini dikirim otomatis oleh %s.</p>
					</td></tr>
				</table>
				</td></tr>
			</table>
			</body>
			</html>`, appName, toName, resetLink, resetLink, appName)

	return sendBrevoEmail(toEmail, toName, subject, html)
}