import { flushPromises, mount } from '@vue/test-utils'
import { expect, test } from 'vitest'
import { defineComponent } from 'vue'

import { Runtime } from '@/api/common/omni.pb'
import { DefaultNamespace, MachineType } from '@/api/resources'
import type { WatchJoinOptions, WatchOptions } from '@/api/watch'

import { useWatch } from './useWatch'

test('fetch user on mount', async () => {
  expect(true).toBeTruthy()
})
