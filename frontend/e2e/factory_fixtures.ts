// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.
import { expect, test as base } from './auth_fixtures'

interface FactoryFixtures {
  /**
   * What the UI is expected to show for the image factory Omni is configured with.
   *
   * Read from the same variables hack/test/common.sh configures Omni with, so the expectations follow
   * whatever factory the suite is run against, and default to the public factory for a local run.
   */
  factory: {
    /** True when the factory is an enterprise build: scan reports and checksums offered, PXE hidden. */
    isEnterprise: boolean
    /** Host the image download URLs point at. */
    imageHost: string
    /** Host the PXE URLs point at, derived the way Omni does it. */
    pxeHost: string
  }
}

const { host } = new URL(process.env.OMNI_IMAGE_FACTORY_BASE_URL ?? 'https://factory.talos.dev')

const test = base.extend<FactoryFixtures>({
  factory: [
    {
      isEnterprise: process.env.WITH_IMAGE_FACTORY_ENTERPRISE === 'true',
      imageHost: host,
      pxeHost: `pxe.${host}`,
    },
    { option: true },
  ],
})

export { expect, test }
