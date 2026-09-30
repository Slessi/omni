<!--
Copyright (c) 2026 Sidero Labs, Inc.

Use of this software is governed by the Business Source License
included in the LICENSE file.
-->
<script setup lang="ts">
import { isPast } from 'date-fns'
import { computed } from 'vue'

import type { ServiceAccountStatusSpecPgpPublicKey } from '@/api/omni/specs/auth.pb'
import IconButton from '@/components/Button/IconButton.vue'
import Tooltip from '@/components/Tooltip/Tooltip.vue'
import { relativeISO } from '@/methods/time'

const { keys = [] } = defineProps<{
  keys?: ServiceAccountStatusSpecPgpPublicKey[]
  canRevoke?: boolean
}>()

const emit = defineEmits<{
  revoke: [publicKeyId: string]
}>()

const sortedKeys = computed(() =>
  keys.toSorted((a, b) => (b.created ?? '').localeCompare(a.created ?? '')),
)
</script>

<template>
  <div class="rounded-md border border-naturals-n5 bg-naturals-n0">
    <p v-if="!sortedKeys.length" class="px-3 py-2.5 text-naturals-n10">No keys</p>

    <div
      v-else
      role="table"
      aria-label="Public keys"
      class="grid grid-cols-[minmax(0,2fr)_repeat(3,minmax(0,1fr))_auto] gap-x-2 px-3"
    >
      <div role="row" class="col-span-full grid grid-cols-subgrid py-2 text-naturals-n9">
        <div role="columnheader">Key ID</div>
        <div role="columnheader">Created</div>
        <div role="columnheader">Last Used</div>
        <div role="columnheader">Expiration</div>
        <div role="columnheader" aria-hidden="true"></div>
      </div>

      <div
        v-for="key in sortedKeys"
        :key="key.id"
        role="row"
        class="col-span-full grid grid-cols-subgrid items-center border-t border-naturals-n4 py-2"
      >
        <div role="cell" class="truncate font-mono text-naturals-n12">{{ key.id }}</div>
        <div role="cell" class="text-naturals-n10">{{ relativeISO(key.created ?? '') }}</div>
        <div role="cell" class="text-naturals-n10">
          {{ key.last_used ? relativeISO(key.last_used) : 'Never' }}
        </div>
        <div role="cell">
          <span v-if="key.expiration && isPast(key.expiration)" class="resource-label label-red">
            Expired
          </span>
          <template v-else>{{ relativeISO(key.expiration ?? '') }}</template>
        </div>
        <div role="cell">
          <Tooltip
            v-if="canRevoke"
            :description="
              sortedKeys.length > 1
                ? 'Revoke key'
                : 'The last key cannot be revoked, delete the service account instead'
            "
          >
            <span class="inline-flex">
              <IconButton
                icon="delete"
                danger
                aria-label="Revoke key"
                :disabled="sortedKeys.length <= 1"
                @click="emit('revoke', key.id!)"
              />
            </span>
          </Tooltip>
        </div>
      </div>
    </div>
  </div>
</template>
