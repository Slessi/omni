// Copyright (c) 2026 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.
import { faker } from '@faker-js/faker'
import type { Meta, StoryObj } from '@storybook/vue3-vite'
import { delay, http, HttpResponse } from 'msw'
import { fn } from 'storybook/test'

import type { Empty } from '@/api/google/protobuf/empty.pb'
import type { RevokeServiceAccountKeyRequest } from '@/api/omni/management/management.pb'

import ServiceAccountKeyRevokeModal from './ServiceAccountKeyRevokeModal.vue'

const meta: Meta<typeof ServiceAccountKeyRevokeModal> = {
  component: ServiceAccountKeyRevokeModal,

  args: {
    open: true,
    'onUpdate:open': fn(),
    identity: 'automation@serviceaccount.omni.sidero.dev',
    publicKeyId: faker.string.hexadecimal({ length: 40, casing: 'lower', prefix: '' }),
  },

  beforeEach({ msw }) {
    msw.use(
      http.post<never, RevokeServiceAccountKeyRequest, Empty>(
        '/management.ManagementService/RevokeServiceAccountKey',
        async () => {
          await delay()

          return HttpResponse.json({})
        },
      ),
    )
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}
