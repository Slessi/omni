// Copyright (c) 2025 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

package check

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/cosi-project/runtime/pkg/controller"
	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/cosi-project/runtime/pkg/state"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/resources/etcd"

	"github.com/siderolabs/omni/client/api/omni/specs"
	"github.com/siderolabs/omni/client/pkg/omni/resources"
	"github.com/siderolabs/omni/client/pkg/omni/resources/omni"
	"github.com/siderolabs/omni/internal/backend/runtime/talos"
)

// Error is returned when the check has failed.
type Error struct {
	message   string
	Status    specs.ControlPlaneStatusSpec_Condition_Status
	Severity  specs.ControlPlaneStatusSpec_Condition_Severity
	Interrupt bool
}

// Error implements error interface.
func (e *Error) Error() string {
	return e.message
}

func newErrorf(severity specs.ControlPlaneStatusSpec_Condition_Severity, interrupt bool, msg string, params ...any) error {
	return &Error{
		Status:    specs.ControlPlaneStatusSpec_Condition_NotReady,
		Severity:  severity,
		message:   fmt.Sprintf(msg, params...),
		Interrupt: interrupt,
	}
}

// Etcd checks that all etcd members are healthy and are in sync, etcd responds on all nodes.
func Etcd(ctx context.Context, r controller.Reader, clusterName string) error {
	clusterMachineStatuses, err := safe.ReaderListAll[*omni.ClusterMachineStatus](ctx, r, state.WithLabelQuery(
		resource.LabelEqual(omni.LabelCluster, clusterName),
		resource.LabelExists(omni.LabelControlPlaneRole),
	),
	)
	if err != nil {
		return newErrorf(specs.ControlPlaneStatusSpec_Condition_Error, false, "Failed to get the list of machines %s", err.Error())
	}

	var members map[uint64]string

	for item := range clusterMachineStatuses.All() {
		nodeMembers, err := checkEtcd(ctx, r, clusterName, item)
		if err != nil {
			return err
		}

		if members == nil {
			members = nodeMembers

			continue
		}

		if !maps.Equal(members, nodeMembers) {
			return newErrorf(
				specs.ControlPlaneStatusSpec_Condition_Error,
				false,
				"Etcd members don't match on nodes %s and %s",
				clusterMachineStatuses.Get(0).Metadata().ID(),
				item.Metadata().ID(),
			)
		}
	}

	quorum := len(members)/2 + 1
	optimalMembers := quorum*2 - 1

	switch {
	case len(members) == 2:
		return newErrorf(
			specs.ControlPlaneStatusSpec_Condition_Warning,
			false,
			"Etcd members count is equal to the minimum quorum %d. The cluster cannot tolerate the loss of any members",
			quorum,
		)
	case len(members) != optimalMembers:
		return newErrorf(
			specs.ControlPlaneStatusSpec_Condition_Warning,
			false,
			"Etcd members count %d is not optimal. Recommended count is %d",
			len(members),
			optimalMembers,
		)
	}

	return nil
}

func checkEtcd(ctx context.Context, r controller.Reader, clusterName string, clusterMachineStatus *omni.ClusterMachineStatus) (map[uint64]string, error) {
	talosClient, err := buildTalosClient(ctx, r, clusterName, clusterMachineStatus)
	if err != nil {
		return nil, err
	}

	defer talosClient.Close() //nolint:errcheck

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	if err = checkEtcdStatus(ctx, talosClient); err != nil {
		if talos.IsClientNotReadyError(err) {
			return nil, newErrorf(specs.ControlPlaneStatusSpec_Condition_Error, false, "Talos client is not ready on node %s", clusterMachineStatus.Metadata().ID())
		}

		return nil, newErrorf(specs.ControlPlaneStatusSpec_Condition_Error, false, "Etcd status check failed: %s", err.Error())
	}

	members, err := getEtcdMembers(ctx, talosClient)
	if err != nil {
		if talos.IsClientNotReadyError(err) {
			return nil, newErrorf(specs.ControlPlaneStatusSpec_Condition_Error, false, "Talos client is not ready on node %s", clusterMachineStatus.Metadata().ID())
		}

		return nil, newErrorf(specs.ControlPlaneStatusSpec_Condition_Error, false, "Failed to get etcd members: %s", err.Error())
	}

	return members, nil
}

