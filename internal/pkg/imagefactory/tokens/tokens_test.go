// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

package tokens_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/siderolabs/omni/internal/pkg/config"
	"github.com/siderolabs/omni/internal/pkg/imagefactory/tokens"
)

func factory(url string, auth0 map[string]string) config.Factory {
	f := config.Factory{}
	f.SetUrl(url)

	if domain, ok := auth0["domain"]; ok {
		f.Auth0.SetDomain(domain)
	}

	if audience, ok := auth0["audience"]; ok {
		f.Auth0.SetAudience(audience)
	}

	if clientID, ok := auth0["client_id"]; ok {
		f.Auth0.SetClientID(clientID)
	}

	if clientSecret, ok := auth0["client_secret"]; ok {
		f.Auth0.SetClientSecret(clientSecret)
	}

	return f
}

func completeAuth0(clientID string) map[string]string {
	return map[string]string{
		"domain":        "tenant.example.com",
		"audience":      "https://image-factory.example.com",
		"client_id":     clientID,
		"client_secret": "client-secret",
	}
}

func TestNewIssuers(t *testing.T) {
	t.Parallel()

	// The primary factory carries Auth0 credentials, the secondary does not. The trailing slash on
	// the primary URL is what a configured value may well look like.
	registries := &config.Registries{
		Factories: config.Factories{
			Primary:   factory("https://factory.example.com/", completeAuth0("primary-client")),
			Secondary: factory("https://secondary.example.com", nil),
		},
	}

	issuers, err := tokens.NewIssuers(registries)
	require.NoError(t, err)

	assert.Equal(t, []string{"https://factory.example.com"}, issuers.FactoryURLs())

	primary := issuers.ForURL("https://factory.example.com/")
	require.NotNil(t, primary)
	assert.Equal(t, "primary-client", primary.ClientID())
	assert.Equal(t, "https://image-factory.example.com", primary.Audience())

	assert.Nil(t, issuers.ForURL("https://secondary.example.com"))

	_, err = issuers.IssueToken(t.Context(), "https://secondary.example.com")
	require.ErrorContains(t, err, `no Auth0 machine-to-machine credentials are configured for image factory "https://secondary.example.com"`)
}

func TestNewIssuersBothFactories(t *testing.T) {
	t.Parallel()

	registries := &config.Registries{
		Factories: config.Factories{
			Primary:   factory("https://factory.example.com", completeAuth0("primary-client")),
			Secondary: factory("https://secondary.example.com", completeAuth0("secondary-client")),
		},
	}

	issuers, err := tokens.NewIssuers(registries)
	require.NoError(t, err)

	assert.Equal(t, []string{"https://factory.example.com", "https://secondary.example.com"}, issuers.FactoryURLs())
	assert.Equal(t, "secondary-client", issuers.ForURL("https://secondary.example.com").ClientID())
}

func TestNewIssuersIncompleteCredentials(t *testing.T) {
	t.Parallel()

	// The configuration schema rejects this combination, so reaching here means the credentials came
	// from somewhere that did not go through validation. Failing loudly beats silently issuing no token.
	auth0 := completeAuth0("primary-client")
	delete(auth0, "domain")

	registries := &config.Registries{
		Factories: config.Factories{
			Primary: factory("https://factory.example.com", auth0),
		},
	}

	_, err := tokens.NewIssuers(registries)
	require.ErrorContains(t, err, `failed to build the Auth0 token issuer for image factory "https://factory.example.com"`)
	require.ErrorContains(t, err, "domain is not set")
}

func TestIssuersNil(t *testing.T) {
	t.Parallel()

	var issuers *tokens.Issuers

	assert.Empty(t, issuers.FactoryURLs())
	assert.Nil(t, issuers.ForURL("https://factory.example.com"))
}
