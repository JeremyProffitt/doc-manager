package models

// Customer represents a customer whose data is used to populate forms.
type Customer struct {
	ID       string `json:"id" dynamodbav:"id"`
	Name     string `json:"name" dynamodbav:"name"`
	Business string `json:"business" dynamodbav:"business"`
	Address  string `json:"address" dynamodbav:"address"`
	City     string `json:"city" dynamodbav:"city"`
	State    string `json:"state" dynamodbav:"state"`
	Zip      string `json:"zip" dynamodbav:"zip"`
	Phone    string `json:"phone" dynamodbav:"phone"`
}
