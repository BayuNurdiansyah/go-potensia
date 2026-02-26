package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func SendOTPEmail(toEmail string, name string, otp string) error {
	apiKey := os.Getenv("BREVO_API_KEY")

	url := "https://api.brevo.com/v3/smtp/email"

	htmlContent := fmt.Sprintf(`
	<div style="font-family: Arial; background:#f4f6f8; padding:20px;">
		<div style="max-width:600px;margin:auto;background:white;border-radius:10px;padding:30px;">
			<h2 style="color:#4f46e5;">Potensia</h2>
			<p>Halo %s 👋</p>
			<p>Gunakan kode OTP berikut untuk verifikasi akun kamu:</p>
			
			<div style="text-align:center;margin:30px 0;">
				<span style="font-size:32px;font-weight:bold;color:#111;">%s</span>
			</div>

			<p style="color:#555;">Kode ini berlaku selama 5 menit.</p>

			<hr>
			<p style="font-size:12px;color:#999;">
				Potensia - Platform Edukasi Skill 🚀
			</p>
		</div>
	</div>
	`, name, otp)

	payload := map[string]interface{}{
		"sender": map[string]string{
			"name":  "Potensia",
			"email": "bayuhaft118@gmail.com",
		},
		"to": []map[string]string{
			{"email": toEmail, "name": name},
		},
		"subject":     "Kode OTP Verifikasi Akun",
		"htmlContent": htmlContent,
	}

	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("accept", "application/json")
	req.Header.Set("api-key", apiKey)
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("gagal kirim email, status: %d", resp.StatusCode)
	}

	return nil
}