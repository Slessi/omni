// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

package omni

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/cosi-project/runtime/pkg/controller"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/siderolabs/omni/client/api/omni/specs"
	"github.com/siderolabs/omni/client/pkg/omni/resources/omni"
	"github.com/siderolabs/omni/internal/pkg/imagefactory/tokens"
)

const (
	// imageFactoryTokenRetryInterval is how long to wait before retrying after a token request failed.
	//
	// A failure is not urgent: tokens are replaced halfway through their lifetime, so the token in
	// state is still valid for about as long as it has already lived.
	imageFactoryTokenRetryInterval = 30 * time.Second

	// imageFactoryTokenMaxSleep caps how long the controller sleeps between reconciliations.
	//
	// Auth0 machine-to-machine tokens live for a day by default, so the computed wake-up is far
	// away; waking up more often bounds how long a token stays stale after the wall clock jumps or
	// the process is suspended.
	imageFactoryTokenMaxSleep = time.Hour
)

// ImageFactoryTokenController issues and rotates the machine-to-machine access tokens Omni presents
// to the image factories that are configured with Auth0 credentials.
//
// The set of factories comes from Omni's configuration rather than from a resource, so the
// controller has no inputs: it reconciles on its own schedule, derived from the lifetimes of the
// tokens it has issued.
type ImageFactoryTokenController struct {
	issuers *tokens.Issuers
}

// NewImageFactoryTokenController creates a new ImageFactoryTokenController.
func NewImageFactoryTokenController(issuers *tokens.Issuers) *ImageFactoryTokenController {
	return &ImageFactoryTokenController{issuers: issuers}
}

// Name implements controller.Controller interface.
func (ctrl *ImageFactoryTokenController) Name() string {
	return "ImageFactoryTokenController"
}

// Inputs implements controller.Controller interface.
func (ctrl *ImageFactoryTokenController) Inputs() []controller.Input {
	return nil
}

// Outputs implements controller.Controller interface.
func (ctrl *ImageFactoryTokenController) Outputs() []controller.Output {
	return []controller.Output{
		{
			Type: omni.ImageFactoryTokenType,
			Kind: controller.OutputExclusive,
		},
	}
}

// Run implements controller.Controller interface.
func (ctrl *ImageFactoryTokenController) Run(ctx context.Context, r controller.Runtime, logger *zap.Logger) error {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.EventCh(): // no inputs, so nothing to react to
			continue
		case <-timer.C:
		}

		sleep, err := ctrl.reconcile(ctx, r, logger)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			// A failing token endpoint must not take the controller down: the tokens already in
			// state stay usable, and the next attempt may well succeed.
			logger.Error("failed to reconcile image factory tokens", zap.Error(err), zap.Duration("retry_in", imageFactoryTokenRetryInterval))

			sleep = imageFactoryTokenRetryInterval
		}

		timer.Reset(sleep)
	}
}

// reconcile brings the ImageFactoryToken resources in line with the configured factories, and
// returns how long to wait before the next reconciliation.
func (ctrl *ImageFactoryTokenController) reconcile(ctx context.Context, r controller.Runtime, logger *zap.Logger) (time.Duration, error) {
	factoryURLs := ctrl.issuers.FactoryURLs()

	if err := ctrl.pruneTokens(ctx, r, factoryURLs); err != nil {
		return 0, err
	}

	sleep := imageFactoryTokenMaxSleep

	var errs []error

	for _, factoryURL := range factoryURLs {
		refreshIn, err := ctrl.reconcileFactory(ctx, r, logger, factoryURL)
		if err != nil {
			errs = append(errs, fmt.Errorf("image factory %q: %w", factoryURL, err))

			continue
		}

		sleep = min(sleep, refreshIn)
	}

	if err := errors.Join(errs...); err != nil {
		return 0, err
	}

	return sleep, nil
}

