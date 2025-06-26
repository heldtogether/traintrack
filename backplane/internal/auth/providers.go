package auth

import (
	"context"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
)

func VerifyToken(ctx context.Context, rawToken string) (*oidc.IDToken, error) {

	config, err := LoadConfig(DefaultConfigPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't load config: %s", err.Error())
	}

	var provider, _ = oidc.NewProvider(
		context.Background(),
		config.AuthURL,
	)

	var verifier = provider.Verifier(&oidc.Config{
		ClientID: config.Name,
		// ☝️ actually wants audience in this case because authorized party (azp)
		// is provided by Auth0. go-oidc doesn't seem to support verifying
		// depending on which claims are returned.
	})

	if verifier == nil {
		return nil, fmt.Errorf("verifier not initialized, call InitAuth first")
	}

	idToken, err := verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	// Optional: extract and inspect claims
	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse token claims: %w", err)
	}

	return idToken, nil
}
