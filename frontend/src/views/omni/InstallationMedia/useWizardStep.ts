// Copyright (c) 2025 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

import { computed, watch, type Ref } from 'vue'
import { useRouter } from 'vue-router'

import type { FormState } from './InstallationMediaCreate.vue'
import { getStepValidation } from './stepConfig'

/**
 * Composable for wizard steps to handle auto-skipping and validation
 *
 * This composable should be used by individual step components to:
 * 1. Report available options for auto-skip logic
 * 2. Automatically navigate to the next step if only one option is available
 * 3. Provide validation state for the current step
 */
export function useWizardStep(
  stepName: string,
  formState: Ref<FormState>,
  availableOptions?: Ref<number | undefined>,
) {
  const router = useRouter()

  // Validate current step with available options
  const validation = computed(() => {
    return getStepValidation(stepName, formState.value)
  })

  // Auto-skip logic: navigate to next step if this step should be skipped
  // This would be called when the component determines it has only one option
  function checkAutoSkip(nextStepName?: string) {
    if (!nextStepName || !availableOptions?.value) return

    // If only one option available and it's been auto-selected, move to next step
    if (availableOptions.value === 1 && validation.value.shouldSkip && validation.value.isValid) {
      // Small delay to allow the auto-selection to propagate
      setTimeout(() => {
        router.push({ name: nextStepName })
      }, 100)
    }
  }

  return {
    validation,
    checkAutoSkip,
    shouldAutoSkip: computed(() => validation.value.shouldSkip),
  }
}

/**
 * Helper to get the next step name from a flow
 */
export function getNextStepName(currentStepName: string, flow: string[]): string | undefined {
  const currentIndex = flow.indexOf(currentStepName)
  if (currentIndex === -1 || currentIndex === flow.length - 1) {
    return undefined
  }
  return flow[currentIndex + 1]
}