// reconcileFactory makes sure the factory has a usable token, and returns how long that token stays
// usable before it should be replaced.
func (ctrl *ImageFactoryTokenController) reconcileFactory(ctx context.Context, r controller.Runtime, logger *zap.Logger, factoryURL string) (time.Duration, error) {
	issuer := ctrl.issuers.ForURL(factoryURL)
	if issuer == nil {
		return 0, errors.New("no token issuer is configured")
	}

	existing, err := safe.ReaderGetByID[*omni.ImageFactoryToken](ctx, r, factoryURL)
	if err != nil && !state.IsNotFoundError(err) {
		return 0, err
	}

	if existing != nil {
		if refreshIn := specRefreshDelay(existing.TypedSpec().Value); refreshIn > 0 && tokenMatchesIssuer(existing.TypedSpec().Value, issuer) {
			return refreshIn, nil
		}
	}

	token, err := issuer.IssueToken(ctx)
	if err != nil {
		return 0, err
	}

	if err = safe.WriterModify(ctx, r, omni.NewImageFactoryToken(factoryURL), func(res *omni.ImageFactoryToken) error {
		res.TypedSpec().Value.AccessToken = token.AccessToken
		res.TypedSpec().Value.TokenType = token.TokenType
		res.TypedSpec().Value.IssuedAt = timestamppb.New(token.IssuedAt)
		res.TypedSpec().Value.ExpiresAt = timestamppb.New(token.ExpiresAt)
		res.TypedSpec().Value.ClientId = issuer.ClientID()
		res.TypedSpec().Value.Audience = issuer.Audience()

		return nil
	}); err != nil {
		return 0, err
	}

	logger.Info(
		"issued image factory token",
		zap.String("factory_url", factoryURL),
		zap.Duration("lifetime", token.Lifetime()),
		zap.Time("expires_at", token.ExpiresAt),
	)

	refreshIn := refreshDelay(token.IssuedAt, token.ExpiresAt)

	if refreshIn <= 0 {
		// The issuer handed back a token that is already halfway through its life. Reporting it as
		// an error puts the retry interval between us and the next attempt instead of spinning.
		return 0, fmt.Errorf("issued token is valid for %s, which is too short to use", token.Lifetime())
	}

	return refreshIn, nil
}

// pruneTokens destroys the tokens of factories that are no longer configured with Auth0 credentials.
func (ctrl *ImageFactoryTokenController) pruneTokens(ctx context.Context, r controller.Runtime, factoryURLs []string) error {
	existing, err := safe.ReaderListAll[*omni.ImageFactoryToken](ctx, r)
	if err != nil {
		return err
	}

	for token := range existing.All() {
		if slices.Contains(factoryURLs, token.Metadata().ID()) {
			continue
		}

		if err = r.Destroy(ctx, token.Metadata()); err != nil && !state.IsNotFoundError(err) {
			return fmt.Errorf("failed to destroy the token of image factory %q: %w", token.Metadata().ID(), err)
		}
	}

	return nil
}

// tokenMatchesIssuer reports whether the persisted token was issued under the identity the issuer
// uses now. Reconfiguring a factory's Auth0 application otherwise leaves the token issued to the
// previous one in place until it expires on its own.
func tokenMatchesIssuer(spec *specs.ImageFactoryTokenSpec, issuer tokens.Issuer) bool {
	return spec.GetClientId() == issuer.ClientID() && spec.GetAudience() == issuer.Audience()
}

// refreshDelay returns how long a token issued at issuedAt and expiring at expiresAt stays in use:
// half of its lifetime, which leaves the remaining half as the window to recover from a failing
// token endpoint. A non-positive result means the token is due to be replaced now.
func refreshDelay(issuedAt, expiresAt time.Time) time.Duration {
	if !expiresAt.After(issuedAt) {
		return 0
	}

	return time.Until(issuedAt.Add(expiresAt.Sub(issuedAt) / 2))
}

// specRefreshDelay is refreshDelay for a persisted token. A token whose lifetime was not fully
// recorded is due immediately, as is one that somehow carries no token at all.
func specRefreshDelay(spec *specs.ImageFactoryTokenSpec) time.Duration {
	if spec.GetAccessToken() == "" || spec.GetIssuedAt() == nil || spec.GetExpiresAt() == nil {
		return 0
	}

	return refreshDelay(spec.GetIssuedAt().AsTime(), spec.GetExpiresAt().AsTime())
}
