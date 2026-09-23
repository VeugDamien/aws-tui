package awsclient

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

// accountAliasTimeout bounds the IAM call so the UI never hangs.
const accountAliasTimeout = 6 * time.Second

// AccountAlias returns the IAM account alias (the friendly name shown in the AWS
// console) for the active account, or an empty string if none is set. A nil error
// with an empty string means "no alias configured". Callers should treat any error
// as non-fatal and fall back to the account number.
func AccountAlias(ctx context.Context, cfg aws.Config) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, accountAliasTimeout)
	defer cancel()

	client := iam.NewFromConfig(cfg)
	out, err := client.ListAccountAliases(ctx, &iam.ListAccountAliasesInput{})
	if err != nil {
		return "", err
	}
	if len(out.AccountAliases) == 0 {
		return "", nil
	}
	return out.AccountAliases[0], nil
}
