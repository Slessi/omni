<!--
Copyright (c) 2026 Sidero Labs, Inc.

Use of this software is governed by the Business Source License
included in the LICENSE file.
-->
<script setup lang="ts">
import { getLocalTimeZone, today } from '@internationalized/date'
import { differenceInDays } from 'date-fns'
import type { DateRange } from 'reka-ui'
import { computed, ref, shallowRef, watchEffect } from 'vue'

import { subscribe } from '@/api/grpc'
import { ManagementService } from '@/api/omni/management/management.pb'
import TButton from '@/components/common/Button/TButton.vue'
import DateRangePicker from '@/components/common/DateRangePicker/DateRangePicker.vue'
import PageContainer from '@/components/common/PageContainer/PageContainer.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import TSelectList from '@/components/common/SelectList/TSelectList.vue'
import TableCell from '@/components/common/Table/TableCell.vue'
import TableRoot from '@/components/common/Table/TableRoot.vue'
import TableRow from '@/components/common/Table/TableRow.vue'
import TInput from '@/components/common/TInput/TInput.vue'
import TAlert from '@/components/TAlert.vue'
import { formatISO } from '@/methods/time'
import DownloadAuditLogsModal from '@/views/omni/Settings/DownloadAuditLogsModal.vue'

interface AuditLogMsg {
  id: string
  event_ts: number
  event_type: string
  event_data: Record<string, unknown>
  resource_type?: string
  resource_id?: string
}

interface BufferedMsg {
  id: string
  event_ts: number
  event_type: string
  event_data: unknown
  session: unknown
  resource_type?: string
  resource_id?: string
}

const logs = ref<BufferedMsg[]>([])
const searchInput = ref('')
const selectedEventType = ref('all')
const selectedResourceType = ref('all')
const downloadAuditLogModalOpen = ref(false)

const dateRange = shallowRef<DateRange>({
  start: today(getLocalTimeZone()).subtract({ days: 3 }),
  end: today(getLocalTimeZone()).add({ days: 1 }),
})

watchEffect((onCleanup) => {
  if (!dateRange.value.end || !dateRange.value.start) return

  logs.value = []
  let buffer: BufferedMsg[] = []
  let flush: number | undefined

  const stream = subscribe(
    ManagementService.ReadAuditLog,
    {
      start_time: dateRange.value.start.toString(),
      end_time: dateRange.value.end.toString(),
    },
    (resp) => {
      clearTimeout(flush)

      if (!resp.audit_log) return

      try {
        const bytes = resp.audit_log.toString()
        const line = window.atob(bytes)
        const data = JSON.parse(line) as AuditLogMsg

        const {
          event_data: { session, ...eventData },
        } = data

        buffer.push({
          ...data,
          event_data: eventData[Object.keys(eventData)[0]],
          session,
          id: bytes,
        })
      } catch (e) {
        console.error(`failed to parse log ${resp.audit_log}`, e)
      }

      // accumulate frequent updates and then flush them in a single call
      flush = window.setTimeout(() => {
        logs.value = logs.value.concat(buffer)
        buffer = []
      }, 50)
    },
  )

  onCleanup(() => {
    clearTimeout(flush)
    stream.shutdown()
  })
})

const filteredLogs = computed(() =>
  logs.value.filter((l) => {
    if (
      selectedEventType.value &&
      selectedEventType.value !== 'all' &&
      l.event_type !== selectedEventType.value
    )
      return false
    if (
      selectedResourceType.value &&
      selectedResourceType.value !== 'all' &&
      l.resource_type !== selectedResourceType.value
    )
      return false

    return true
  }),
)

const eventTypes = computed(() => {
  const set = new Set<string>(['all'])

  logs.value.forEach((l) => set.add(l.event_type))

  return Array.from(set)
})

const resourceTypes = computed(() => {
  const set = new Set<string>(['all'])

  logs.value.forEach((l) => l.resource_type && set.add(l.resource_type))

  return Array.from(set)
})
</script>

<template>
  <PageContainer class="flex h-full flex-col">
    <div class="flex items-start justify-between gap-1">
      <PageHeader title="Audit logs" />
      <TButton variant="highlighted" aria-haspopup="true" @click="downloadAuditLogModalOpen = true">
        Download audit logs
      </TButton>
    </div>

    <div class="mb-4 flex gap-2">
      <TInput v-model="searchInput" placeholder="Search..." icon="search" class="grow" />

      <TSelectList
        title="Event Type"
        :values="eventTypes"
        default-value="all"
        @checked-value="selectedEventType = $event"
      />

      <TSelectList
        title="Resource Type"
        :values="resourceTypes"
        default-value="all"
        @checked-value="selectedResourceType = $event"
      />

      <DateRangePicker v-model="dateRange" />
    </div>

    <TAlert
      v-if="
        dateRange.start &&
        dateRange.end &&
        differenceInDays(
          dateRange.end.toDate(getLocalTimeZone()),
          dateRange.start.toDate(getLocalTimeZone()),
        ) >= 7
      "
      type="warn"
      title="Long date range"
      class="mb-4"
    >
      When downloading audit logs for longer time periods the file size may be excessively large
      which can negatively impact Omni performance. Consider shortening the date range.
    </TAlert>

    <TableRoot>
      <template #head>
        <TableRow>
          <TableCell th>Date</TableCell>
          <TableCell th>Event</TableCell>
          <TableCell th>Session</TableCell>
          <TableCell th>Data</TableCell>
        </TableRow>
      </template>

      <template #body>
        <TableRow v-for="item in filteredLogs" :key="item.id">
          <TableCell class="whitespace-nowrap">
            {{ formatISO(new Date(item.event_ts).toISOString()) }}
          </TableCell>
          <TableCell>
            <div>
              <div>{{ item.event_type }}</div>
              <div>{{ item.resource_type }}</div>
              <div>{{ item.resource_id }}</div>
            </div>
          </TableCell>
          <TableCell>
            <pre class="whitespace-pre-wrap">{{ item.session }}</pre>
          </TableCell>
          <TableCell>
            <pre class="whitespace-pre-wrap">{{ item.event_data }}</pre>
          </TableCell>
        </TableRow>
      </template>
    </TableRoot>

    <DownloadAuditLogsModal v-model:open="downloadAuditLogModalOpen" />
  </PageContainer>
</template>
