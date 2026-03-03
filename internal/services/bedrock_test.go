package services

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

// mockBedrockClient implements BedrockClient for testing.
type mockBedrockClient struct {
	response []byte
	err      error
	// captured stores the last InvokeModelInput for assertion.
	captured *bedrockruntime.InvokeModelInput
}

func (m *mockBedrockClient) InvokeModel(_ context.Context, params *bedrockruntime.InvokeModelInput, _ ...func(*bedrockruntime.Options)) (*bedrockruntime.InvokeModelOutput, error) {
	m.captured = params
	if m.err != nil {
		return nil, m.err
	}
	return &bedrockruntime.InvokeModelOutput{
		Body: m.response,
	}, nil
}

// bedrockResponse builds a Claude Messages API response body with the given text content.
func bedrockResponse(text string) []byte {
	resp := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": text,
			},
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

func TestBedrockService_AnalyzeForm(t *testing.T) {
	tests := []struct {
		name           string
		response       []byte
		clientErr      error
		imageData      []byte
		contentType    string
		standardFields []string
		wantFields     int
		wantErr        bool
		checkFields    func(t *testing.T, fields []models.Field)
		checkRequest   func(t *testing.T, input *bedrockruntime.InvokeModelInput)
	}{
		{
			name: "valid JSON response parses into fields correctly",
			response: bedrockResponse(`[
				{"field_name":"Name","page":1,"x":10.5,"y":20.3,"width":30.0,"height":5.0,"confidence":0.95,"reasoning":"Found name label"},
				{"field_name":"Date","page":1,"x":60.0,"y":20.3,"width":20.0,"height":5.0,"confidence":0.88,"reasoning":"Found date label"}
			]`),
			imageData:      []byte("fake-image-data"),
			contentType:    "image/png",
			standardFields: []string{"Name", "Date"},
			wantFields:     2,
			wantErr:        false,
			checkFields: func(t *testing.T, fields []models.Field) {
				t.Helper()
				if fields[0].FieldName != "Name" {
					t.Errorf("expected field name 'Name', got %q", fields[0].FieldName)
				}
				if fields[0].Page != 1 {
					t.Errorf("expected page 1, got %d", fields[0].Page)
				}
				if fields[0].X != 10.5 {
					t.Errorf("expected X 10.5, got %f", fields[0].X)
				}
				if fields[0].Confidence != 0.95 {
					t.Errorf("expected confidence 0.95, got %f", fields[0].Confidence)
				}
				if fields[1].FieldName != "Date" {
					t.Errorf("expected field name 'Date', got %q", fields[1].FieldName)
				}
			},
		},
		{
			name:           "malformed JSON response returns error",
			response:       bedrockResponse(`{not valid json`),
			imageData:      []byte("fake-image-data"),
			contentType:    "image/png",
			standardFields: []string{"Name"},
			wantErr:        true,
		},
		{
			name:           "empty response returns empty field list",
			response:       bedrockResponse(`[]`),
			imageData:      []byte("fake-image-data"),
			contentType:    "image/png",
			standardFields: []string{"Name"},
			wantFields:     0,
			wantErr:        false,
			checkFields: func(t *testing.T, fields []models.Field) {
				t.Helper()
				if len(fields) != 0 {
					t.Errorf("expected 0 fields, got %d", len(fields))
				}
			},
		},
		{
			name: "fields with low confidence are preserved with scores intact",
			response: bedrockResponse(`[
				{"field_name":"SSN","page":1,"x":10.0,"y":50.0,"width":20.0,"height":5.0,"confidence":0.15,"reasoning":"Uncertain match"},
				{"field_name":"Phone","page":1,"x":10.0,"y":60.0,"width":20.0,"height":5.0,"confidence":0.02,"reasoning":"Very low confidence"}
			]`),
			imageData:      []byte("fake-image-data"),
			contentType:    "image/png",
			standardFields: []string{"SSN", "Phone"},
			wantFields:     2,
			wantErr:        false,
			checkFields: func(t *testing.T, fields []models.Field) {
				t.Helper()
				if fields[0].Confidence != 0.15 {
					t.Errorf("expected confidence 0.15, got %f", fields[0].Confidence)
				}
				if fields[1].Confidence != 0.02 {
					t.Errorf("expected confidence 0.02, got %f", fields[1].Confidence)
				}
			},
		},
		{
			name: "prompt includes all configured standard fields",
			response: bedrockResponse(`[
				{"field_name":"FirstName","page":1,"x":10.0,"y":20.0,"width":20.0,"height":5.0,"confidence":0.9,"reasoning":"ok"}
			]`),
			imageData:      []byte("fake-image-data"),
			contentType:    "image/jpeg",
			standardFields: []string{"FirstName", "LastName", "DOB", "Address"},
			wantFields:     1,
			wantErr:        false,
			checkRequest: func(t *testing.T, input *bedrockruntime.InvokeModelInput) {
				t.Helper()
				var reqBody map[string]interface{}
				if err := json.Unmarshal(input.Body, &reqBody); err != nil {
					t.Fatalf("failed to unmarshal request body: %v", err)
				}
				messages, ok := reqBody["messages"].([]interface{})
				if !ok || len(messages) == 0 {
					t.Fatal("expected messages in request body")
				}
				msg := messages[0].(map[string]interface{})
				content := msg["content"].([]interface{})

				// Find the text content block
				var promptText string
				for _, block := range content {
					b := block.(map[string]interface{})
					if b["type"] == "text" {
						promptText = b["text"].(string)
					}
				}
				for _, field := range []string{"FirstName", "LastName", "DOB", "Address"} {
					if !strings.Contains(promptText, field) {
						t.Errorf("prompt should contain field %q, but it doesn't: %s", field, promptText)
					}
				}
			},
		},
		{
			name: "multi-page form has page numbers assigned correctly",
			response: bedrockResponse(`[
				{"field_name":"Name","page":1,"x":10.0,"y":20.0,"width":30.0,"height":5.0,"confidence":0.95,"reasoning":"Page 1 field"},
				{"field_name":"Signature","page":3,"x":10.0,"y":80.0,"width":40.0,"height":10.0,"confidence":0.85,"reasoning":"Page 3 field"}
			]`),
			imageData:      []byte("fake-image-data"),
			contentType:    "image/png",
			standardFields: []string{"Name", "Signature"},
			wantFields:     2,
			wantErr:        false,
			checkFields: func(t *testing.T, fields []models.Field) {
				t.Helper()
				if fields[0].Page != 1 {
					t.Errorf("expected page 1 for first field, got %d", fields[0].Page)
				}
				if fields[1].Page != 3 {
					t.Errorf("expected page 3 for second field, got %d", fields[1].Page)
				}
			},
		},
		{
			name:           "Bedrock client error returns error",
			clientErr:      errMock("bedrock invocation failed"),
			imageData:      []byte("fake-image-data"),
			contentType:    "image/png",
			standardFields: []string{"Name"},
			wantErr:        true,
		},
		{
			name:           "malformed outer response body returns error",
			response:       []byte(`not json at all`),
			imageData:      []byte("fake-image-data"),
			contentType:    "image/png",
			standardFields: []string{"Name"},
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockBedrockClient{
				response: tt.response,
				err:      tt.clientErr,
			}
			svc := NewBedrockService(client, "anthropic.claude-sonnet-4-20250514")

			fields, err := svc.AnalyzeForm(tt.imageData, tt.contentType, tt.standardFields)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AnalyzeForm() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(fields) != tt.wantFields {
				t.Errorf("expected %d fields, got %d", tt.wantFields, len(fields))
			}
			if tt.checkFields != nil {
				tt.checkFields(t, fields)
			}
			if tt.checkRequest != nil {
				tt.checkRequest(t, client.captured)
			}
		})
	}
}

// errMock is a simple error type for tests.
type errMock string

func (e errMock) Error() string { return string(e) }
