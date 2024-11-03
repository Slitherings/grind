package notifications

type TelegramNotifier struct {
	botKey string
	chatID string
}

func NewTelegramNotifier(botKey, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botKey: botKey,
		chatID: chatID,
	}
}
