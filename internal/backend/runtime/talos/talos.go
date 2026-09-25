// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

// Package talos implements the connector that can pull data from the Talos controller runtime.
package talos

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/cosi-project/runtime/api/v1alpha1"
	cosiresource "github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/resource/meta"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	clientconfig "github.com/siderolabs/talos/pkg/machinery/client/config"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	talosrole "github.com/siderolabs/talos/pkg/machinery/role"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/siderolabs/omni/client/api/common"
	"github.com/siderolabs/omni/client/pkg/cosi/labels"
	"github.com/siderolabs/omni/client/pkg/omni/resources/omni"
	pkgruntime "github.com/siderolabs/omni/client/pkg/runtime"
	"github.com/siderolabs/omni/internal/backend/logging"
	"github.com/siderolabs/omni/internal/backend/runtime"
	"github.com/siderolabs/omni/internal/backend/runtime/cosi"
	"github.com/siderolabs/omni/internal/pkg/auth/accesspolicy"
	"github.com/siderolabs/omni/internal/pkg/auth/talosaccess"
)

// Name talos runtime string id.
var Name = common.Runtime_Talos.String()

// Runtime implements runtime.Runtime for Talos resources.
type Runtime struct {
	clientFactory *ClientFactory
	logger        *zap.Logger
	accountName   string
	apiURL        string
}

// New creates a new Talos runtime.
func New(clientFactory *ClientFactory, logger *zap.Logger, accountName string, apiURL string) *Runtime {
	return &Runtime{
		clientFactory: clientFactory,
		logger:        logger.With(logging.Component("talos_runtime")),
		accountName:   accountName,
		apiURL:        apiURL,
	}
}

// Watch implements runtime.Runtime.
func (r *Runtime) Watch(ctx context.Context, events chan<- runtime.WatchResponse, setters ...runtime.QueryOption) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("watch panic")

			r.logger.Error("watch panicked", zap.Stack("stack"), zap.Error(err))
		}
	}()

	return r.watch(ctx, events, setters...)
}

func (r *Runtime) watch(ctx context.Context, events chan<- runtime.WatchResponse, setters ...runtime.QueryOption) error {
	opts := runtime.NewQueryOptions(setters...)

	var (
		c   *Client
		err error
	)

	switch len(opts.Machines) {
	case 0:
		ctx, c, err = r.callerClient(ctx, opts.Context, "", opts.Resource, v1alpha1.State_Watch_FullMethodName)
	case 1:
		ctx, c, err = r.callerClient(ctx, "", opts.Machines[0], opts.Resource, v1alpha1.State_Watch_FullMethodName)
	default:
		return errors.New("multiple machines are not supported for Watch")
	}

	if err != nil {
		return err
	}

	defer c.Close() //nolint:errcheck

	var queries []cosiresource.LabelQuery

	if len(opts.LabelSelectors) > 0 {
		queries, err = labels.ParseSelectors(opts.LabelSelectors)
		if err != nil {
			return err
		}
	}

	return cosi.WatchLegacy(
		ctx,
		c.COSI,
		cosiresource.NewMetadata(
			opts.Namespace,
			opts.Resource,
			opts.Name,
			cosiresource.VersionUndefined,
		),
		events,
		opts.TailEvents,
		queries,
	)
}

// Get implements runtime.Runtime.
func (r *Runtime) Get(ctx context.Context, setters ...runtime.QueryOption) (any, error) {
	opts := runtime.NewQueryOptions(setters...)

	var (
		c   *Client
		err error
	)

	switch len(opts.Machines) {
	case 0:
		ctx, c, err = r.callerClient(ctx, opts.Context, "", opts.Resource, v1alpha1.State_Get_FullMethodName)
	case 1:
		ctx, c, err = r.callerClient(ctx, "", opts.Machines[0], opts.Resource, v1alpha1.State_Get_FullMethodName)
	default:
		return nil, errors.New("multiple machines are not supported for Get")
	}

	if err != nil {
		return nil, err
	}

	defer c.Close() //nolint:errcheck

	res, err := c.COSI.Get(ctx, cosiresource.NewMetadata(opts.Namespace, opts.Resource, opts.Name, cosiresource.VersionUndefined))
	if err != nil {
		return nil, err
	}

	return runtime.NewResource(res)
}

// List implements runtime.Runtime.
func (r *Runtime) List(ctx context.Context, setters ...runtime.QueryOption) (runtime.ListResult, error) {
	opts := runtime.NewQueryOptions(setters...)

	if len(opts.Machines) == 0 {
		opts.Machines = []string{""}
	}

	var res []pkgruntime.ListItem

	for _, machine := range opts.Machines {
		items, err := r.list(ctx, machine, opts)
		if err != nil {
			return runtime.ListResult{}, err
		}

		res = append(res, items...)
	}

	return runtime.ListResult{
		Items: res,
		Total: len(res),
	}, nil
}

