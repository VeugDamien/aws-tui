// Package awsclient provides AWS access through the AWS SDK for Go v2.
package awsclient

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// LoadConfig builds an aws.Config for the given profile and region. The SDK reads
// ~/.aws/config, the SSO cache and runs credential_process as needed; no manual
// credential handling is performed here.
func LoadConfig(ctx context.Context, profile, region string) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithSharedConfigProfile(profile),
	}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("chargement de la configuration AWS (profil %q): %w", profile, err)
	}
	return cfg, nil
}
