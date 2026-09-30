<!--
Copyright (c) 2026 Sidero Labs, Inc.

Use of this software is governed by the Business Source License
included in the LICENSE file.
-->
<script setup lang="ts">
import { isPast } from 'date-fns'
import { CollapsibleContent, CollapsibleRoot, CollapsibleTrigger } from 'reka-ui'
import WordHighlighter from 'vue-word-highlighter'

import type { Resource } from '@/api/grpc'
import type { ServiceAccountStatusSpec } from '@/api/omni/specs/auth.pb'
import { RoleInfraProvider } from '@/api/resources'
import TActionsBox from '@/components/ActionsBox/TActionsBox.vue'
import TActionsBoxItem from '@/components/ActionsBox/TActionsBoxItem.vue'
import TIcon from '@/components/Icon/TIcon.vue'
import { usePermissions } from '@/methods/auth'
import { relativeISO } from '@/methods/time'
import ServiceAccountKeys from '@/views/Users/components/ServiceAccountKeys.vue'

const { item } = defineProps<{
  item: Resource<ServiceAccountStatusSpec>
  lastActive: string
  search: string
}>()

const emit = defineEmits<{
  renew: []
  edit: []
  delete: []
  revokeKey: [publicKeyId: string]
}>()

const open = defineModel<boolean>({ default: false })

const { canManageUsers } = usePermissions()

function toggleRow() {
  open.value = !open.value
}
</script>

<template>
  <CollapsibleRoot v-model:open="open" class="group/root contents">
    <CollapsibleTrigger
      as="div"
      role="row"
      tabindex="0"
      class="group/trigger col-span-full grid cursor-pointer grid-cols-subgrid items-center px-2 py-2.5 select-none group-hover/root:bg-white/5"
      @keydown.enter.prevent="toggleRow"
      @keydown.space.prevent="toggleRow"
    >
      <div role="cell" aria-hidden="true">
        <div class="size-5 rounded-md bg-naturals-n5 p-0.5 text-naturals-n10">
          <TIcon
            icon="dropdown"
            class="transition-transform group-data-[state=open]/trigger:rotate-180"
          />
        </div>
      </div>

      <div role="cell" class="truncate">
        <WordHighlighter
          :query="search"
          :text-to-highlight="item.metadata.id"
          split-by-space
          class="font-bold"
          highlight-class="bg-naturals-n14"
        />
      </div>

      <div role="cell">
        <span class="resource-label">{{ item.spec.role ?? 'None' }}</span>
      </div>

      <div role="cell" class="text-naturals-n10">{{ lastActive }}</div>

      <div role="cell" class="text-naturals-n10">{{ item.spec.public_keys?.length ?? 0 }}</div>

      <div role="cell">
        <span
          v-if="item.spec.expiration && isPast(item.spec.expiration)"
          class="resource-label label-red"
        >
          Expired
        </span>
        <template v-else>{{ relativeISO(item.spec.expiration ?? '') }}</template>
      </div>

      <div role="cell" @click.stop @keydown.stop>
        <TActionsBox v-if="canManageUsers">
          <TActionsBoxItem icon="refresh" @select="emit('renew')">Renew Key</TActionsBoxItem>

          <TActionsBoxItem
            v-if="item.spec.role !== RoleInfraProvider"
            icon="edit"
            @select="emit('edit')"
          >
            Edit Service Account
          </TActionsBoxItem>

          <TActionsBoxItem icon="delete" danger @select="emit('delete')">
            Delete Service Account
          </TActionsBoxItem>
        </TActionsBox>
      </div>
    </CollapsibleTrigger>

    <CollapsibleContent
      role="row"
      class="collapsible-content col-span-full overflow-hidden group-hover/root:bg-white/5"
    >
      <div role="cell" class="px-2 pb-2 pl-11">
        <ServiceAccountKeys
          :keys="item.spec.public_keys"
          :can-revoke="canManageUsers"
          @revoke="(publicKeyId) => emit('revokeKey', publicKeyId)"
        />
      </div>
    </CollapsibleContent>
  </CollapsibleRoot>
</template>

<style scoped>
.collapsible-content[data-state='open'] {
  animation: slideDown 200ms ease-out;
}

.collapsible-content[data-state='closed'] {
  animation: slideUp 200ms ease-out;
}

@keyframes slideDown {
  from {
    height: 0;
  }
  to {
    height: var(--reka-collapsible-content-height);
  }
}

@keyframes slideUp {
  from {
    height: var(--reka-collapsible-content-height);
  }
  to {
    height: 0;
  }
}
</style>
