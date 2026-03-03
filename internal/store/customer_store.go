package store

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

// DynamoCustomerStore implements CustomerStore using DynamoDB.
type DynamoCustomerStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewCustomerStore creates a new DynamoDB-backed CustomerStore.
func NewCustomerStore(client *dynamodb.Client, tableName string) *DynamoCustomerStore {
	return &DynamoCustomerStore{
		client:    client,
		tableName: tableName,
	}
}

// CreateCustomer creates a new customer in DynamoDB. Returns an error if the
// customer already exists (uses a condition expression to prevent overwrites).
func (s *DynamoCustomerStore) CreateCustomer(customer *models.Customer) error {
	item, err := attributevalue.MarshalMap(customer)
	if err != nil {
		return fmt.Errorf("marshaling customer: %w", err)
	}

	_, err = s.client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})
	if err != nil {
		return fmt.Errorf("putting customer item: %w", err)
	}
	return nil
}

// GetCustomer retrieves a customer by ID. Returns nil if the customer does not exist.
func (s *DynamoCustomerStore) GetCustomer(id string) (*models.Customer, error) {
	result, err := s.client.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("getting customer item: %w", err)
	}
	if result.Item == nil {
		return nil, nil
	}

	var customer models.Customer
	if err := attributevalue.UnmarshalMap(result.Item, &customer); err != nil {
		return nil, fmt.Errorf("unmarshaling customer: %w", err)
	}
	return &customer, nil
}

// ListCustomers retrieves all customers from DynamoDB using a Scan operation.
func (s *DynamoCustomerStore) ListCustomers() ([]models.Customer, error) {
	result, err := s.client.Scan(context.Background(), &dynamodb.ScanInput{
		TableName: aws.String(s.tableName),
	})
	if err != nil {
		return nil, fmt.Errorf("scanning customers table: %w", err)
	}

	var customers []models.Customer
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &customers); err != nil {
		return nil, fmt.Errorf("unmarshaling customers: %w", err)
	}
	return customers, nil
}
