<!--
Copyright (c) 2026 Sidero Labs, Inc.

Use of this software is governed by the Business Source License
included in the LICENSE file.
-->
<script setup lang="ts">
import { ref, watchEffect } from 'vue'

import ConfirmModal from '@/components/Modals/ConfirmModal.vue'
import { revokeServiceAccountKey } from '@/methods/user'
import { showError, showSuccess } from '@/notification'

const { identity, publicKeyId } = defineProps<{
  identity: string
  publicKeyId: string
}>()

const open = defineModel<boolean>('open', { default: false })

const isRevoking = ref(false)

watchEffect(() => {
  if (open.value) return

  isRevoking.value = false
})

const revoke = async () => {
  try {
    isRevoking.value = true

    await revokeServiceAccountKey(identity, publicKeyId)

    showSuccess('Revoked Service Account Key', publicKeyId)

    open.value = false
  } catch (e) {
    showError('Failed to Revoke Service Account Key', e instanceof Error ? e.message : String(e))
  } finally {
    isRevoking.value = false
  }
}
</script>

<template>
  <ConfirmModal
    v-model:open="open"
    title="Revoke Service Account Key"
    action-label="Revoke"
    :loading="isRevoking"
    @confirm="revoke"
  >
    <template #description>{{ identity }}</template>

    <p class="text-xs">
      Clients using the key
      <code class="font-mono text-naturals-n14">{{ publicKeyId }}</code>
      will immediately lose access. Please confirm the action.
    </p>
  </ConfirmModal>
</template>
