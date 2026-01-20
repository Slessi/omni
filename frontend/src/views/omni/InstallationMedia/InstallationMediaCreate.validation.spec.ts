// Copyright (c) 2025 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.
import { render, screen, waitFor } from '@testing-library/vue'
import { beforeEach, describe, expect, test, vi } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'

vi.mock('@/notification', () => ({
  showSuccess: vi.fn(),
}))

import InstallationMediaCreate from './InstallationMediaCreate.vue'

const mockSavePresetModal = {
  name: 'SavePresetModal',
  template: '<div v-if="open" class="save-preset-modal"><slot /></div>',
  props: ['open', 'formState'],
  emits: ['close', 'saved'],
}

const routes = [
  {
    path: '/',
    name: 'InstallationMediaCreateEntry',
    component: { template: '<div>Entry</div>' },
  },
  {
    path: '/talos-version',
    name: 'InstallationMediaCreateTalosVersion',
    component: { template: '<div>Talos Version</div>' },
  },
  {
    path: '/machine-arch',
    name: 'InstallationMediaCreateMachineArch',
    component: { template: '<div>Machine Arch</div>' },
  },
  {
    path: '/cloud-provider',
    name: 'InstallationMediaCreateCloudProvider',
    component: { template: '<div>Cloud Provider</div>' },
  },
  {
    path: '/sbc-type',
    name: 'InstallationMediaCreateSBCType',
    component: { template: '<div>SBC Type</div>' },
  },
  {
    path: '/system-extensions',
    name: 'InstallationMediaCreateSystemExtensions',
    component: { template: '<div>System Extensions</div>' },
  },
  {
    path: '/extra-args',
    name: 'InstallationMediaCreateExtraArgs',
    component: { template: '<div>Extra Args</div>' },
  },
  {
    path: '/confirmation',
    name: 'InstallationMediaCreateConfirmation',
    component: { template: '<div>Confirmation</div>' },
  },
]

