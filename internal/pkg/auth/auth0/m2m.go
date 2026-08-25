// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

package auth0

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// ClientCredentials describes an Auth0 machine-to-machine application: the tenant that issues the
// tokens, the API the tokens are meant for, and the application's own credentials.
type ClientCredentials struct {
	// Domain is the Auth0 tenant domain, e.g. "mycompany.auth0.com". A scheme must not be included.
	Domain string
	// Audience is the identifier of the Auth0 API the token is requested for, e.g.
	// "https://image-factory.example.com". Auth0 rejects a client credentials grant without one.
	Audience string
	// ClientID is the Auth0 application's client ID.
	ClientID string
	// ClientSecret is the Auth0 application's client secret.
	ClientSecret string
}

// Validate reports whether every field needed for a client credentials grant is set.
func (c ClientCredentials) Validate() error {
	var errs []error

	if c.Domain == "" {
		errs = append(errs, errors.New("domain is not set"))
	}

	if c.Audience == "" {
		errs = append(errs, errors.New("audience is not set"))
	}

	if c.ClientID == "" {
		errs = append(errs, errors.New("client ID is not set"))
	}

	if c.ClientSecret == "" {
		errs = append(errs, errors.New("client secret is not set"))
	}

	return errors.Join(errs...)
}

// M2MToken is a machine-to-machine access token issued by Auth0.
type M2MToken struct {
	// AccessToken is the token itself, to be sent as a bearer token.
	AccessToken string
	// IssuedAt is when the token was requested. Auth0 reports a token's lifetime as a duration
	// relative to the response, so the issue time has to be recorded on our side to recover it.
	IssuedAt time.Time
	// ExpiresAt is when the token stops being accepted.
	ExpiresAt time.Time
	// TokenType is the type Auth0 returned, normally "Bearer".
	TokenType string
}

// Lifetime returns the validity period of the token.
func (t M2MToken) Lifetime() time.Duration {
	return t.ExpiresAt.Sub(t.IssuedAt)
}

// TokenIssuer issues machine-to-machine access tokens.
//
// It is an interface so that callers which only need a token — the token rotation controller, most
// of all — can be tested without reaching an Auth0 tenant.
type TokenIssuer interface {
	IssueToken(ctx context.Context) (M2MToken, error)
}

// M2MTokenIssuer issues machine-to-machine access tokens through the Auth0 client credentials grant.
//
// It deliberately does not cache: each call performs a token request. Callers that need a token to
// outlive a single process are expected to persist it themselves.
type M2MTokenIssuer struct {
	config clientcredentials.Config
	now    func() time.Time
}

// NewM2MTokenIssuer creates a token issuer for the given Auth0 machine-to-machine application.
func NewM2MTokenIssuer(creds ClientCredentials) (*M2MTokenIssuer, error) {
	if err := creds.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Auth0 client credentials: %w", err)
	}

	tokenURL, err := url.Parse("https://" + creds.Domain + "/oauth/token")
	if err != nil {
		return nil, fmt.Errorf("invalid Auth0 domain %q: %w", creds.Domain, err)
	}

	return &M2MTokenIssuer{
		config: clientcredentials.Config{
			ClientID:     creds.ClientID,
			ClientSecret: creds.ClientSecret,
			TokenURL:     tokenURL.String(),
			// Auth0 keys the issued token's claims off the audience, and refuses the grant when it
			// is missing, so it always travels with the request.
			EndpointParams: url.Values{"audience": {creds.Audience}},
			// Auth0 documents the client credentials grant with the credentials in the request
			// body. Saying so up front skips the auto-detection probe, which would otherwise spend
			// a round trip trying HTTP Basic auth first.
			AuthStyle: oauth2.AuthStyleInParams,
		},
		now: time.Now,
	}, nil
}

// IssueToken requests a new access token from Auth0.
func (i *M2MTokenIssuer) IssueToken(ctx context.Context) (M2MToken, error) {
	issuedAt := i.now()

	token, err := i.config.Token(ctx)
	if err != nil {
		return M2MToken{}, fmt.Errorf("failed to request Auth0 machine-to-machine token: %w", err)
	}

	if token.AccessToken == "" {
		return M2MToken{}, errors.New("Auth0 returned an empty access token")
	}

	if token.Expiry.IsZero() {
		return M2MToken{}, errors.New("Auth0 returned a token without an expiration")
	}

	tokenType := token.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	return M2MToken{
		AccessToken: token.AccessToken,
		TokenType:   tokenType,
		IssuedAt:    issuedAt,
		ExpiresAt:   token.Expiry,
	}, nil
}