func buildTalosClient(ctx context.Context, r controller.Reader, clusterName string, clusterMachineStatuses ...*omni.ClusterMachineStatus) (*client.Client, error) {
	talosConfig, err := safe.ReaderGet[*omni.TalosConfig](ctx, r, resource.NewMetadata(resources.DefaultNamespace, omni.TalosConfigType, clusterName, resource.VersionUndefined))
	if err != nil {
		if state.IsNotFoundError(err) {
			return nil, talos.NewClientNotReadyError(err)
		}

		return nil, err
	}

	endpoints := make([]string, 0, len(clusterMachineStatuses))

	var unixSocketOpts []client.OptionFunc

	for _, status := range clusterMachineStatuses {
		if status.TypedSpec().Value.ManagementAddress == "" || status.TypedSpec().Value.IsRemoved {
			continue
		}

		if o := talos.GetSocketOptions(status.TypedSpec().Value.ManagementAddress); o != nil {
			unixSocketOpts = o

			break
		}

		endpoints = append(endpoints, status.TypedSpec().Value.ManagementAddress)
	}

	if len(unixSocketOpts) == 0 && len(endpoints) == 0 {
		return nil, talos.NewClientNotReadyError(errors.New("no management addresses found on cluster machine statuses"))
	}

	config := omni.NewTalosClientConfig(talosConfig, endpoints...)

	return client.New(ctx, append(unixSocketOpts, client.WithConfig(config))...)
}

// GetEtcdMembers returns etcd members for a machine.
func getEtcdMembers(ctx context.Context, talosClient *client.Client) (map[uint64]string, error) {
	response, err := talosClient.EtcdMemberList(ctx, &machine.EtcdMemberListRequest{})
	if err != nil {
		return nil, err
	}

	members := map[uint64]string{}

	for _, memberList := range response.Messages {
		for _, member := range memberList.Members {
			members[member.Id] = member.Hostname
		}
	}

	return members, nil
}

func checkEtcdStatus(ctx context.Context, talosClient *client.Client) error {
	status, err := talosClient.EtcdStatus(ctx)
	if err != nil {
		return err
	}

	for _, message := range status.Messages {
		memberErrors := message.GetMemberStatus().GetErrors()
		if len(memberErrors) > 0 {
			return fmt.Errorf("etcd member %q has errors: %s", message.GetMemberStatus().GetMemberId(), strings.Join(memberErrors, ", "))
		}
	}

	return nil
}

// CanScaleDown verifies that the machine can be safely removed from the control planes machine set.
func CanScaleDown(status *EtcdStatusResult, machine resource.Resource) error {
	member, ok := status.Members[machine.Metadata().ID()]
	if !ok {
		return nil
	}

	totalMembers := len(status.Members)
	healthyMembers := status.HealthyMembers

	if healthyMembers < totalMembers/2+1 {
		return fmt.Errorf("removing machine %q is not possible, etcd doesn't have quorum", machine.Metadata().ID())
	}

	totalMembers--

	if member.Healthy {
		healthyMembers--
	}

	if healthyMembers < totalMembers/2+1 {
		return fmt.Errorf("removing machine %q will break etcd quorum", machine.Metadata().ID())
	}

	return nil
}

// EtcdStatusResult is the current etcd state: members count, healthy members count and the health of each member.
type EtcdStatusResult struct {
	// Members all members statuses.
	// Also contains the members which haven't joined yet.
	Members map[string]EtcdMemberStatus
	// HealthyMembers is the number of healthy members.
	HealthyMembers int
}

// EtcdMemberStatus is the current state of the etcd member.
type EtcdMemberStatus struct {
	// Error describes check error.
	Error string
	// Healthy is the response from etcd service health request.
	Healthy bool
}

