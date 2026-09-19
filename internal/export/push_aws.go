package export

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"launch-pg/internal/model"
)

const (
	OptRegion  = "region"
	OptProfile = "profile"
)

// SecretsManagerAPI is the subset of the AWS client the pusher uses.
type SecretsManagerAPI interface {
	CreateSecret(ctx context.Context, in *secretsmanager.CreateSecretInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
	PutSecretValue(ctx context.Context, in *secretsmanager.PutSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error)
}

// SecretsManagerFactory builds a client for a region/profile ("" = defaults).
type SecretsManagerFactory func(ctx context.Context, region, profile string) (SecretsManagerAPI, string, error)

// DefaultSecretsManagerFactory uses the standard AWS credential chain
// (env vars, shared config/credentials files, SSO, instance roles).
func DefaultSecretsManagerFactory(ctx context.Context, region, profile string) (SecretsManagerAPI, string, error) {
	var loaders []func(*awsconfig.LoadOptions) error
	if region != "" {
		loaders = append(loaders, awsconfig.WithRegion(region))
	}
	if profile != "" {
		loaders = append(loaders, awsconfig.WithSharedConfigProfile(profile))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, loaders...)
	if err != nil {
		return nil, "", fmt.Errorf("load AWS config: %w", err)
	}
	if cfg.Region == "" {
		return nil, "", errors.New("no AWS region: set region= or AWS_REGION")
	}
	return secretsmanager.NewFromConfig(cfg), cfg.Region, nil
}

// AWSSecretsPusher stores each credential as a JSON secret in AWS Secrets
// Manager, creating it or adding a new version if it exists.
type AWSSecretsPusher struct {
	newClient SecretsManagerFactory
}

func NewAWSSecretsPusher(factory SecretsManagerFactory) *AWSSecretsPusher {
	return &AWSSecretsPusher{newClient: factory}
}

func (*AWSSecretsPusher) Kind() string             { return "aws" }
func (*AWSSecretsPusher) Check(opts Options) error { return nil }

func (a *AWSSecretsPusher) Push(ctx context.Context, c model.Credential, opts Options) (string, error) {
	client, region, err := a.newClient(ctx, opts[OptRegion], opts[OptProfile])
	if err != nil {
		return "", err
	}
	name := Expand(opts.Get(OptName, DefaultPath), c)
	payload, err := json.Marshal(FieldMap(c))
	if err != nil {
		return "", err
	}
	value := string(payload)

	_, err = client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(name),
		SecretString: aws.String(value),
		Description:  aws.String("Postgres credentials for role " + c.User + " (managed by launch-pg)"),
	})
	var exists *types.ResourceExistsException
	if errors.As(err, &exists) {
		_, err = client.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput{
			SecretId:     aws.String(name),
			SecretString: aws.String(value),
		})
	}
	if err != nil {
		return "", fmt.Errorf("secrets manager: %w", err)
	}
	return fmt.Sprintf("aws:%s:%s", region, name), nil
}
