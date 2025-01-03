package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"vault-injector/config"
)

type Message struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type Telegram struct {
	ChatID int64 `json:"chat_id"`
	Token  config.Password
}

func NewTelegram(config *config.Config) *Telegram {
	return &Telegram{
		ChatID: config.Telegram.Channel,
		Token:  config.Telegram.Token,
	}
}
func (t *Telegram) SendMessage(str string) error {
	msg := &Message{
		ChatID: t.ChatID,
		Text:   fmt.Sprintf("%s", str),
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.Token)
	response, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			log.Println("failed to close response body")
		}
	}(response.Body)
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send successful request. Status was %q", response.Status)
	}
	return nil
}
