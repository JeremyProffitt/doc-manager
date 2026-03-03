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

// DynamoFormStore implements FormStore using DynamoDB.
type DynamoFormStore struct {
	client    *dynamodb.Client
	tableName string
}

// NewFormStore creates a new DynamoDB-backed FormStore.
func NewFormStore(client *dynamodb.Client, tableName string) *DynamoFormStore {
	return &DynamoFormStore{
		client:    client,
		tableName: tableName,
	}
}

// CreateForm creates a new form in DynamoDB with a condition to prevent overwrites.
func (s *DynamoFormStore) CreateForm(form *models.Form) error {
	item, err := attributevalue.MarshalMap(form)
	if err != nil {
		return fmt.Errorf("marshaling form: %w", err)
	}

	_, err = s.client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})
	if err != nil {
		return fmt.Errorf("putting form item: %w", err)
	}
	return nil
}

// GetForm retrieves a form by ID. Returns nil if the form does not exist.
func (s *DynamoFormStore) GetForm(id string) (*models.Form, error) {
	result, err := s.client.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("getting form item: %w", err)
	}
	if result.Item == nil {
		return nil, nil
	}

	var form models.Form
	if err := attributevalue.UnmarshalMap(result.Item, &form); err != nil {
		return nil, fmt.Errorf("unmarshaling form: %w", err)
	}
	return &form, nil
}

// ListForms retrieves all forms for a given user using a GSI query on userId.
func (s *DynamoFormStore) ListForms(userId string) ([]models.Form, error) {
	result, err := s.client.Query(context.Background(), &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		IndexName:              aws.String("userId-index"),
		KeyConditionExpression: aws.String("userId = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userId},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("querying forms by userId: %w", err)
	}

	var forms []models.Form
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &forms); err != nil {
		return nil, fmt.Errorf("unmarshaling forms: %w", err)
	}
	return forms, nil
}

// UpdateForm overwrites an existing form in DynamoDB.
func (s *DynamoFormStore) UpdateForm(form *models.Form) error {
	item, err := attributevalue.MarshalMap(form)
	if err != nil {
		return fmt.Errorf("marshaling form: %w", err)
	}

	_, err = s.client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("updating form item: %w", err)
	}
	return nil
}

// DeleteForm removes a form by ID from DynamoDB.
func (s *DynamoFormStore) DeleteForm(id string) error {
	_, err := s.client.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return fmt.Errorf("deleting form item: %w", err)
	}
	return nil
}
