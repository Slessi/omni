// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

package omni_test

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/cosi-project/runtime/pkg/resource/rtestutils"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/siderolabs/omni/client/pkg/omni/resources/omni"
	omnictrl "github.com/siderolabs/omni/internal/backend/runtime/omni/controllers/omni"
	"github.com/siderolabs/omni/internal/pkg/auth/auth0"
	"github.com/siderolabs/omni/internal/pkg/imagefactory/tokens"
)

const (
	testFactoryURL          = "https://factory.example.com"
	testFactoryClientID     = "client-id"
	testFactoryAudience     = "https://image-factory.example.com"
	testFactoryTokenTimeout = 10 * time.Second

	// imageFactoryTokenControllerName is the owner stamped on the resources the controller writes.
	imageFactoryTokenControllerName = "ImageFactoryTokenController"
)

// fakeIssuer hands out tokens without talking to Auth0, numbering them so a test can tell one
// issued token from the next.
type fakeIssuer struct { //nolint:govet // grouped by role rather than by alignment
	clientID string
	audience string
	lifetime time.Duration

	mu    sync.Mutex
	count int
	err   error
}

func (i *fakeIssuer) IssueToken(context.Context) (auth0.M2MToken, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.err != nil {
		return auth0.M2MToken{}, i.err
	}

	i.count++

	issuedAt := time.Now()

	return auth0.M2MToken{
		AccessToken: tokenValue(i.count),
		TokenType:   "Bearer",
		IssuedAt:    issuedAt,
		ExpiresAt:   issuedAt.Add(i.lifetime),
	}, nil
}

func (i *fakeIssuer) ClientID() string { return i.clientID }

func (i *fakeIssuer) Audience() string { return i.audience }

func (i *fakeIssuer) issued() int {
	i.mu.Lock()
	defer i.mu.Unlock()

	return i.count
}

func (i *fakeIssuer) setError(err error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.err = err
}

func tokenValue(n int) string {
	return "token-" + strconv.Itoa(n)
}

type ImageFactoryTokenSuite struct {
	OmniSuite
}

func TestImageFactoryTokenSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(ImageFactoryTokenSuite))
}

func (suite *ImageFactoryTokenSuite) registerController(issuer *fakeIssuer) {
	suite.Require().NoError(suite.runtime.RegisterController(omnictrl.NewImageFactoryTokenController(
		tokens.NewIssuersFromMap(map[string]tokens.Issuer{testFactoryURL: issuer}),
	)))
}

// TestIssue covers the initial issuance for a configured factory.
func (suite *ImageFactoryTokenSuite) TestIssue() {
	ctx, cancel := context.WithTimeout(suite.ctx, testFactoryTokenTimeout)
	defer cancel()

	suite.startRuntime()

	issuer := &fakeIssuer{clientID: testFactoryClientID, audience: testFactoryAudience, lifetime: time.Hour}

	suite.registerController(issuer)

	rtestutils.AssertResources(
		ctx, suite.T(), suite.state, []string{testFactoryURL},
		func(res *omni.ImageFactoryToken, assert *assert.Assertions) {
			spec := res.TypedSpec().Value

			assert.Equal(tokenValue(1), spec.GetAccessToken())
			assert.Equal("Bearer", spec.GetTokenType())
			assert.Equal(testFactoryClientID, spec.GetClientId())
			assert.Equal(testFactoryAudience, spec.GetAudience())
			assert.WithinDuration(time.Now().Add(time.Hour), spec.GetExpiresAt().AsTime(), time.Minute)
		},
	)

	// A token good for another hour is left alone.
	suite.Never(func() bool { return issuer.issued() > 1 }, time.Second, 100*time.Millisecond)
}

// TestRotate covers replacing a token once it is halfway through its lifetime.
func (suite *ImageFactoryTokenSuite) TestRotate() {
	ctx, cancel := context.WithTimeout(suite.ctx, testFactoryTokenTimeout)
	defer cancel()

	suite.startRuntime()

	// Half of this lifetime has passed almost immediately, so the controller rotates right away.
	issuer := &fakeIssuer{clientID: testFactoryClientID, audience: testFactoryAudience, lifetime: 200 * time.Millisecond}

	suite.registerController(issuer)

	rtestutils.AssertResources(
		ctx, suite.T(), suite.state, []string{testFactoryURL},
		func(res *omni.ImageFactoryToken, assert *assert.Assertions) {
			assert.NotEqual(tokenValue(1), res.TypedSpec().Value.GetAccessToken())
		},
	)
}

