package models

// Setting represents a key-value configuration entry stored in DynamoDB.
type Setting struct {
	Key   string `json:"key" dynamodbav:"key"`
	Value string `json:"value" dynamodbav:"value"` // JSON-encoded value
}

// FieldDefinition represents a standard field that can be placed on a form.
type FieldDefinition struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