describe('InstallationMediaCreate - Validation Integration', () => {
  let router: ReturnType<typeof createRouter>

  beforeEach(() => {
    sessionStorage.clear()
    vi.clearAllMocks()
    router = createRouter({
      history: createMemoryHistory(),
      routes,
    })
  })

  test('Next button is disabled when current step is invalid', async () => {
    // Start at entry page without hardware type selected
    await router.push({ name: 'InstallationMediaCreateEntry' })
    await router.isReady()

    render(InstallationMediaCreate, {
      global: {
        plugins: [router],
        components: { SavePresetModal: mockSavePresetModal },
        stubs: {
          TIcon: true,
          TButton: { template: '<button v-bind="$attrs"><slot /></button>' },
          Stepper: true,
          Tooltip: true,
        },
      },
    })

    // Next button should be disabled when hardwareType is not set
    const nextButton = screen.queryByRole('link', { name: 'Next' })
    expect(nextButton).toBeInTheDocument()
    // Check if the button has disabled attribute/class
    expect(nextButton?.getAttribute('aria-disabled')).toBe('true')
  })

  test('Next button is enabled when current step is valid', async () => {
    // Set hardware type
    sessionStorage.setItem(
      '_installation_media_form',
      JSON.stringify({ hardwareType: 'metal' }),
    )

    await router.push({ name: 'InstallationMediaCreateEntry' })
    await router.isReady()

    render(InstallationMediaCreate, {
      global: {
        plugins: [router],
        components: { SavePresetModal: mockSavePresetModal },
        stubs: {
          TIcon: true,
          TButton: { template: '<button v-bind="$attrs"><slot /></button>' },
          Stepper: true,
          Tooltip: true,
        },
      },
    })

    // Next button should be enabled when hardwareType is set
    await waitFor(() => {
      const nextButton = screen.getByRole('link', { name: 'Next' })
      expect(nextButton?.getAttribute('aria-disabled')).not.toBe('true')
    })
  })

  test('Cannot navigate to future step when intermediate steps are invalid', async () => {
    // Set hardware type but not talosVersion
    sessionStorage.setItem(
      '_installation_media_form',
      JSON.stringify({ hardwareType: 'metal' }),
    )

    await router.push({ name: 'InstallationMediaCreateMachineArch' })
    await router.isReady()

    render(InstallationMediaCreate, {
      global: {
        plugins: [router],
        components: { SavePresetModal: mockSavePresetModal },
        stubs: {
          TIcon: true,
          TButton: true,
          Stepper: true,
          Tooltip: true,
        },
      },
    })

    // Should be redirected to TalosVersion step (first invalid step)
    await waitFor(() => {
      expect(router.currentRoute.value.name).toBe('InstallationMediaCreateTalosVersion')
    })
  })

  test('Can navigate to future step when all intermediate steps are valid', async () => {
    // Set all required fields for metal flow up to MachineArch
    sessionStorage.setItem(
      '_installation_media_form',
      JSON.stringify({
        hardwareType: 'metal',
        talosVersion: '1.8.0',
        joinToken: 'token-123',
        machineArch: 0, // AMD64
      }),
    )

    await router.push({ name: 'InstallationMediaCreateSystemExtensions' })
    await router.isReady()

    render(InstallationMediaCreate, {
      global: {
        plugins: [router],
        components: { SavePresetModal: mockSavePresetModal },
        stubs: {
          TIcon: true,
          TButton: true,
          Stepper: true,
          Tooltip: true,
        },
      },
    })

    // Should remain on SystemExtensions step
    expect(router.currentRoute.value.name).toBe('InstallationMediaCreateSystemExtensions')
  })

  test('Redirects to entry when accessing step without hardware type', async () => {
    // No hardware type set
    sessionStorage.clear()

    await router.push({ name: 'InstallationMediaCreateTalosVersion' })
    await router.isReady()

    render(InstallationMediaCreate, {
      global: {
        plugins: [router],
        components: { SavePresetModal: mockSavePresetModal },
        stubs: {
          TIcon: true,
          TButton: true,
          Stepper: true,
          Tooltip: true,
        },
      },
    })

    // Should be redirected to entry
    await waitFor(() => {
      expect(router.currentRoute.value.name).toBe('InstallationMediaCreateEntry')
    })
  })

  test('Maintains correct step count for different hardware types', async () => {
    const testCases = [
      { hardwareType: 'metal' as const, expectedSteps: 5 },
      { hardwareType: 'cloud' as const, expectedSteps: 6 },
      { hardwareType: 'sbc' as const, expectedSteps: 5 },
    ]

    for (const { hardwareType, expectedSteps } of testCases) {
      sessionStorage.clear()
      sessionStorage.setItem(
        '_installation_media_form',
        JSON.stringify({ hardwareType }),
      )

      await router.push({ name: 'InstallationMediaCreateEntry' })
      await router.isReady()

      const { unmount } = render(InstallationMediaCreate, {
        global: {
          plugins: [router],
          components: { SavePresetModal: mockSavePresetModal },
          stubs: {
            TIcon: true,
            TButton: true,
            Stepper: {
              template: '<div data-testid="stepper" :data-steps="stepCount">{{ stepCount }}</div>',
              props: ['modelValue', 'stepCount'],
            },
            Tooltip: true,
          },
        },
      })

      const stepper = screen.getByTestId('stepper')
      expect(stepper.getAttribute('data-steps')).toBe(String(expectedSteps))

      unmount()
    }
  })

  test('Next button disabled on TalosVersion step without required fields', async () => {
    sessionStorage.setItem(
      '_installation_media_form',
      JSON.stringify({
        hardwareType: 'metal',
        // talosVersion and joinToken missing
      }),
    )

    await router.push({ name: 'InstallationMediaCreateTalosVersion' })
    await router.isReady()

    render(InstallationMediaCreate, {
      global: {
        plugins: [router],
        components: { SavePresetModal: mockSavePresetModal },
        stubs: {
          TIcon: true,
          TButton: { template: '<button v-bind="$attrs"><slot /></button>' },
          Stepper: true,
          Tooltip: true,
        },
      },
    })

    // Next button should be disabled
    const nextButton = screen.queryByRole('link', { name: 'Next' })
    expect(nextButton?.getAttribute('aria-disabled')).toBe('true')
  })

  test('Next button enabled on optional steps', async () => {
    sessionStorage.setItem(
      '_installation_media_form',
      JSON.stringify({
        hardwareType: 'metal',
        talosVersion: '1.8.0',
        joinToken: 'token-123',
        machineArch: 0,
        // systemExtensions is optional
      }),
    )

    await router.push({ name: 'InstallationMediaCreateSystemExtensions' })
    await router.isReady()

    render(InstallationMediaCreate, {
      global: {
        plugins: [router],
        components: { SavePresetModal: mockSavePresetModal },
        stubs: {
          TIcon: true,
          TButton: { template: '<button v-bind="$attrs"><slot /></button>' },
          Stepper: true,
          Tooltip: true,
        },
      },
    })

    // Next button should be enabled even without selecting extensions
    await waitFor(() => {
      const nextButton = screen.getByRole('link', { name: 'Next' })
      expect(nextButton?.getAttribute('aria-disabled')).not.toBe('true')
    })
  })
})
