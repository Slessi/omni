// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

package talosaccess_test

import (
	"context"
	"testing"

	"github.com/blang/semver/v4"
	"github.com/cosi-project/runtime/api/v1alpha1"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/api/storage"
	talosrole "github.com/siderolabs/talos/pkg/machinery/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/siderolabs/omni/client/pkg/access/role"
	"github.com/siderolabs/omni/internal/pkg/auth"
	"github.com/siderolabs/omni/internal/pkg/auth/talosaccess"
	"github.com/siderolabs/omni/internal/pkg/ctxstore"
)

func callerContext(ctx context.Context, r role.Role) context.Context {
	ctx = ctxstore.WithValue(ctx, auth.EnabledAuthContextKey{Enabled: true})
	ctx = ctxstore.WithValue(ctx, auth.IdentityContextKey{Identity: "user@example.org"})

	return ctxstore.WithValue(ctx, auth.RoleContextKey{Role: r})
}

func TestCheck(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name       string
		role       role.Role
		clusterID  string
		wantModify bool
		wantCode   codes.Code
	}{
		{name: "none is denied", role: role.None, clusterID: "alpha", wantCode: codes.PermissionDenied},
		{name: "reader reads", role: role.Reader, clusterID: "alpha"},
		{name: "operator modifies", role: role.Operator, clusterID: "alpha", wantModify: true},
		{name: "admin modifies", role: role.Admin, clusterID: "alpha", wantModify: true},
		{name: "reader is denied outside of a cluster", role: role.Reader, wantCode: codes.PermissionDenied},
		{name: "operator modifies outside of a cluster", role: role.Operator, wantModify: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hasModifyAccess, err := talosaccess.Check(callerContext(t.Context(), tt.role), tt.clusterID)

			if tt.wantCode != codes.OK {
				require.Error(t, err)
				assert.Equal(t, tt.wantCode, status.Code(err))

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantModify, hasModifyAccess)
		})
	}
}

func TestCheckWithoutSignature(t *testing.T) {
	t.Parallel()

	// a context with auth enabled but no role was not signed by a known identity
	ctx := ctxstore.WithValue(t.Context(), auth.EnabledAuthContextKey{Enabled: true})

	_, err := talosaccess.Check(ctx, "alpha")
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestRoles(t *testing.T) {
	t.Parallel()

	v1_3 := semver.MustParse("1.3.0")
	v1_11 := semver.MustParse("1.11.0")
	v1_12 := semver.MustParse("1.12.0")

	for _, tt := range []struct {
		version *semver.Version
		want    talosrole.Set
		name    string
		method  string
		modify  bool
	}{
		// resource reads never act as admin, so sensitive resources stay unreadable for everyone
		{name: "reader reads resources", method: v1alpha1.State_Get_FullMethodName, version: &v1_11, want: talosrole.MakeSet(talosrole.Reader)},
		{name: "operator reads resources", method: v1alpha1.State_Get_FullMethodName, modify: true, version: &v1_11, want: talosrole.MakeSet(talosrole.Operator)},
		{name: "operator reads resources of old Talos", method: v1alpha1.State_List_FullMethodName, modify: true, version: &v1_3, want: talosrole.MakeSet(talosrole.Reader)},
		{name: "operator reads resources of unknown Talos", method: v1alpha1.State_Watch_FullMethodName, modify: true, want: talosrole.MakeSet(talosrole.Reader)},

		{name: "reader cannot reboot", method: machine.MachineService_Reboot_FullMethodName, version: &v1_11, want: talosrole.MakeSet(talosrole.Reader)},
		{name: "operator reboots", method: machine.MachineService_Reboot_FullMethodName, modify: true, version: &v1_11, want: talosrole.MakeSet(talosrole.Operator)},
		{name: "operator reboots old Talos", method: machine.MachineService_Reboot_FullMethodName, modify: true, version: &v1_3, want: talosrole.MakeSet(talosrole.Admin)},

		{name: "operator wipes block devices", method: storage.StorageService_BlockDeviceWipe_FullMethodName, modify: true, version: &v1_11, want: talosrole.MakeSet(talosrole.Admin)},

		{name: "operator reads files of Talos 1.12", method: machine.MachineService_Read_FullMethodName, modify: true, version: &v1_12, want: talosrole.MakeSet(talosrole.Admin)},
		{name: "operator reads files of Talos 1.11", method: machine.MachineService_Read_FullMethodName, modify: true, version: &v1_11, want: talosrole.MakeSet(talosrole.Operator)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, talosaccess.Roles(tt.method, tt.version, tt.modify))
		})
	}
}
