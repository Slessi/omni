<!--
Copyright (c) 2026 Sidero Labs, Inc.

Use of this software is governed by the Business Source License
included in the LICENSE file.
-->
<script setup lang="ts">
import { computed } from 'vue'

import { Runtime } from '@/api/common/omni.pb'
import type { MachineStatusMetricsSpec } from '@/api/omni/specs/omni.pb'
import {
  EphemeralNamespace,
  MachineStatusMetricsID,
  MachineStatusMetricsType,
} from '@/api/resources'
import Card from '@/components/common/Card/Card.vue'
import RadialBar from '@/components/common/Charts/RadialBar.vue'
import { useResourceWatch } from '@/methods/useResourceWatch'

const { data } = useResourceWatch<MachineStatusMetricsSpec>({
  resource: {
    namespace: EphemeralNamespace,
    type: MachineStatusMetricsType,
    id: MachineStatusMetricsID,
  },
  runtime: Runtime.Omni,
})

const counts = computed(() => {
  const spec = data.value?.spec

  const allocatedCount = spec?.allocated_machines_count ?? 0
  const connectedCount = spec?.connected_machines_count ?? 0
  const pendingCount = spec?.pending_machines_count ?? 0
  const registeredCount = spec?.registered_machines_count ?? 0

  return {
    totalCount: registeredCount + pendingCount,
    connectedCount,
    pendingCount,
    notConnectedCount: registeredCount - connectedCount,
    inClusterCount: allocatedCount,
    freeMachineCount: registeredCount - allocatedCount,
  }
})
</script>

<template>
  <Card class="p-4">
    <RadialBar
      title="Machines"
      show-hollow-total
      :total="202"
      :items="[
        { label: 'Connected', value: 198 },
        { label: 'Not Connected', value: 4 },
        { label: 'In Cluster', value: 133 },
        { label: 'Free Machine', value: 65 },
        { label: 'Pending', value: 17 },
      ]"
    />
  </Card>
</template>
