package notifications

import (
	"log"
)

func (t *TelegramNotifier) SendMessage(message string) error {
	log.Printf("Sending telegram notification: %s", message)
	// Implementation
	return nil
}