// EtcdStatus reads control plane etcd members health.
func EtcdStatus(ctx context.Context, r controller.Reader, machineSet *omni.MachineSet) (*EtcdStatusResult, error) {
	if _, ok := machineSet.Metadata().Labels().Get(omni.LabelControlPlaneRole); !ok {
		return nil, errors.New("the machine set is not control planes")
	}

	clusterName, ok := machineSet.Metadata().Labels().Get(omni.LabelCluster)
	if !ok {
		return nil, fmt.Errorf("machine set doesn't have %s label", omni.LabelCluster)
	}

	talosConfig, err := safe.ReaderGet[*omni.TalosConfig](ctx, r, resource.NewMetadata(resources.DefaultNamespace, omni.TalosConfigType, clusterName, resource.VersionUndefined))
	if err != nil {
		return nil, fmt.Errorf("failed to get talos config for the cluster %q: %w", clusterName, err)
	}

	members := map[string]EtcdMemberStatus{}

	var healthyMembers int

	statuses, err := safe.ReaderListAll[*omni.ClusterMachineStatus](ctx, r,
		state.WithLabelQuery(resource.LabelEqual(omni.LabelMachineSet, machineSet.Metadata().ID())),
	)
	if err != nil {
		return nil, err
	}

	talosClient, err := buildTalosClient(ctx, r, clusterName, toSlice(statuses)...)
	if err != nil {
		return nil, err
	}

	defer talosClient.Close() //nolint:errcheck

	etcdMembers, err := getEtcdMembers(ctx, talosClient)
	if err != nil {
		return nil, err
	}

	if err = statuses.ForEachErr(func(status *omni.ClusterMachineStatus) error {
		var (
			member   EtcdMemberStatus
			identity *omni.ClusterMachineIdentity
		)

		member, err = getMemberState(ctx, talosConfig, status)
		if err != nil {
			return err
		}

		identity, err = safe.ReaderGet[*omni.ClusterMachineIdentity](ctx, r, omni.NewClusterMachineIdentity(
			resources.DefaultNamespace,
			status.Metadata().ID(),
		).Metadata())
		if err != nil {
			if state.IsNotFoundError(err) {
				return nil
			}

			return err
		}

		memberID := identity.TypedSpec().Value.EtcdMemberId

		if memberID == 0 {
			return nil
		}

		if _, ok := etcdMembers[memberID]; !ok {
			return nil
		}

		delete(etcdMembers, memberID)

		members[status.Metadata().ID()] = member

		if member.Healthy {
			healthyMembers++
		}

		return nil
	}); err != nil {
		return nil, err
	}

	if len(etcdMembers) > 0 {
		extraMembers := make([]string, 0, len(etcdMembers))

		for id := range etcdMembers {
			extraMembers = append(extraMembers, etcd.FormatMemberID(id))
		}

		return nil, fmt.Errorf("couldn't map etcd members to cluster machines: %q", strings.Join(extraMembers, ", "))
	}

	return &EtcdStatusResult{
		Members:        members,
		HealthyMembers: healthyMembers,
	}, nil
}

func getMemberState(ctx context.Context, talosConfig *omni.TalosConfig, clusterMachineStatus *omni.ClusterMachineStatus) (EtcdMemberStatus, error) {
	var status EtcdMemberStatus

	_, connected := clusterMachineStatus.Metadata().Labels().Get(omni.MachineStatusLabelConnected)

	if !connected || clusterMachineStatus.TypedSpec().Value.ManagementAddress == "" {
		status.Error = "the machine is unreachable"

		return status, nil
	}

	opts := []client.OptionFunc{}

	if o := talos.GetSocketOptions(clusterMachineStatus.TypedSpec().Value.ManagementAddress); o != nil {
		opts = o
	}

	config := omni.NewTalosClientConfig(talosConfig, clusterMachineStatus.TypedSpec().Value.ManagementAddress)

	opts = append(opts, client.WithConfig(config))

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)

	defer cancel()

	c, err := client.New(ctx, opts...)
	if err != nil {
		return status, err
	}

	defer c.Close() //nolint:errcheck

	list, err := c.ServiceInfo(ctx, "etcd")
	if err != nil {
		status.Error = err.Error()

		return status, nil
	}

	for _, info := range list {
		if info.Metadata != nil && info.Metadata.Error != "" {
			status.Error = info.Metadata.Error

			return status, nil
		}

		status.Healthy = info.Service.Health.Healthy
	}

	return status, nil
}

func toSlice[T resource.Resource](list safe.List[T]) []T {
	res := make([]T, 0, list.Len())

	list.ForEach(func(t T) {
		res = append(res, t)
	})

	return res
}
