package models

// FieldPlacement represents a versioned set of field placements for a form.
type FieldPlacement struct {
	FormID     string  `json:"formId" dynamodbav:"formId"`
	Version    int     `json:"version" dynamodbav:"version"`
	CreatedAt  string  `json:"createdAt" dynamodbav:"createdAt"`
	Source     string  `json:"source" dynamodbav:"source"` // ai_analysis, manual_edit, revert_from_vN
	FontFamily string  `json:"fontFamily" dynamodbav:"fontFamily"`
	FontSize   int     `json:"fontSize" dynamodbav:"fontSize"`
	Fields     []Field `json:"fields" dynamodbav:"fields"`
}

// Field represents a single field placement on a PDF page.
type Field struct {
	FieldName  string  `json:"fieldName" dynamodbav:"fieldName"`
	Page       int     `json:"page" dynamodbav:"page"`
	X          float64 `json:"x" dynamodbav:"x"`
	Y          float64 `json:"y" dynamodbav:"y"`
	Width      float64 `json:"width" dynamodbav:"width"`
	Height     float64 `json:"height" dynamodbav:"height"`
	FontFamily *string `json:"fontFamily" dynamodbav:"fontFamily"`
	FontSize   *int    `json:"fontSize" dynamodbav:"fontSize"`
	Confidence float64 `json:"confidence,omitempty" dynamodbav:"confidence,omitempty"`
	Reasoning  string  `json:"reasoning,omitempty" dynamodbav:"reasoning,omitempty"`
}
