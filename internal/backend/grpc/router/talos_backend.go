// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

package router

import (
	"context"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/siderolabs/gen/xslices"
	"github.com/siderolabs/go-api-signature/pkg/message"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/siderolabs/omni/internal/backend/dns"
	"github.com/siderolabs/omni/internal/pkg/auth"
	"github.com/siderolabs/omni/internal/pkg/auth/accesspolicy"
	"github.com/siderolabs/omni/internal/pkg/auth/talosaccess"
	"github.com/siderolabs/omni/internal/pkg/ctxstore"
	"github.com/siderolabs/omni/internal/pkg/grpcutil"
)

// TalosBackend implements a backend (proxying directly to a single Talos node over SideroLink).
type TalosBackend struct {
	nodeResolver NodeResolver
	omniState    state.State
	conn         *grpc.ClientConn
	verifier     grpc.UnaryServerInterceptor
	talosAuditor TalosAuditor
	name         string
	clusterID    string
	authEnabled  bool
}

// NewTalosBackend builds new Talos API backend.
func NewTalosBackend(
	name, clusterID string,
	nodeResolver NodeResolver,
	conn *grpc.ClientConn,
	authEnabled bool,
	verifier grpc.UnaryServerInterceptor,
	st state.State,
	talosAuditor TalosAuditor,
) *TalosBackend {
	return &TalosBackend{
		name:         name,
		clusterID:    clusterID,
		nodeResolver: nodeResolver,
		conn:         conn,
		authEnabled:  authEnabled,
		verifier:     verifier,
		omniState:    st,
		talosAuditor: talosAuditor,
	}
}

func (backend *TalosBackend) String() string {
	return backend.name
}

// GetConnection returns a grpc connection to the backend.
func (backend *TalosBackend) GetConnection(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(nil)
	}

	// we can't use regular gRPC server interceptors here, as proxy interface is a bit different

	// prepare context values for the verifier
	ctx = ctxstore.WithValue(ctx, auth.EnabledAuthContextKey{Enabled: backend.authEnabled})
	ctx = ctxstore.WithValue(ctx, auth.GRPCMessageContextKey{Message: message.NewGRPC(md, fullMethodName)})

	grpcutil.SetShouldLog(ctx, "talos-backend")

	if backend.clusterID != "" {
		grpcutil.AddLogPair(ctx, "cluster", backend.clusterID)
	}

	// perform authentication, result of the authentication should be written to ctx
	_, err := backend.verifier(
		ctx, nil, nil,
		func(innerCtx context.Context, _ any) (any, error) {
			// save enhanced context
			//nolint:fatcontext
			ctx = innerCtx

			return nil, nil //nolint:nilnil
		},
	)
	if err != nil {
		// authentication failed
		return ctx, nil, err
	}

	ctx, err = accesspolicy.ApplyClusterAccessPolicy(ctx, backend.clusterID, backend.omniState)
	if err != nil {
		return ctx, nil, err
	}

	nodes, err := resolveNodes(backend.nodeResolver, md)
	if err != nil {
		return ctx, nil, err
	}

	if backend.talosAuditor != nil {
		if err = backend.talosAuditor.AuditTalosAccess(ctx, strings.TrimLeft(fullMethodName, "/"), backend.clusterID, getNodeID(md)); err != nil {
			return ctx, nil, err
		}
	}

	hasModifyAccess, err := talosaccess.Check(ctx, backend.clusterID)
	if err != nil {
		return ctx, nil, err
	}

	md = md.Copy()

	// Always strip the "node" header — Omni has already resolved and routed directly.
	md.Delete(nodeHeaderKey)

	// Preserve the "nodes" header (rewritten with resolved node addresses) when
	// the original request had "nodes". Talos apid uses it for One2Many fan-out (2+ nodes)
	// or loopback (1 node pointing to self), which sets Metadata.Hostname in the response.
	// This preserves the response shape that talosctl and other API consumers expect.
	// GetAddress() returns the cluster-internal IP when available, falling back to the
	// SideroLink management address during early bootstrap before NodeIPs are populated.
	if len(md.Get(nodesHeaderKey)) > 0 {
		addresses := xslices.Map(nodes, func(info dns.Info) string {
			return info.GetAddress()
		})

		setHeaderData(ctx, md, nodesHeaderKey, addresses...)
	}

	backend.setRoleHeaders(ctx, md, fullMethodName, nodes, hasModifyAccess)

	outCtx := metadata.NewOutgoingContext(ctx, md)

	return outCtx, backend.conn, nil
}

func (backend *TalosBackend) setRoleHeaders(ctx context.Context, md metadata.MD, fullMethodName string, nodes []dns.Info, hasModifyAccess bool) {
	roles := talosaccess.Roles(fullMethodName, backend.minTalosVersion(nodes), hasModifyAccess)

	setHeaderData(ctx, md, constants.APIAuthzRoleMetadataKey, roles.Strings()...)
}

func (backend *TalosBackend) minTalosVersion(nodes []dns.Info) *semver.Version {
	var ver *semver.Version

	for _, node := range nodes {
		nodeVer := takePtr(semver.ParseTolerant(node.TalosVersion))
		if nodeVer != nil && (ver == nil || nodeVer.LT(*ver)) {
			ver = nodeVer
		}
	}

	return ver
}

func takePtr[T any](v T, err error) *T {
	if err != nil {
		return nil
	}

	return &v
}

// AppendInfo is called to enhance response from the backend with additional data.
func (backend *TalosBackend) AppendInfo(_ bool, resp []byte) ([]byte, error) {
	return resp, nil
}

// BuildError is called to convert error from upstream into response field.
func (backend *TalosBackend) BuildError(bool, error) ([]byte, error) {
	return nil, nil
}

func setHeaderData(ctx context.Context, md metadata.MD, k string, v ...string) {
	if len(v) == 0 {
		return
	}

	md.Set(k, v...)

	if len(v) == 1 {
		grpcutil.AddLogPair(ctx, k, v[0])
	} else {
		grpcutil.AddLogPair(ctx, k, v)
	}
}
