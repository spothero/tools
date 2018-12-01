package core

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
)

// AWSConfig allows for easy loading of AWS configuration
type AWSConfig struct {
	Region  string
	Profile string
}

// LoadAWSConfig loads AWS configuration and credentials from the default
// locations (environment variables and ~/.aws/credentials) and optionally
// attempts to load credentials for the specified profile and in the specified
// region
func (a AWSConfig) LoadAWSConfig() (aws.Config, error) {
	awsConfigs := make([]external.Config, 0, 2)
	if a.Region != "" {
		awsConfigs = append(awsConfigs, external.WithRegion(a.Region))
	}
	if a.Profile != "" {
		awsConfigs = append(awsConfigs, external.WithSharedConfigProfile(a.Profile))
	}
	return external.LoadDefaultAWSConfig(awsConfigs...)
}
