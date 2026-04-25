package oauth

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

type manager struct {
	config *oauth2.Config
}

// exchangeCode demonstrates correct error wrapping with context.
func (m *manager) exchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := m.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange OAuth code: %w", err)
	}
	return token, nil
}
