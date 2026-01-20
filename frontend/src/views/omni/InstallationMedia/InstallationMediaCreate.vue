<!--
Copyright (c) 2025 Sidero Labs, Inc.

Use of this software is governed by the Business Source License
included in the LICENSE file.
-->
<script lang="ts">
import type { SchematicBootloader } from '@/api/omni/management/management.pb'
import { PlatformConfigSpecArch } from '@/api/omni/specs/virtual.pb'
import type { LabelSelectItem } from '@/components/common/Labels/Labels.vue'

import type { HardwareType } from './types'

export interface FormState {
  currentStep: number
  hardwareType?: HardwareType
  talosVersion?: string
  useGrpcTunnel?: boolean
  joinToken?: string
  machineUserLabels?: Record<string, LabelSelectItem>
  machineArch?: PlatformConfigSpecArch
  secureBoot?: boolean
  cloudPlatform?: string
  sbcType?: string
  systemExtensions?: string[]
  cmdline?: string
  overlayOptions?: string
  bootloader?: SchematicBootloader
}
</script>

<script setup lang="ts">
import { useSessionStorage } from '@vueuse/core'
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'

import TButton from '@/components/common/Button/TButton.vue'
import Stepper from '@/components/common/Stepper/Stepper.vue'
import { showSuccess } from '@/notification'
import SavePresetModal from '@/views/omni/InstallationMedia/SavePresetModal.vue'
import EntryStep from '@/views/omni/InstallationMedia/Steps/Entry.vue'
import { getStepsForHardwareType } from '@/views/omni/InstallationMedia/stepConfigurations'
import { useWizardSteps } from '@/views/omni/InstallationMedia/useWizardSteps'

const router = useRouter()

const formState = useSessionStorage<FormState>(
  '_installation_media_form',
  { currentStep: 0 },
  { writeDefaults: false },
)

// Get current flow steps based on hardware type
const currentFlowSteps = computed(() => getStepsForHardwareType(formState.value.hardwareType))

// Calculate the actual step index (0-based index into the steps array)
const actualStepIndex = computed({
  get: () => Math.max(0, formState.value.currentStep - 1),
  set: (value) => {
    formState.value.currentStep = value + 1
  },
})

// Use the wizard steps composable
const {
  isCurrentStepValid,
  isStepAccessible,
  goToNextStep: navigateToNextStep,
  goToPreviousStep: navigateToPreviousStep,
  tryNavigateToStep,
  totalVisibleSteps,
  getVisibleStepNumber,
  getNextVisibleStep,
} = useWizardSteps(currentFlowSteps, formState, actualStepIndex)

// Current step component
const currentStepComponent = computed(() => {
  if (!formState.value.hardwareType || formState.value.currentStep === 0) {
    return EntryStep
  }
  
  const steps = currentFlowSteps.value
  const index = actualStepIndex.value
  
  if (index >= 0 && index < steps.length) {
    return steps[index].component
  }
  
  return EntryStep
})

// Check if we're at the last step
const isLastStep = computed(() => {
  if (!formState.value.hardwareType || formState.value.currentStep === 0) {
    return false
  }
  
  // Find the last non-skipped step
  const steps = currentFlowSteps.value
  for (let i = steps.length - 1; i >= 0; i--) {
    const step = steps[i]
    if (!step.shouldSkip || !step.shouldSkip(formState.value)) {
      return actualStepIndex.value === i
    }
  }
  
  return false
})

// Count of total steps (for display)
const stepCount = computed(() => totalVisibleSteps.value)

// Can go back
const canGoBack = computed(() => {
  if (formState.value.currentStep <= 0) {
    return false
  }
  
  // Find previous non-skipped step
  const prevIndex = getNextVisibleStep(actualStepIndex.value - 1, -1)
  return prevIndex >= 0
})

// Can go forward (current step is valid)
const canGoNext = computed(() => {
  if (formState.value.currentStep === 0) {
    // Entry step - can proceed if hardware type is selected
    return !!formState.value.hardwareType
  }
  
  return isCurrentStepValid.value
})

// Handle back button
function goBackStep() {
  if (formState.value.currentStep === 0) {
    return
  }
  
  if (formState.value.hardwareType && formState.value.currentStep > 0) {
    navigateToPreviousStep()
  } else {
    // Go back to entry step
    formState.value.currentStep = 0
  }
}

// Handle next button
function goNextStep() {
  if (formState.value.currentStep === 0) {
    // Moving from entry to first actual step
    if (formState.value.hardwareType) {
      formState.value.currentStep = 1
      
      // Apply defaults for the first step
      const steps = currentFlowSteps.value
      if (steps.length > 0) {
        const firstStep = getNextVisibleStep(0, 1)
        if (firstStep >= 0 && steps[firstStep].applyDefaults) {
          steps[firstStep].applyDefaults(formState.value)
        }
      }
    }
  } else {
    navigateToNextStep()
  }
}

// Handle clicking on a step in the stepper
const handleStepAccessibility = (step: number): boolean => {
  return isStepAccessible(step - 1)
}

function onStepClick(stepNumber: number) {
  if (stepNumber === formState.value.currentStep) {
    return // Already on this step
  }
  
  const targetIndex = stepNumber - 1
  
  // Try to navigate to the step
  tryNavigateToStep(targetIndex)
}

const savePresetModalOpen = ref(false)
const isSaved = ref(false)

function openSavePresetModal() {
  savePresetModalOpen.value = true
}

function goToPresetList() {
  formState.value = { currentStep: 0 }
  router.push({ name: 'InstallationMedia' })
}

function onSaved(name: string) {
  showSuccess(`Preset saved as ${name}`)

  isSaved.value = true
  savePresetModalOpen.value = false
}
</script>

<template>
  <div class="flex h-full flex-col">
    <div class="flex grow flex-col gap-6 overflow-auto p-6">
      <h1 class="shrink-0 text-xl font-medium text-naturals-n14">Create New Media</h1>

      <component :is="currentStepComponent" v-model="formState" />
    </div>

    <div
      class="flex w-full shrink-0 items-center gap-4 border-t border-naturals-n4 bg-naturals-n1 px-4 max-md:flex-col max-md:p-4 md:h-16 md:justify-end"
    >
      <Stepper
        v-if="currentFlowSteps.length > 0 && formState.currentStep > 0"
        v-model="formState.currentStep"
        :step-count="stepCount"
        :is-step-accessible="handleStepAccessibility"
        class="mx-auto w-full"
        @update:model-value="onStepClick"
      />

      <div class="flex items-center gap-2 max-md:self-end">
        <TButton
          v-if="formState.currentStep > 0"
          :disabled="!canGoBack || isSaved"
          @click="goBackStep"
        >
          Back
        </TButton>

        <TButton
          type="highlighted"
          :disabled="!canGoNext && formState.currentStep > 0"
          @click="isLastStep ? (isSaved ? goToPresetList() : openSavePresetModal()) : goNextStep()"
        >
          {{ isLastStep ? (isSaved ? 'Finished' : 'Save') : 'Next' }}
        </TButton>
      </div>
    </div>

    <SavePresetModal
      :open="savePresetModalOpen"
      :form-state
      @close="savePresetModalOpen = false"
      @saved="onSaved"
    />
  </div>
</template>
