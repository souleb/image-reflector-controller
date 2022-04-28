package azure

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

type FakeTokenCredential struct {
	Token     string
	ExpiresOn time.Time
	Err       error
}

func (tc *FakeTokenCredential) GetToken(ctx context.Context, options policy.TokenRequestOptions) (*azcore.AccessToken, error) {
	if tc.Err != nil {
		return nil, tc.Err
	}
	return &azcore.AccessToken{Token: tc.Token, ExpiresOn: tc.ExpiresOn}, nil
}
