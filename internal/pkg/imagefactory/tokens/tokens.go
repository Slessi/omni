// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

// Package tokens issues the machine-to-machine access tokens Omni presents to the image factories
// it is configured to talk to.
package tokens

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/siderolabs/omni/internal/pkg/auth/auth0"
	"github.com/siderolabs/omni/internal/pkg/config"
)

// Issuer issues machine-to-machine access tokens for a single image factory, and reports the
// identity the tokens are issued under.
type Issuer interface {
	auth0.TokenIssuer

	// ClientID is the Auth0 client ID the tokens are issued to.
	ClientID() string
	// Audience is the Auth0 API identifier the tokens are issued for.
	Audience() string
}

// Issuers is the set of token issuers Omni has, one per configured image factory that carries Auth0
// machine-to-machine credentials. Factories configured without them are simply absent, which is how
// a caller learns that a factory needs no token.
//
// Keys are factory base URLs with any trailing slash stripped, matching the IDs of the
// ImageFactoryAuth and ImageFactoryToken resources.
type Issuers struct {
	issuers map[string]Issuer
}

type issuer struct {
	*auth0.M2MTokenIssuer

	clientID string
	audience string
}

func (i issuer) ClientID() string { return i.clientID }

func (i issuer) Audience() string { return i.audience }

// NewIssuers builds the token issuers for the configured image factories.
//
// A factory without an Auth0 client ID gets no issuer: such a factory either needs no
// authentication at all, or authenticates with the basic auth credentials tracked separately on its
// ImageFactoryAuth resource.
func NewIssuers(registries *config.Registries) (*Issuers, error) {
	factories := []config.Factory{registries.GetPrimaryFactory()}

	if secondary, ok := registries.GetSecondaryFactory(); ok {
		factories = append(factories, secondary)
	}

	issuers := map[string]Issuer{}

	for _, factory := range factories {
		if factory.Auth0.GetClientID() == "" {
			continue
		}

		creds := auth0.ClientCredentials{
			Domain:       factory.Auth0.GetDomain(),
			Audience:     factory.Auth0.GetAudience(),
			ClientID:     factory.Auth0.GetClientID(),
			ClientSecret: factory.Auth0.GetClientSecret(),
		}

		m2m, err := auth0.NewM2MTokenIssuer(creds)
		if err != nil {
			return nil, fmt.Errorf("failed to build the Auth0 token issuer for image factory %q: %w", factory.GetUrl(), err)
		}

		issuers[NormalizeFactoryURL(factory.GetUrl())] = issuer{
			M2MTokenIssuer: m2m,
			clientID:       creds.ClientID,
			audience:       creds.Audience,
		}
	}

	return &Issuers{issuers: issuers}, nil
}

// NewIssuersFromMap builds an Issuers out of already-constructed issuers. It exists for tests and
// for callers that source their issuers from somewhere other than the Omni configuration.
func NewIssuersFromMap(issuers map[string]Issuer) *Issuers {
	normalized := make(map[string]Issuer, len(issuers))

	for factoryURL, i := range issuers {
		normalized[NormalizeFactoryURL(factoryURL)] = i
	}

	return &Issuers{issuers: normalized}
}

// FactoryURLs returns the normalized URLs of the factories that have an issuer, in a stable order.
func (i *Issuers) FactoryURLs() []string {
	if i == nil {
		return nil
	}

	return slices.Sorted(maps.Keys(i.issuers))
}

// ForURL returns the issuer configured for the factory at the given URL, or nil when that factory
// has no Auth0 machine-to-machine credentials.
//
// The URL is normalized before lookup, so it may be passed in as it was configured.
func (i *Issuers) ForURL(factoryURL string) Issuer {
	if i == nil {
		return nil
	}

	return i.issuers[NormalizeFactoryURL(factoryURL)]
}

// IssueToken issues a new token for the factory at the given URL.
func (i *Issuers) IssueToken(ctx context.Context, factoryURL string) (auth0.M2MToken, error) {
	factoryIssuer := i.ForURL(factoryURL)
	if factoryIssuer == nil {
		return auth0.M2MToken{}, fmt.Errorf("no Auth0 machine-to-machine credentials are configured for image factory %q", factoryURL)
	}

	return factoryIssuer.IssueToken(ctx)
}

// NormalizeFactoryURL strips a trailing slash so that factory URLs coming from different sources
// compare equal, and so that they match the resource IDs keyed off them.
func NormalizeFactoryURL(factoryURL string) string {
	return strings.TrimRight(factoryURL, "/")
}
