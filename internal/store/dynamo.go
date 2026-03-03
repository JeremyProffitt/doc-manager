package store

import (
	"context"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	appconfig "github.com/JeremyProffitt/doc-manager/internal/config"
)

// NewDynamoClient initializes an AWS DynamoDB client using the
// application configuration.
func NewDynamoClient(cfg *appconfig.Config) (*dynamodb.Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(cfg.AWSRegion),
	)
	if err != nil {
		return nil, err
	}
	return dynamodb.NewFromConfig(awsCfg), nil
}
