package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

// BedrockClient interface for mocking the Bedrock runtime API.
type BedrockClient interface {
	InvokeModel(ctx context.Context, params *bedrockruntime.InvokeModelInput, optFns ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error)
}

// BedrockService wraps the Bedrock runtime client for form analysis.
type BedrockService struct {
	client  BedrockClient
	modelID string
}

// NewBedrockService creates a new BedrockService with the given client and model ID.
func NewBedrockService(client BedrockClient, modelID string) *BedrockService {
	return &BedrockService{client: client, modelID: modelID}
}

// bedrockRequest represents the Claude Messages API request body sent to Bedrock.
type bedrockRequest struct {
	AnthropicVersion string           `json:"anthropic_version"`
	MaxTokens        int              `json:"max_tokens"`
	Messages         []bedrockMessage `json:"messages"`
}

// bedrockMessage represents a single message in the Claude Messages API.
type bedrockMessage struct {
	Role    string                `json:"role"`
	Content []bedrockContentBlock `json:"content"`
}

// bedrockContentBlock represents either an image or text block in the request.
type bedrockContentBlock struct {
	Type   string              `json:"type"`
	Source *bedrockImageSource `json:"source,omitempty"`
	Text   string              `json:"text,omitempty"`
}

// bedrockImageSource represents a base64-encoded image source.
type bedrockImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// bedrockResponse represents the Claude Messages API response body from Bedrock.
type bedrockResponseBody struct {
	Content []bedrockResponseContent `json:"content"`
}

// bedrockResponseContent represents a content block in the Bedrock response.
type bedrockResponseContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// aiField represents a field as returned by the AI model (using snake_case JSON tags).
type aiField struct {
	FieldName  string  `json:"field_name"`
	Page       int     `json:"page"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

// AnalyzeForm sends a form image to Bedrock for field detection.
// Returns detected fields with coordinates as percentages.
func (s *BedrockService) AnalyzeForm(imageData []byte, contentType string, standardFields []string) ([]models.Field, error) {
	base64Image := base64.StdEncoding.EncodeToString(imageData)

	prompt := fmt.Sprintf(
		"Analyze this form image. Identify where these fields should be filled in: %s.\n"+
			"For each field, return a JSON array with objects containing:\n"+
			"field_name, page, x, y, width, height (all as percentages 0-100), confidence (0.0-1.0), reasoning.\n"+
			"Return ONLY a valid JSON array. No other text.",
		strings.Join(standardFields, ", "),
	)

	reqBody := bedrockRequest{
		AnthropicVersion: "bedrock-2023-05-31",
		MaxTokens:        4096,
		Messages: []bedrockMessage{
			{
				Role: "user",
				Content: []bedrockContentBlock{
					{
						Type: "image",
						Source: &bedrockImageSource{
							Type:      "base64",
							MediaType: contentType,
							Data:      base64Image,
						},
					},
					{
						Type: "text",
						Text: prompt,
					},
				},
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling bedrock request: %w", err)
	}

	result, err := s.client.InvokeModel(context.Background(), &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(s.modelID),
		ContentType: aws.String("application/json"),
		Body:        body,
	})
	if err != nil {
		return nil, fmt.Errorf("invoking bedrock model: %w", err)
	}

	var resp bedrockResponseBody
	if err := json.Unmarshal(result.Body, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling bedrock response: %w", err)
	}

	// Extract the text content from the response
	var textContent string
	for _, block := range resp.Content {
		if block.Type == "text" {
			textContent = block.Text
			break
		}
	}

	if textContent == "" {
		return nil, fmt.Errorf("no text content in bedrock response")
	}

	// Parse the JSON array of fields from the text
	var aiFields []aiField
	if err := json.Unmarshal([]byte(textContent), &aiFields); err != nil {
		return nil, fmt.Errorf("parsing field detection response: %w", err)
	}

	// Convert AI fields to model fields
	fields := make([]models.Field, len(aiFields))
	for i, af := range aiFields {
		fields[i] = models.Field{
			FieldName:  af.FieldName,
			Page:       af.Page,
			X:          af.X,
			Y:          af.Y,
			Width:      af.Width,
			Height:     af.Height,
			Confidence: af.Confidence,
			Reasoning:  af.Reasoning,
		}
	}

	return fields, nil
}
