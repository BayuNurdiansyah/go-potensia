package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const brevoAPI = "https://api.brevo.com/v3/smtp/email"

type brevoPayload struct {
	Sender      map[string]string   `json:"sender"`
	To          []map[string]string `json:"to"`
	Subject     string              `json:"subject"`
	HTMLContent string              `json:"htmlContent"`
}

func sendEmail(toEmail, toName, subject, html string) error {
	apiKey := os.Getenv("BREVO_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("BREVO_API_KEY not set")
	}

	senderEmail := os.Getenv("BREVO_SENDER_EMAIL")
	senderName  := os.Getenv("BREVO_SENDER_NAME")
	if senderEmail == "" {
		senderEmail = os.Getenv("SMTP_EMAIL")
	}

	name := appName()
	if senderName == "" {
		senderName = name
	}

	payload := brevoPayload{
		Sender:      map[string]string{"name": senderName, "email": senderEmail},
		To:          []map[string]string{{"email": toEmail, "name": toName}},
		Subject:     subject,
		HTMLContent: html,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, brevoAPI, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("brevo error status: %d", resp.StatusCode)
	}
	return nil
}

func appName() string {
	if n := os.Getenv("APP_NAME"); n != "" {
		return n
	}
	return "Potensia"
}

// appBaseURL returns the base URL of the backend server itself.
// Link reset password diarahkan ke halaman HTML yang di-serve Go server.
// Di .env set: APP_URL=https://go-potensia.onrender.com
// Di local:    APP_URL=http://localhost:8080
func appBaseURL() string {
	if u := os.Getenv("APP_URL"); u != "" {
		return u
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return "http://localhost:" + port
}

func SendOTPEmail(toEmail, toName, otp string) error {
	app := appName()
	html := fmt.Sprintf(`<!DOCTYPE html><html><body style="font-family:Arial,sans-serif;background:#f4f4f4;margin:0;padding:0">
<table width="100%%" cellpadding="0" cellspacing="0" style="padding:40px 0">
<tr><td align="center">
<table width="520" style="background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,.08)">
<tr><td style="background:#1B4FD8;padding:28px 40px"><h1 style="color:#fff;margin:0;font-size:22px">%s</h1></td></tr>
<tr><td style="padding:32px 40px">
<h2 style="color:#1a1a1a;margin:0 0 12px">Verifikasi Akun</h2>
<p style="color:#555;line-height:1.6;margin:0 0 24px">Halo <strong>%s</strong>, gunakan kode berikut untuk verifikasi akun kamu:</p>
<div style="background:#EFF6FF;border:2px dashed #1B4FD8;border-radius:8px;padding:20px;text-align:center;margin-bottom:24px">
<span style="font-size:40px;font-weight:bold;letter-spacing:10px;color:#1B4FD8">%s</span>
</div>
<p style="color:#888;font-size:13px;margin:0">Kode berlaku <strong>5 menit</strong>. Jangan bagikan ke siapapun.</p>
</td></tr>
<tr><td style="background:#f9f9f9;padding:16px 40px;text-align:center">
<p style="color:#aaa;font-size:12px;margin:0">Email otomatis dari %s. Abaikan jika tidak mendaftar.</p>
</td></tr>
</table></td></tr></table></body></html>`, app, toName, otp, app)

	return sendEmail(toEmail, toName, fmt.Sprintf("Kode OTP Verifikasi %s", app), html)
}

func SendForgotPasswordEmail(toEmail, toName, token string) error {
	app  := appName()
	base := appBaseURL()

	// Link mengarah ke halaman HTML di Go server itu sendiri
	link := fmt.Sprintf("%s/reset-password?token=%s", base, token)

	html := fmt.Sprintf(`<!DOCTYPE html><html><body style="font-family:Arial,sans-serif;background:#f4f4f4;margin:0;padding:0">
<table width="100%%" cellpadding="0" cellspacing="0" style="padding:40px 0">
<tr><td align="center">
<table width="520" style="background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,.08)">
<tr><td style="background:#1B4FD8;padding:28px 40px"><h1 style="color:#fff;margin:0;font-size:22px">%s</h1></td></tr>
<tr><td style="padding:32px 40px">
<h2 style="color:#1a1a1a;margin:0 0 12px">Reset Password</h2>
<p style="color:#555;line-height:1.6;margin:0 0 24px">Halo <strong>%s</strong>, klik tombol berikut untuk membuat password baru:</p>
<div style="text-align:center;margin-bottom:28px">
<a href="%s" style="display:inline-block;background:#1B4FD8;color:#fff;text-decoration:none;padding:14px 36px;border-radius:8px;font-size:16px;font-weight:bold">Buat Password Baru</a>
</div>
<p style="color:#888;font-size:13px;margin:0 0 8px">Atau copy link ini ke browser:</p>
<p style="color:#1B4FD8;font-size:12px;word-break:break-all;margin:0 0 24px">%s</p>
<p style="color:#888;font-size:13px;margin:0">Link berlaku <strong>15 menit</strong> dan hanya bisa dipakai sekali.<br/>Abaikan email ini jika kamu tidak meminta reset password.</p>
</td></tr>
<tr><td style="background:#f9f9f9;padding:16px 40px;text-align:center">
<p style="color:#aaa;font-size:12px;margin:0">Email otomatis dari %s.</p>
</td></tr>
</table></td></tr></table></body></html>`, app, toName, link, link, app)

	return sendEmail(toEmail, toName, fmt.Sprintf("Reset Password %s", app), html)
}