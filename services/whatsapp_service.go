package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type BulkMessageItem struct {
	PhoneNumber string `json:"phone_number"`
	Message     string `json:"message"`
}

type BulkMessageResult struct {
	SuccessCount int                  `json:"success_count"`
	FailCount    int                  `json:"fail_count"`
	Errors       []BulkMessageItemErr `json:"errors"`
}

type BulkMessageItemErr struct {
	PhoneNumber string `json:"phone_number"`
	Error       string `json:"error"`
}

type WhatsAppService interface {
	SendMessage(phoneNumber, message string) error
	SendScheduledMessage(phoneNumber, message string, scheduledAt time.Time) error
	SendBulkMessages(messages []BulkMessageItem) BulkMessageResult
}

type whatsappService struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type SendMessageRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	Text        string `json:"text"`
	Mode        string `json:"mode,omitempty"`
}

type sendMessageErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

func NewWhatsAppService(baseURL, apiKey string) WhatsAppService {
	return &whatsappService{
		baseURL: baseURL,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *whatsappService) SendMessage(phoneNumber, message string) error {
	return s.send(SendMessageRequest{
		PhoneNumber: phoneNumber,
		Text:        message,
		Mode:        "chat",
	})
}

func (s *whatsappService) SendScheduledMessage(phoneNumber, message string, scheduledAt time.Time) error {
	_ = scheduledAt

	return s.send(SendMessageRequest{
		PhoneNumber: phoneNumber,
		Text:        message,
		Mode:        "chat",
	})
}

func (s *whatsappService) SendBulkMessages(messages []BulkMessageItem) BulkMessageResult {
	result := BulkMessageResult{}
	for _, item := range messages {
		if err := s.SendMessage(item.PhoneNumber, item.Message); err != nil {
			result.FailCount++
			result.Errors = append(result.Errors, BulkMessageItemErr{
				PhoneNumber: item.PhoneNumber,
				Error:       err.Error(),
			})
		} else {
			result.SuccessCount++
		}
	}
	return result
}

func (s *whatsappService) send(reqBody SendMessageRequest) error {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/messages/send", strings.TrimRight(s.baseURL, "/")),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(s.apiKey) != "" {
		req.Header.Set("X-API-KEY", s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("whatsapp api returned status %d", resp.StatusCode)
	}

	var result sendMessageErrorResponse
	if len(body) > 0 && json.Unmarshal(body, &result) == nil {
		switch {
		case strings.TrimSpace(result.Error) != "":
			return fmt.Errorf("whatsapp api returned status %d: %s", resp.StatusCode, result.Error)
		case strings.TrimSpace(result.Message) != "":
			return fmt.Errorf("whatsapp api returned status %d: %s", resp.StatusCode, result.Message)
		}
	}

	if trimmedBody := strings.TrimSpace(string(body)); trimmedBody != "" {
		return fmt.Errorf("whatsapp api returned status %d: %s", resp.StatusCode, trimmedBody)
	}

	return fmt.Errorf("whatsapp api returned status %d", resp.StatusCode)
}
