// Copyright (c) 2025 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

import { describe, expect, it } from 'vitest'
import { computed, ref } from 'vue'

import { PlatformConfigSpecArch } from '@/api/omni/specs/virtual.pb'

import type { FormState } from './InstallationMediaCreate.vue'
import type { StepConfig } from './useWizardSteps'
import { useWizardSteps } from './useWizardSteps'

// Mock components for testing
const MockComponent1 = { name: 'MockComponent1' }
const MockComponent2 = { name: 'MockComponent2' }
const MockComponent3 = { name: 'MockComponent3' }

describe('useWizardSteps', () => {
  it('should validate steps correctly', () => {
    const formState = ref<FormState>({
      currentStep: 0,
      talosVersion: 'v1.0.0',
      joinToken: 'token123',
    })

    const steps = computed<StepConfig[]>(() => [
      {
        component: MockComponent1,
        isValid: (state: FormState) => !!state.talosVersion && !!state.joinToken,
      },
      {
        component: MockComponent2,
        isValid: (state: FormState) => !!state.machineArch,
      },
      {
        component: MockComponent3,
        isValid: () => true,
      },
    ])

    const currentStepIndex = ref(0)
    const { isCurrentStepValid, isStepAccessible } = useWizardSteps(
      steps,
      formState,
      currentStepIndex,
    )

    // First step should be valid
    expect(isCurrentStepValid.value).toBe(true)

    // Second step should be accessible (first step is valid)
    expect(isStepAccessible(1)).toBe(true)

    // Third step should not be accessible (second step is invalid)
    expect(isStepAccessible(2)).toBe(false)

    // Add machine arch
    formState.value.machineArch = PlatformConfigSpecArch.AMD64

    // Now third step should be accessible
    expect(isStepAccessible(2)).toBe(true)
  })

  it('should skip steps correctly', () => {
    const formState = ref<FormState>({
      currentStep: 0,
      hardwareType: 'sbc',
    })

    const steps = computed<StepConfig[]>(() => [
      {
        component: MockComponent1,
        isValid: () => true,
      },
      {
        component: MockComponent2,
        isValid: () => true,
        shouldSkip: (state: FormState) => state.hardwareType === 'sbc',
      },
      {
        component: MockComponent3,
        isValid: () => true,
      },
    ])

    const currentStepIndex = ref(0)
    const { getNextVisibleStep, totalVisibleSteps } = useWizardSteps(
      steps,
      formState,
      currentStepIndex,
    )

    // Should skip step 1 (index 1)
    expect(getNextVisibleStep(0, 1)).toBe(0) // First visible step
    expect(getNextVisibleStep(1, 1)).toBe(2) // Skip step 1, go to step 2

    // Total visible steps should be 2 (step 0 and 2)
    expect(totalVisibleSteps.value).toBe(2)
  })

  it('should navigate to next and previous steps', () => {
    const formState = ref<FormState>({
      currentStep: 0,
    })

    const steps = computed<StepConfig[]>(() => [
      {
        component: MockComponent1,
        isValid: () => true,
      },
      {
        component: MockComponent2,
        isValid: () => true,
      },
      {
        component: MockComponent3,
        isValid: () => true,
      },
    ])

    const currentStepIndex = ref(0)
    const { goToNextStep, goToPreviousStep } = useWizardSteps(steps, formState, currentStepIndex)

    // Navigate forward
    goToNextStep()
    expect(currentStepIndex.value).toBe(1)

    goToNextStep()
    expect(currentStepIndex.value).toBe(2)

    // Navigate backward
    goToPreviousStep()
    expect(currentStepIndex.value).toBe(1)

    goToPreviousStep()
    expect(currentStepIndex.value).toBe(0)
  })

  it('should apply defaults when navigating to a step', () => {
    const formState = ref<FormState>({
      currentStep: 0,
    })

    let defaultsApplied = false

    const steps = computed<StepConfig[]>(() => [
      {
        component: MockComponent1,
        isValid: () => true,
        applyDefaults: (state: FormState) => {
          defaultsApplied = true
          state.talosVersion = 'v1.0.0'
        },
      },
    ])

    const currentStepIndex = ref(0)
    const { navigateToStep } = useWizardSteps(steps, formState, currentStepIndex)

    // Navigate to step 0
    navigateToStep(0)

    expect(defaultsApplied).toBe(true)
    expect(formState.value.talosVersion).toBe('v1.0.0')
  })

  it('should check if step is accessible based on previous step validity', () => {
    const formState = ref<FormState>({
      currentStep: 0,
      talosVersion: '',
    })

    const steps = computed<StepConfig[]>(() => [
      {
        component: MockComponent1,
        isValid: (state: FormState) => !!state.talosVersion,
      },
      {
        component: MockComponent2,
        isValid: () => true,
      },
    ])

    const currentStepIndex = ref(0)
    const { isStepAccessible } = useWizardSteps(steps, formState, currentStepIndex)

    // Step 1 should not be accessible (step 0 is invalid)
    expect(isStepAccessible(1)).toBe(false)

    // Set talos version
    formState.value.talosVersion = 'v1.0.0'

    // Now step 1 should be accessible
    expect(isStepAccessible(1)).toBe(true)
  })
})
