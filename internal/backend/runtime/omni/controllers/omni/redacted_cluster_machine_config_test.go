// Copyright (c) 2025 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

package omni_test

import (
	"testing"

	"github.com/cosi-project/runtime/pkg/resource"
	"github.com/cosi-project/runtime/pkg/resource/rtestutils"
	"github.com/siderolabs/crypto/x509"
	"github.com/siderolabs/talos/pkg/machinery/config/generate"
	"github.com/siderolabs/talos/pkg/machinery/config/machine"
	"github.com/siderolabs/talos/pkg/machinery/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/siderolabs/omni/client/pkg/omni/resources"
	"github.com/siderolabs/omni/client/pkg/omni/resources/omni"
	omnictrl "github.com/siderolabs/omni/internal/backend/runtime/omni/controllers/omni"
)

type RedactedClusterMachineConfigSuite struct {
	OmniSuite
}

func (suite *RedactedClusterMachineConfigSuite) TestReconcile() {
	suite.startRuntime()

	suite.Require().NoError(suite.runtime.RegisterQController(omnictrl.NewClusterMachineConfigController(imageFactoryHost, nil)))
	suite.Require().NoError(suite.runtime.RegisterQController(omnictrl.NewRedactedClusterMachineConfigController()))
	suite.Require().NoError(suite.runtime.RegisterQController(omnictrl.NewMachineJoinConfigController()))

	id := "test"

	cmc := omni.NewClusterMachineConfig(resources.DefaultNamespace, id)

	suite.Require().NoError(cmc.TypedSpec().Value.SetUncompressedData(suite.generateConfig()))

	suite.Require().NoError(suite.state.Create(suite.ctx, cmc))

	rtestutils.AssertResources(suite.ctx, suite.T(), suite.state, []string{id},
		func(rcmc *omni.RedactedClusterMachineConfig, assert *assert.Assertions) {
			buffer, err := rcmc.TypedSpec().Value.GetUncompressedData()
			assert.NoError(err)

			defer buffer.Free()

			data := string(buffer.Data())

			assert.Contains(data, x509.Redacted)
		},
	)

	rtestutils.Destroy[*omni.ClusterMachineConfig](suite.ctx, suite.T(), suite.state, []resource.ID{id})

	rtestutils.AssertNoResource[*omni.RedactedClusterMachineConfig](suite.ctx, suite.T(), suite.state, id)
}

func (suite *RedactedClusterMachineConfigSuite) generateConfig() []byte {
	input, err := generate.NewInput(
		"does-not-matter",
		"https://doesntmatter:6443",
		constants.DefaultKubernetesVersion,
	)
	suite.Require().NoError(err)

	config, err := input.Config(machine.TypeControlPlane)
	suite.Require().NoError(err)

	data, err := config.Bytes()
	suite.Require().NoError(err)

	return data
}

func TestRedactedClusterMachineConfigSuite(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(RedactedClusterMachineConfigSuite))
}