// TestReissueOnCredentialChange covers a factory whose Auth0 application was reconfigured: the token
// issued to the previous client must not be left in place until it expires on its own.
func (suite *ImageFactoryTokenSuite) TestReissueOnCredentialChange() {
	ctx, cancel := context.WithTimeout(suite.ctx, testFactoryTokenTimeout)
	defer cancel()

	stale := omni.NewImageFactoryToken(testFactoryURL)
	stale.TypedSpec().Value.AccessToken = "stale-token"
	stale.TypedSpec().Value.TokenType = "Bearer"
	stale.TypedSpec().Value.IssuedAt = timestamppb.New(time.Now())
	stale.TypedSpec().Value.ExpiresAt = timestamppb.New(time.Now().Add(time.Hour))
	stale.TypedSpec().Value.ClientId = "previous-client-id"
	stale.TypedSpec().Value.Audience = testFactoryAudience

	// The controller only writes resources it owns, so the pre-existing one has to look like one it wrote.
	suite.Require().NoError(suite.state.Create(ctx, stale, state.WithCreateOwner(imageFactoryTokenControllerName)))

	suite.startRuntime()

	suite.registerController(&fakeIssuer{clientID: testFactoryClientID, audience: testFactoryAudience, lifetime: time.Hour})

	rtestutils.AssertResources(
		ctx, suite.T(), suite.state, []string{testFactoryURL},
		func(res *omni.ImageFactoryToken, assert *assert.Assertions) {
			assert.Equal(tokenValue(1), res.TypedSpec().Value.GetAccessToken())
			assert.Equal(testFactoryClientID, res.TypedSpec().Value.GetClientId())
		},
	)
}

// TestPrune covers a factory that is no longer configured with Auth0 credentials.
func (suite *ImageFactoryTokenSuite) TestPrune() {
	ctx, cancel := context.WithTimeout(suite.ctx, testFactoryTokenTimeout)
	defer cancel()

	orphaned := omni.NewImageFactoryToken("https://retired.example.com")
	orphaned.TypedSpec().Value.AccessToken = "orphaned-token"
	orphaned.TypedSpec().Value.IssuedAt = timestamppb.New(time.Now())
	orphaned.TypedSpec().Value.ExpiresAt = timestamppb.New(time.Now().Add(time.Hour))

	suite.Require().NoError(suite.state.Create(ctx, orphaned, state.WithCreateOwner(imageFactoryTokenControllerName)))

	suite.startRuntime()

	suite.registerController(&fakeIssuer{clientID: testFactoryClientID, audience: testFactoryAudience, lifetime: time.Hour})

	rtestutils.AssertNoResource[*omni.ImageFactoryToken](ctx, suite.T(), suite.state, orphaned.Metadata().ID())
	rtestutils.AssertResources(
		ctx, suite.T(), suite.state, []string{testFactoryURL},
		func(*omni.ImageFactoryToken, *assert.Assertions) {},
	)
}

// TestIssuerFailure covers a token endpoint that is failing: the controller keeps retrying instead of
// giving up, and the existing token is left in place meanwhile.
func (suite *ImageFactoryTokenSuite) TestIssuerFailure() {
	ctx, cancel := context.WithTimeout(suite.ctx, testFactoryTokenTimeout)
	defer cancel()

	suite.startRuntime()

	issuer := &fakeIssuer{clientID: testFactoryClientID, audience: testFactoryAudience, lifetime: time.Hour}
	issuer.setError(errors.New("Auth0 is unreachable"))

	suite.registerController(issuer)

	rtestutils.AssertNoResource[*omni.ImageFactoryToken](ctx, suite.T(), suite.state, testFactoryURL)

	issuer.setError(nil)

	rtestutils.AssertResources(
		ctx, suite.T(), suite.state, []string{testFactoryURL},
		func(res *omni.ImageFactoryToken, assert *assert.Assertions) {
			assert.Equal(tokenValue(1), res.TypedSpec().Value.GetAccessToken())
		},
	)
}
