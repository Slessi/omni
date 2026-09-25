// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

// Package talosaccess maps a caller's Omni access onto the Talos roles it acts with on the Talos API.
//
// Omni talks to Talos with an `os:impersonator` certificate and passes the Talos roles of the request in the gRPC
// metadata, as Talos trusts the roles in the metadata only from impersonators.
package talosaccess

import (
	"context"

	"github.com/blang/semver/v4"
	"github.com/siderolabs/gen/xslices"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/api/storage"
	talosrole "github.com/siderolabs/talos/pkg/machinery/role"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/siderolabs/omni/client/pkg/access/role"
	"github.com/siderolabs/omni/internal/pkg/auth"
)

// operatorMethodSet is the set of methods that are allowed to be called by the minimum role of os:operator.
var operatorMethodSet = xslices.ToSet([]string{
	machine.MachineService_EtcdAlarmDisarm_FullMethodName,
	machine.MachineService_EtcdAlarmList_FullMethodName,
	machine.MachineService_EtcdDefragment_FullMethodName,
	machine.MachineService_EtcdStatus_FullMethodName,
	machine.MachineService_PacketCapture_FullMethodName,
	machine.MachineService_Reboot_FullMethodName,
	machine.MachineService_Restart_FullMethodName,
	machine.MachineService_ServiceRestart_FullMethodName,
	machine.MachineService_ServiceStart_FullMethodName,
	machine.MachineService_ServiceStop_FullMethodName,
	machine.MachineService_Shutdown_FullMethodName,
})

// adminMethodSet is the set of methods that are allowed to be called by the minimum role of os:admin.
var adminMethodSet = xslices.ToSet([]string{
	storage.StorageService_BlockDeviceWipe_FullMethodName,

	machine.LVMService_LogicalVolumeRemove_FullMethodName,
	machine.LVMService_PhysicalVolumeRemove_FullMethodName,
	machine.LVMService_VolumeGroupRemove_FullMethodName,

	machine.MDService_Destroy_FullMethodName,

	machine.MachineService_EtcdDowngradeCancel_FullMethodName,
	machine.MachineService_EtcdDowngradeEnable_FullMethodName,
	machine.MachineService_EtcdDowngradeValidate_FullMethodName,
	machine.MachineService_EtcdForfeitLeadership_FullMethodName,
	machine.MachineService_MetaWrite_FullMethodName,
	machine.MachineService_MetaDelete_FullMethodName,

	machine.DebugService_ContainerRun_FullMethodName,

	machine.ImageService_Remove_FullMethodName,
})

// adminMethodSet1_12 is the set of methods that are allowed to be called by the minimum role of os:admin for Talos versions >= 1.12.0.
var adminMethodSet1_12 = xslices.ToSet([]string{
	// read/copy APIs were not considered safe for older Talos versions, as the STATE partition has always been mounted
	machine.MachineService_Copy_FullMethodName,
	machine.MachineService_Read_FullMethodName,
})

// Check verifies that the caller may access the Talos API of the given cluster, and reports whether it may modify it.
//
// The context must already carry the caller's role for the cluster, see accesspolicy.ApplyClusterAccessPolicy. A
// machine outside of any cluster (an empty clusterID) is reachable only over its insecure maintenance API, which
// requires the operator role.
func Check(ctx context.Context, clusterID string) (hasModifyAccess bool, err error) {
	if _, err = auth.Check(ctx, auth.WithRole(role.Operator)); err == nil {
		return true, nil
	}

	if clusterID == "" {
		return false, status.Error(codes.PermissionDenied, "permission denied")
	}

	// at least read access is required
	if _, err = auth.CheckGRPC(ctx, auth.WithRole(role.Reader)); err != nil {
		return false, err
	}

	return false, nil
}

// Roles returns the Talos roles a caller acts with when calling the given Talos API method.
//
// minTalosVersion is the lowest Talos version among the targeted nodes, nil if unknown.
func Roles(fullMethodName string, minTalosVersion *semver.Version, hasModifyAccess bool) talosrole.Set {
	if !hasModifyAccess {
		return talosrole.MakeSet(talosrole.Reader)
	}

	// methods that should have admin access
	if _, ok := adminMethodSet[fullMethodName]; ok {
		return talosrole.MakeSet(talosrole.Admin)
	}

	// methods that should have admin access for Talos >= 1.12.0
	if _, ok := adminMethodSet1_12[fullMethodName]; ok {
		if minTalosVersion != nil && minTalosVersion.GTE(semver.MustParse("1.12.0")) {
			return talosrole.MakeSet(talosrole.Admin)
		}
	}

	// min Talos version is >= 1.4.0, we can use Operator role
	if minTalosVersion != nil && minTalosVersion.GTE(semver.MustParse("1.4.0")) {
		return talosrole.MakeSet(talosrole.Operator)
	}

	// min Talos version is unknown or < 1.4.0, fallback to backwards-compatibility logic
	if _, ok := operatorMethodSet[fullMethodName]; ok {
		return talosrole.MakeSet(talosrole.Admin)
	}

	return talosrole.MakeSet(talosrole.Reader)
}
