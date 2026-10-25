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
	SendReminder(to, number string, total, balance int, dueDate string) (string, string)
	Send(to, subject, html string) (string, string)
}
type brevoService struct {
	apiKey, senderEmail, senderName string
	client                          *http.Client
}

func NewBrevoService() BrevoService {
	return &brevoService{apiKey: strings.TrimSpace(os.Getenv("BREVO_API_KEY")), senderEmail: strings.TrimSpace(os.Getenv("BREVO_SENDER_EMAIL")), senderName: strings.TrimSpace(os.Getenv("BREVO_SENDER_NAME")), client: &http.Client{Timeout: 10 * time.Second}}
}

// Configured reports whether Brevo credentials are present, so callers can
// record a meaningful notification status instead of a generic failure.
func (s *brevoService) Configured() bool { return s.apiKey != "" && s.senderEmail != "" }

func (s *brevoService) SendInvoice(to, number string, total, balance int) (string, string) {
	html := fmt.Sprintf("<p>Invoice <strong>%s</strong> telah diterbitkan oleh Cantika.net.</p><p>Total: Rp %d<br/>Sisa tagihan: Rp %d</p>", number, total, balance)
	return s.Send(to, "Invoice "+number, html)
}

func (s *brevoService) SendReminder(to, number string, total, balance int, dueDate string) (string, string) {
	html := fmt.Sprintf("<p>Pengingat pembayaran invoice <strong>%s</strong>.</p><p>Jatuh tempo: %s<br/>Total: Rp %d<br/>Sisa tagihan: Rp %d</p>", number, dueDate, total, balance)
	return s.Send(to, "Pengingat Invoice "+number, html)
}

func (s *brevoService) Send(to, subject, html string) (string, string) {
	if s.apiKey == "" || s.senderEmail == "" {
		return "", "Brevo is not configured"
	}
	payload := map[string]interface{}{"sender": map[string]string{"email": s.senderEmail, "name": s.senderName}, "to": []map[string]string{{"email": to}}, "subject": subject, "htmlContent": html}
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
