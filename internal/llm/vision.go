package llm

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Image is one photo to show the AI.
type Image struct {
	Data      []byte
	MediaType string // image/jpeg, image/png, image/webp
}

// ImageReader is a client that can look at photos. Whether the chosen
// model can see images is only known by trying.
type ImageReader interface {
	ReadImages(ctx context.Context, model, prompt string, images []Image) (string, Usage, error)
}

// partsMessage is an OpenAI-style message with text and image parts.
type partsMessage struct {
	Role    string `json:"role"`
	Content []part `json:"content"`
}

type part struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

// ReadImages asks an OpenAI-compatible vision model (OpenAI, Gemini, Ollama
// llama3.2-vision, qwen2.5vl…) about photos sent as data: URLs.
func (c *Client) ReadImages(ctx context.Context, model, prompt string, images []Image) (string, Usage, error) {
	if strings.TrimSpace(c.BaseURL) == "" || strings.TrimSpace(model) == "" {
		return "", Usage{}, errors.New("LLM base URL and model must be configured")
	}
	parts := []part{{Type: "text", Text: prompt}}
	for _, im := range images {
		data := "data:" + im.MediaType + ";base64," + base64.StdEncoding.EncodeToString(im.Data)
		parts = append(parts, part{Type: "image_url", ImageURL: &imageURL{URL: data}})
	}
	return c.send(ctx, chatRequest{
		Model:     model,
		Messages:  []any{partsMessage{Role: "user", Content: parts}},
		MaxTokens: MaxTokens(model),
	})
}

// ReadImages asks Claude about photos.
func (c *AnthropicClient) ReadImages(ctx context.Context, model, prompt string, images []Image) (string, Usage, error) {
	if strings.TrimSpace(c.APIKey) == "" || strings.TrimSpace(model) == "" {
		return "", Usage{}, errors.New("Claude needs an Anthropic API key and a model")
	}
	opts := []option.RequestOption{option.WithAPIKey(c.APIKey)}
	if c.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(c.BaseURL))
	}
	var blocks []anthropic.ContentBlockParamUnion
	for _, im := range images {
		blocks = append(blocks, anthropic.NewImageBlockBase64(im.MediaType, base64.StdEncoding.EncodeToString(im.Data)))
	}
	blocks = append(blocks, anthropic.NewTextBlock(prompt))
	client := anthropic.NewClient(opts...)
	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: int64(MaxTokens(model)),
		Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(blocks...)},
	})
	if err != nil {
		return "", Usage{}, err
	}
	usage := Usage{PromptTokens: int(resp.Usage.InputTokens), CompletionTokens: int(resp.Usage.OutputTokens)}
	var text strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			text.WriteString(t.Text)
		}
	}
	return text.String(), usage, nil
}