// list lists the resources on a single machine, or on the cluster if the machine is empty.
func (r *Runtime) list(ctx context.Context, machine string, opts *runtime.QueryOptions) ([]pkgruntime.ListItem, error) {
	var (
		c   *Client
		err error
	)

	if machine == "" {
		ctx, c, err = r.callerClient(ctx, opts.Context, "", opts.Resource, v1alpha1.State_List_FullMethodName)
	} else {
		ctx, c, err = r.callerClient(ctx, "", machine, opts.Resource, v1alpha1.State_List_FullMethodName)
	}

	if err != nil {
		return nil, err
	}

	defer c.Close() //nolint:errcheck

	items, err := c.COSI.List(ctx, cosiresource.NewMetadata(opts.Namespace, opts.Resource, "", cosiresource.VersionUndefined))
	if err != nil {
		return nil, err
	}

	var res []pkgruntime.ListItem

	for _, item := range items.Items {
		resource, err := runtime.NewResource(item)
		if err != nil {
			return nil, err
		}

		res = append(res, newItem(resource))
	}

	return res, nil
}

// Create implements runtime.Runtime.
func (r *Runtime) Create(context.Context, cosiresource.Resource, ...runtime.QueryOption) error {
	return status.Error(codes.Unimplemented, "not implemented")
}

// Update implements runtime.Runtime.
func (r *Runtime) Update(context.Context, cosiresource.Resource, ...runtime.QueryOption) error {
	return status.Error(codes.Unimplemented, "not implemented")
}

// Delete implements runtime.Runtime.
func (r *Runtime) Delete(context.Context, ...runtime.QueryOption) error {
	return status.Error(codes.Unimplemented, "not implemented")
}

// GetTalosconfigRaw returns raw talosconfig for the cluster (or for whole instance if the cluster is not specified).
func (r *Runtime) GetTalosconfigRaw(context *common.Context, identity string) ([]byte, error) {
	auth := clientconfig.Auth{}

	auth.SideroV1 = &clientconfig.SideroV1{
		Identity: identity,
	}

	contextName := r.accountName
	apiURL := r.apiURL

	cluster := ""

	if context != nil {
		cluster = context.Name
	}

	if cluster != "" {
		contextName = contextName + "-" + cluster
	}

	talosconfig := clientconfig.Config{
		Context: contextName,
		Contexts: map[string]*clientconfig.Context{
			contextName: {
				Endpoints: []string{
					apiURL,
				},
				Auth:    auth,
				Cluster: cluster,
			},
		},
	}

	return talosconfig.Bytes()
}

// callerClient returns a client for a request made on behalf of the caller in the context: either to the cluster, or
// directly to the machine when machineID is set.
//
// It checks that the caller may access the target, and returns the context carrying the Talos roles the caller acts
// with, mapped from its Omni role for the cluster the same way as for the Talos API requests Omni proxies.
//
// The returned client must be closed by the caller.
func (r *Runtime) callerClient(ctx context.Context, clusterID, machineID, resourceType, fullMethodName string) (context.Context, *Client, error) {
	var talosVersion string

	if machineID != "" {
		machineStatus, err := safe.StateGet[*omni.MachineStatus](ctx, r.clientFactory.omniState, omni.NewMachineStatus(machineID).Metadata())
		if err != nil {
			return nil, nil, err
		}

		// authorize against the cluster the machine belongs to, never one given by the caller. It is empty for
		// a machine outside of any cluster, which only the callers with modify access may reach.
		clusterID = machineStatus.TypedSpec().Value.Cluster
		talosVersion = machineStatus.TypedSpec().Value.TalosVersion
	} else {
		if clusterID == "" {
			return nil, nil, status.Error(codes.InvalidArgument, "either a cluster or a machine is required")
		}

		cluster, err := safe.StateGet[*omni.Cluster](ctx, r.clientFactory.omniState, omni.NewCluster(clusterID).Metadata())
		if err != nil && !state.IsNotFoundError(err) {
			return nil, nil, err
		}

		if cluster != nil {
			talosVersion = cluster.TypedSpec().Value.TalosVersion
		}
	}

	ctx, err := accesspolicy.ApplyClusterAccessPolicy(ctx, clusterID, r.clientFactory.omniState)
	if err != nil {
		return nil, nil, err
	}

	hasModifyAccess, err := talosaccess.Check(ctx, clusterID)
	if err != nil {
		return nil, nil, err
	}

	var minTalosVersion *semver.Version

	if v, parseErr := semver.ParseTolerant(talosVersion); parseErr == nil {
		minTalosVersion = &v
	}

	var c *Client

	if machineID != "" {
		c, err = r.clientFactory.GetImpersonatorForMachine(ctx, machineID)
	} else {
		c, err = r.clientFactory.GetImpersonatorForCluster(ctx, clusterID)
	}

	if err != nil {
		return nil, nil, err
	}

	// the factory reads the machine status again, the machine might have moved to another cluster since it was authorized
	if c.ClusterID() != "" && c.ClusterID() != clusterID {
		c.Close() //nolint:errcheck

		return nil, nil, status.Errorf(codes.Unavailable, "machine %q changed its cluster", machineID)
	}

	if c, err = r.checkConnected(ctx, c); err != nil {
		return nil, nil, err
	}

	roles := talosaccess.Roles(fullMethodName, minTalosVersion, hasModifyAccess)

	// A machine in maintenance mode is reached over its maintenance API, which grants every SideroLink peer, Omni
	// included, all the roles. Talos cannot enforce the caller's roles there, so enforce its sensitivity rule here.
	if c.ClusterID() == "" {
		if err = checkSensitivity(ctx, c, resourceType, roles); err != nil {
			c.Close() //nolint:errcheck

			return nil, nil, err
		}
	}

	roleStrings := roles.Strings()
	kv := make([]string, 0, 2*len(roleStrings))

	for _, talosRole := range roleStrings {
		kv = append(kv, constants.APIAuthzRoleMetadataKey, talosRole)
	}

	return metadata.AppendToOutgoingContext(ctx, kv...), c, nil
}

