package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Notifier interface {
	Send(message string) error
}

type TelegramNotifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

type DiscordNotifier struct {
	webhookURL string
	client     *http.Client
}

func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
		client:   &http.Client{},
	}
}

func (t *TelegramNotifier) Send(message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)
	payload := map[string]string{
		"chat_id": t.chatID,
		"text":    message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := t.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status: %d", resp.StatusCode)
	}
	return nil
}
