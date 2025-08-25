import { flushPromises, mount } from '@vue/test-utils'
import { expect, test } from 'vitest'

import { Runtime } from '@/api/common/omni.pb'
import { DefaultNamespace, MachineType } from '@/api/resources'

import Watch from './Watch.vue'

test('fetch user on mount', async () => {
  const wrapper = mount(Watch, {
    props: {
      opts: {
        runtime: Runtime.Omni,
        resource: {
          id: 'test-id',
          type: MachineType,
          namespace: DefaultNamespace,
        },
      },
    },
  })

  expect(wrapper.vm.opts).toBeUndefined()

  await flushPromises()

  expect(wrapper.vm.opts).toEqual({ id: 1, name: 'User' })
})