// checkSensitivity denies the access to a sensitive resource type unless the roles include os:admin, as Talos does.
func checkSensitivity(ctx context.Context, c *Client, resourceType string, roles talosrole.Set) error {
	rd, err := safe.StateGet[*meta.ResourceDefinition](ctx, c.COSI,
		cosiresource.NewMetadata(meta.NamespaceName, meta.ResourceDefinitionType, strings.ToLower(resourceType), cosiresource.VersionUndefined),
	)
	if err != nil {
		if state.IsNotFoundError(err) {
			return status.Errorf(codes.PermissionDenied, "resource type %q is not supported", resourceType)
		}

		return err
	}

	if rd.TypedSpec().Sensitivity == meta.Sensitive && !roles.Includes(talosrole.Admin) {
		return status.Errorf(codes.PermissionDenied, "access to the sensitive resource type %q is not permitted", resourceType)
	}

	return nil
}

// GetClientForCluster returns talos client for the cluster name.
//
// The client authenticates as `os:admin`, it must only be used for Omni's own calls, never on behalf of a caller.
//
// The returned client must be closed by the caller.
func (r *Runtime) GetClientForCluster(ctx context.Context, clusterName string) (*Client, error) {
	c, err := r.clientFactory.GetForCluster(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	return r.checkConnected(ctx, c)
}

// GetClientForMachine returns a Talos client connected directly to the given machine's SideroLink endpoint.
//
// Cluster membership is determined automatically from the machine's state.
//
// The client authenticates as `os:admin`, it must only be used for Omni's own calls, never on behalf of a caller.
//
// The returned client must be closed by the caller.
func (r *Runtime) GetClientForMachine(ctx context.Context, machineID string) (*Client, error) {
	c, err := r.clientFactory.GetForMachine(ctx, machineID)
	if err != nil {
		return nil, err
	}

	return r.checkConnected(ctx, c)
}

// checkConnected returns the client if its cluster or machine is reachable, otherwise it closes the client.
func (r *Runtime) checkConnected(ctx context.Context, c *Client) (*Client, error) {
	connected, err := c.Connected(ctx, r.clientFactory.omniState)
	if err != nil {
		c.Close() //nolint:errcheck

		return nil, err
	}

	if !connected {
		c.Close() //nolint:errcheck

		if c.clusterID == "" {
			return nil, fmt.Errorf("the machine %s is not reachable", c.machineID)
		}

		return nil, fmt.Errorf("the cluster %s is not reachable", c.clusterID)
	}

	return c, nil
}

type item struct {
	runtime.BasicItem[*runtime.Resource]
}

func (it *item) Field(name string) (string, bool) {
	val, ok := it.BasicItem.Field(name)
	if ok {
		return val, true
	}

	val, ok = runtime.ResourceField(it.BasicItem.Unwrap().Resource, name)
	if ok {
		return val, true
	}

	return "", false
}

func (it *item) Match(searchFor string) bool {
	return it.BasicItem.Match(searchFor) || runtime.MatchResource(it.BasicItem.Unwrap().Resource, searchFor)
}

func (it *item) Unwrap() any {
	return it.BasicItem.Unwrap()
}

func newItem(res *runtime.Resource) pkgruntime.ListItem {
	return &item{BasicItem: runtime.MakeBasicItem(res.Metadata.ID, res.Metadata.Namespace, res)}
}
