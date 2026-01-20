// Copyright (c) 2025 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

import { type Component, computed, type Ref, watch } from 'vue'

import type { FormState } from './InstallationMediaCreate.vue'

/**
 * Configuration for a wizard step
 */
export interface StepConfig {
  /** The Vue component for this step */
  component: Component

  /**
   * Validates the current step
   * @param formState Current form state
   * @returns true if valid, false otherwise
   */
  isValid?: (formState: FormState) => boolean

  /**
   * Determines if this step should be skipped
   * @param formState Current form state
   * @returns true if step should be skipped, false otherwise
   */
  shouldSkip?: (formState: FormState) => boolean

  /**
   * Applies default values when entering this step
   * @param formState Current form state
   */
  applyDefaults?: (formState: FormState) => void

  /**
   * Called when dependencies change that might invalidate this step
   * @param formState Current form state
   */
  onDependencyChange?: (formState: FormState) => void
}

/**
 * Composable for managing wizard step navigation and validation
 */
export function useWizardSteps(
  steps: Ref<StepConfig[]>,
  formState: Ref<FormState>,
  currentStepIndex: Ref<number>,
) {
  /**
   * Get the actual step index, skipping any steps that should be skipped
   */
  const getNextVisibleStep = (fromIndex: number, direction: 1 | -1): number => {
    let index = fromIndex
    const maxSteps = steps.value.length

    while (index >= 0 && index < maxSteps) {
      const step = steps.value[index]
      if (!step.shouldSkip || !step.shouldSkip(formState.value)) {
        return index
      }
      index += direction
    }

    return direction === 1 ? maxSteps : -1
  }

  /**
   * Check if the current step is valid
   */
  const isCurrentStepValid = computed(() => {
    const step = steps.value[currentStepIndex.value]
    if (!step?.isValid) {
      return true // No validation means always valid
    }
    return step.isValid(formState.value)
  })

  /**
   * Check if a specific step is accessible (all previous steps are valid)
   */
  const isStepAccessible = (targetIndex: number): boolean => {
    if (targetIndex < 0 || targetIndex >= steps.value.length) {
      return false
    }

    // Check all steps before the target
    for (let i = 0; i < targetIndex; i++) {
      const step = steps.value[i]
      
      // Skip steps that should be skipped
      if (step.shouldSkip && step.shouldSkip(formState.value)) {
        continue
      }

      // Check if step is valid
      if (step.isValid && !step.isValid(formState.value)) {
        return false
      }
    }

    return true
  }

  /**
   * Navigate to the next visible step
   */
  const goToNextStep = () => {
    const nextIndex = getNextVisibleStep(currentStepIndex.value + 1, 1)
    if (nextIndex >= 0 && nextIndex < steps.value.length) {
      navigateToStep(nextIndex)
    }
  }

  /**
   * Navigate to the previous visible step
   */
  const goToPreviousStep = () => {
    const prevIndex = getNextVisibleStep(currentStepIndex.value - 1, -1)
    if (prevIndex >= 0) {
      navigateToStep(prevIndex)
    }
  }

  /**
   * Navigate to a specific step
   */
  const navigateToStep = (index: number) => {
    if (index < 0 || index >= steps.value.length) {
      return
    }

    // Apply defaults for the new step
    const step = steps.value[index]
    if (step.applyDefaults) {
      step.applyDefaults(formState.value)
    }

    currentStepIndex.value = index
  }

  /**
   * Navigate to a step if it's accessible
   */
  const tryNavigateToStep = (index: number): boolean => {
    if (!isStepAccessible(index)) {
      return false
    }

    navigateToStep(index)
    return true
  }

  /**
   * Get the visible step number (1-indexed, accounting for skipped steps)
   */
  const getVisibleStepNumber = (index: number): number => {
    let visibleCount = 0
    for (let i = 0; i <= index && i < steps.value.length; i++) {
      const step = steps.value[i]
      if (!step.shouldSkip || !step.shouldSkip(formState.value)) {
        visibleCount++
      }
    }
    return visibleCount
  }

  /**
   * Get the total number of visible steps
   */
  const totalVisibleSteps = computed(() => {
    let count = 0
    for (const step of steps.value) {
      if (!step.shouldSkip || !step.shouldSkip(formState.value)) {
        count++
      }
    }
    return count
  })

  /**
   * Watch for changes in key form state properties that might affect step validity
   * and reset dependent steps if needed
   */
  watch(
    [
      () => formState.value.hardwareType,
      () => formState.value.talosVersion,
      () => formState.value.cloudPlatform,
      () => formState.value.sbcType,
    ],
    () => {
      // Call onDependencyChange for all steps after the current one
      for (let i = currentStepIndex.value + 1; i < steps.value.length; i++) {
        const step = steps.value[i]
        if (step.onDependencyChange) {
          step.onDependencyChange(formState.value)
        }
      }
    },
  )

  return {
    isCurrentStepValid,
    isStepAccessible,
    goToNextStep,
    goToPreviousStep,
    navigateToStep,
    tryNavigateToStep,
    getVisibleStepNumber,
    totalVisibleSteps,
    getNextVisibleStep,
  }
}
