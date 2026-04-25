package oauth

import (
	"context"

	"golang.org/x/oauth2"
)

type manager struct {
	config *oauth2.Config
}

// exchangeCodeBad returns the error without wrapping — loses call-site context.
func (m *manager) exchangeCodeBad(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := m.config.Exchange(ctx, code)
	// ruleid: slotly-003
	if err != nil {
		return nil, err
	}
	return token, nil
}
