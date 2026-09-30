<!--
Copyright (c) 2026 Sidero Labs, Inc.

Use of this software is governed by the Business Source License
included in the LICENSE file.
-->
<script setup lang="ts">
import { computed, ref } from 'vue'

import { Runtime } from '@/api/common/omni.pb'
import type { Resource } from '@/api/grpc'
import type { IdentityStatusSpec, ServiceAccountStatusSpec } from '@/api/omni/specs/auth.pb'
import {
  EphemeralNamespace,
  IdentityStatusType,
  LabelIdentityTypeServiceAccount,
  ServiceAccountStatusType,
} from '@/api/resources'
import TButton from '@/components/Button/TButton.vue'
import PageContainer from '@/components/PageContainer/PageContainer.vue'
import PageHeader from '@/components/PageHeader.vue'
import Pagination from '@/components/Pagination/Pagination.vue'
import TSelectList from '@/components/SelectList/TSelectList.vue'
import TInput from '@/components/TInput/TInput.vue'
import { usePermissions } from '@/methods/auth'
import { useResourcePagination } from '@/methods/resource/useResourcePagination'
import { useResourceSearch } from '@/methods/resource/useResourceSearch'
import { relativeISO } from '@/methods/time'
import { useTitle } from '@/methods/title'
import { useResourceWatch } from '@/methods/useResourceWatch'
import RoleEditModal from '@/views/Users/components/RoleEditModal.vue'
import ServiceAccountCreateModal from '@/views/Users/components/ServiceAccountCreateModal.vue'
import ServiceAccountItem from '@/views/Users/components/ServiceAccountItem.vue'
import ServiceAccountKeyRevokeModal from '@/views/Users/components/ServiceAccountKeyRevokeModal.vue'
import ServiceAccountRenewModal from '@/views/Users/components/ServiceAccountRenewModal.vue'
import UserDestroyModal from '@/views/Users/components/UserDestroyModal.vue'

definePage({
  name: 'ServiceAccounts',
})

useTitle('Service Accounts')

const { canManageUsers } = usePermissions()

const serviceAccCreateModalOpen = ref(false)
const filterValue = ref('')

const userDestroyModal = ref<{ open: boolean; identity?: string }>({ open: false })
const roleEditModal = ref<{ open: boolean; identity?: string; userId?: string }>({ open: false })
const serviceAccountRenewModal = ref<{ open: boolean; identity?: string }>({ open: false })
const keyRevokeModal = ref<{ open: boolean; identity?: string; publicKeyId?: string }>({
  open: false,
})

const expandedIds = ref(new Set<string>())

function toggleExpanded(id: string) {
  if (expandedIds.value.has(id)) {
    expandedIds.value.delete(id)
  } else {
    expandedIds.value.add(id)
  }
}

const { watchOptions: searchState, searchQuery } = useResourceSearch({ filterValue })

const {
  total,
  watchOptions: paginationState,
  currentPage,
  currentPageSize,
  pageCount,
  pageSizeSelectValues,
} = useResourcePagination({
  resetOn: [searchState],
})

const { data: identities } = useResourceWatch<IdentityStatusSpec>({
  runtime: Runtime.Omni,
  resource: {
    type: IdentityStatusType,
    namespace: EphemeralNamespace,
  },
  selectors: [LabelIdentityTypeServiceAccount],
})

const { data: serviceAccounts } = useResourceWatch<ServiceAccountStatusSpec>(
  () => ({
    runtime: Runtime.Omni,
    resource: {
      type: ServiceAccountStatusType,
      namespace: EphemeralNamespace,
    },
    ...paginationState.value,
    ...searchState.value,
  }),
  { total },
)

const identityMap = computed(() => new Map(identities.value.map((s) => [s.metadata.id!, s])))

const getLastActive = (serviceAcc: Resource<ServiceAccountStatusSpec>) => {
  const identity = identityMap.value.get(serviceAcc.metadata.id!)

  return identity?.spec.last_active ? relativeISO(identity.spec.last_active) : 'Never'
}
</script>

<template>
  <PageContainer class="flex h-full flex-col gap-4">
    <PageHeader title="Settings" subtitle="Service Accounts" />

    <div class="flex grow flex-col gap-2">
      <div class="flex justify-end">
        <TButton
          icon="plus"
          icon-position="left"
          variant="highlighted"
          :disabled="!canManageUsers"
          @click="serviceAccCreateModalOpen = true"
        >
          Create Service Account
        </TButton>
      </div>

      <TInput v-model="filterValue" icon="search" />

      <TSelectList
        v-model="currentPageSize"
        class="self-end"
        title="Items per Page"
        :values="pageSizeSelectValues"
      />

      <div role="grid" class="text-xs text-naturals-n13">
        <div
          role="rowgroup"
          class="grid grid-cols-[36px_minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)_60px_minmax(0,1fr)_36px] gap-x-0.5 bg-naturals-n2 px-2 text-left"
        >
          <div role="columnheader" aria-hidden="true" class="py-2 uppercase"></div>
          <div role="columnheader" class="py-2 uppercase">ID</div>
          <div role="columnheader" class="py-2 uppercase">Role</div>
          <div role="columnheader" class="py-2 uppercase">Last Active</div>
          <div role="columnheader" class="py-2 uppercase">Keys</div>
          <div role="columnheader" class="py-2 uppercase">Expiration</div>
          <div role="columnheader" aria-hidden="true" class="py-2 uppercase"></div>
        </div>

        <div role="rowgroup">
          <div
            v-for="item in serviceAccounts"
            :key="item.metadata.id"
            class="grid grid-cols-[36px_minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)_60px_minmax(0,1fr)_36px] gap-x-0.5 border-t border-naturals-n5"
          >
            <ServiceAccountItem
              :item
              :last-active="getLastActive(item)"
              :search="searchQuery"
              :model-value="expandedIds.has(item.metadata.id!)"
              @update:model-value="toggleExpanded(item.metadata.id!)"
              @renew="serviceAccountRenewModal = { open: true, identity: item.metadata.id }"
              @edit="
                roleEditModal = {
                  open: true,
                  identity: item.metadata.id,
                  userId: identityMap.get(item.metadata.id!)?.spec.user_id,
                }
              "
              @delete="userDestroyModal = { open: true, identity: item.metadata.id }"
              @revoke-key="
                (publicKeyId) =>
                  (keyRevokeModal = { open: true, identity: item.metadata.id, publicKeyId })
              "
            />
          </div>
        </div>
      </div>
    </div>

    <Pagination v-model:current-page="currentPage" :page-count="pageCount" />

    <ServiceAccountCreateModal v-model:open="serviceAccCreateModalOpen" />

    <ServiceAccountRenewModal
      v-if="serviceAccountRenewModal.identity"
      v-model:open="serviceAccountRenewModal.open"
      :identity="serviceAccountRenewModal.identity"
    />

    <ServiceAccountKeyRevokeModal
      v-if="keyRevokeModal.identity && keyRevokeModal.publicKeyId"
      v-model:open="keyRevokeModal.open"
      :identity="keyRevokeModal.identity"
      :public-key-id="keyRevokeModal.publicKeyId"
    />

    <RoleEditModal
      v-if="roleEditModal.identity && roleEditModal.userId"
      v-model:open="roleEditModal.open"
      :identity="roleEditModal.identity"
      :user-id="roleEditModal.userId"
      is-service-account
    />

    <UserDestroyModal
      v-if="userDestroyModal.identity"
      v-model:open="userDestroyModal.open"
      :identity="userDestroyModal.identity"
      is-service-account
    />
  </PageContainer>
</template>
