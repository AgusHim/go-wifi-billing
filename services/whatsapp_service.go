package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WhatsAppService interface {
	SendMessage(phoneNumber, message string) error
	SendScheduledMessage(phoneNumber, message string, scheduledAt time.Time) error
}

type whatsappService struct {
	baseURL string
	client  *http.Client
}

type SendMessageRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	Message     string `json:"message"`
	ScheduledAt string `json:"scheduledAt,omitempty"`
	Priority    int    `json:"priority,omitempty"`
}

type SendMessageResponse struct {
	Success bool   `json:"success"`
	JobID   string `json:"jobId"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func NewWhatsAppService(baseURL string) WhatsAppService {
	return &whatsappService{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *whatsappService) SendMessage(phoneNumber, message string) error {
	return s.send(SendMessageRequest{
		PhoneNumber: phoneNumber,
		Message:     message,
	})
}

func (s *whatsappService) SendScheduledMessage(phoneNumber, message string, scheduledAt time.Time) error {
	return s.send(SendMessageRequest{
		PhoneNumber: phoneNumber,
		Message:     message,
		ScheduledAt: scheduledAt.Format(time.RFC3339),
	})
}

func (s *whatsappService) send(reqBody SendMessageRequest) error {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := s.client.Post(
		fmt.Sprintf("%s/send", s.baseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var result SendMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("whatsapp api error: %s", result.Error)
	}

	return nil
}
