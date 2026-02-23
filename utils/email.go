package utils

import (
	"fmt"
	"net/smtp"
	"os"
)

func SendOTPEmail(to string, otp string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	auth := smtp.PlainAuth("", from, password, host)

	html := `
	<div style="font-family: Arial; background:#f4f6f8; padding:20px;">
		<div style="max-width:500px; margin:auto; background:white; padding:20px; border-radius:10px;">
			<h2 style="color:#4F46E5;">Potensia 🎓</h2>
			<p>Halo,</p>
			<p>Gunakan kode OTP berikut untuk verifikasi akun kamu:</p>
			
			<div style="text-align:center; margin:20px 0;">
				<span style="font-size:28px; letter-spacing:5px; font-weight:bold; color:#111;">
					` + otp + `
				</span>
			</div>

			<p>Kode ini berlaku selama <b>5 menit</b>.</p>

			<hr/>
			<small>Potensia</small>
		</div>
	</div>
	`

	msg := "MIME-version: 1.0;\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\";\r\n" +
		"Subject: Kode OTP Potensia\r\n\r\n" +
		html

	err := smtp.SendMail(
		host+":"+port,
		auth,
		from,
		[]string{to},
		[]byte(msg),
	)

	if err != nil {
		fmt.Println("Error kirim email:", err)
		return err
	}

	fmt.Println("OTP berhasil dikirim ke:", to)
	return nil
}