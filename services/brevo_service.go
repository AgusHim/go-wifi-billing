package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type BrevoService interface {
	SendInvoice(to, number string, total, balance int) (string, string)
}
type brevoService struct {
	apiKey, senderEmail, senderName string
	client                          *http.Client
}

func NewBrevoService() BrevoService {
	return &brevoService{apiKey: strings.TrimSpace(os.Getenv("BREVO_API_KEY")), senderEmail: strings.TrimSpace(os.Getenv("BREVO_SENDER_EMAIL")), senderName: strings.TrimSpace(os.Getenv("BREVO_SENDER_NAME")), client: &http.Client{Timeout: 10 * time.Second}}
}
func (s *brevoService) SendInvoice(to, number string, total, balance int) (string, string) {
	if s.apiKey == "" || s.senderEmail == "" {
		return "", "Brevo is not configured"
	}
	payload := map[string]interface{}{"sender": map[string]string{"email": s.senderEmail, "name": s.senderName}, "to": []map[string]string{{"email": to}}, "subject": "Invoice " + number, "htmlContent": fmt.Sprintf("<p>Invoice <strong>%s</strong> telah diterbitkan oleh Cantika.net.</p><p>Total: Rp %d<br/>Sisa tagihan: Rp %d</p>", number, total, balance)}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewReader(body))
	if err != nil {
		return "", err.Error()
	}
	req.Header.Set("api-key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Sprintf("Brevo returned HTTP %d", resp.StatusCode)
	}
	var out struct {
		MessageID string `json:"messageId"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return out.MessageID, ""
}
