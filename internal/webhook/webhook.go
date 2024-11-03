package webhook

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/goccy/go-json"
)

type Embed struct {
	Color       int            `json:"color"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Thumbnail   EmbedThumbnail `json:"thumbnail"`
}

type EmbedThumbnail struct {
	URL string `json:"url"`
}

type WebhookPayload struct {
	Embeds []Embed `json:"embeds"`
}

func SendWebhook(webhookURL string, embed Embed) {
	if webhookURL == "" {
		log.Error("Webhook URL is empty")
		return
	}

	payload := WebhookPayload{
		Embeds: []Embed{embed},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Error("Failed to marshal webhook payload:", err)
		return
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(payloadJSON))
	if err != nil {
		log.Error("Failed to create new webhook request:", "Request Error", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: 3 * time.Second, // Increased timeout
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error("Failed to send webhook:", "Client Error", err)
		return
	}
	defer resp.Body.Close()

	// Check the status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Error("Non-OK webhook response:", resp.Status, string(body))
		return
	}

	log.Info("Webhook sent successfully with status:", "Status", resp.Status)
}
