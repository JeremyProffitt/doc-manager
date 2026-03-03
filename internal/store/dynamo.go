package store

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	appconfig "github.com/JeremyProffitt/doc-manager/internal/config"
)

// NewDynamoClient creates a new DynamoDB client using the application config.
func NewDynamoClient(cfg *appconfig.Config) (*dynamodb.Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	return dynamodb.NewFromConfig(awsCfg), nil
}

// NewDynamoClientFromConfig creates a new DynamoDB client from an existing
// AWS config. This is useful for testing with custom endpoints.
func NewDynamoClientFromConfig(awsCfg aws.Config) *dynamodb.Client {
	return dynamodb.NewFromConfig(awsCfg)
}
