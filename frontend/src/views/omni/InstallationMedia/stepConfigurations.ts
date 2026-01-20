// Copyright (c) 2025 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

import { PlatformConfigSpecArch } from '@/api/omni/specs/virtual.pb'

import CloudProviderStep from './Steps/CloudProvider.vue'
import ConfirmationStep from './Steps/Confirmation.vue'
import ExtraArgsStep from './Steps/ExtraArgs.vue'
import MachineArchStep from './Steps/MachineArch.vue'
import SBCTypeStep from './Steps/SBCType.vue'
import SystemExtensionsStep from './Steps/SystemExtensions.vue'
import TalosVersionStep from './Steps/TalosVersion.vue'
import type { FormState } from './InstallationMediaCreate.vue'
import type { StepConfig } from './useWizardSteps'

/**
 * Step configurations for the metal flow
 */
export const metalFlowSteps: StepConfig[] = [
  {
    component: TalosVersionStep,
    isValid: (formState: FormState) => {
      return !!formState.talosVersion && !!formState.joinToken
    },
  },
  {
    component: MachineArchStep,
    isValid: (formState: FormState) => {
      return !!formState.machineArch
    },
    shouldSkip: (formState: FormState) => {
      // Skip if only one architecture is available
      // This would need actual architecture data, but we'll use a simple check for now
      return false
    },
    applyDefaults: (formState: FormState) => {
      // Default to AMD64 if not set
      if (!formState.machineArch && formState.hardwareType === 'metal') {
        formState.machineArch = PlatformConfigSpecArch.AMD64
      }
    },
    onDependencyChange: (formState: FormState) => {
      // If hardware type changes, reset architecture
      // This is a simple example - real logic would be more complex
    },
  },
  {
    component: SystemExtensionsStep,
    isValid: () => {
      // System extensions are optional, always valid
      return true
    },
  },
  {
    component: ExtraArgsStep,
    isValid: () => {
      // Extra args are optional, always valid
      return true
    },
  },
  {
    component: ConfirmationStep,
    isValid: () => {
      // Confirmation step is always valid
      return true
    },
  },
]

/**
 * Step configurations for the cloud flow
 */
export const cloudFlowSteps: StepConfig[] = [
  {
    component: TalosVersionStep,
    isValid: (formState: FormState) => {
      return !!formState.talosVersion && !!formState.joinToken
    },
  },
  {
    component: CloudProviderStep,
    isValid: (formState: FormState) => {
      return !!formState.cloudPlatform
    },
    onDependencyChange: (formState: FormState) => {
      // Reset cloud platform if Talos version changes and platform is no longer compatible
      // This is a placeholder - real implementation would check compatibility
    },
  },
  {
    component: MachineArchStep,
    isValid: (formState: FormState) => {
      return !!formState.machineArch
    },
    shouldSkip: (formState: FormState) => {
      // Skip if only one architecture is available for the selected cloud platform
      // This is a placeholder - real implementation would check available architectures
      return false
    },
    applyDefaults: (formState: FormState) => {
      // Default to AMD64 if not set
      if (!formState.machineArch && formState.hardwareType === 'cloud') {
        formState.machineArch = PlatformConfigSpecArch.AMD64
      }
    },
    onDependencyChange: (formState: FormState) => {
      // Reset architecture if cloud platform changes and architecture is no longer available
      // This is a placeholder - real implementation would check available architectures
    },
  },
  {
    component: SystemExtensionsStep,
    isValid: () => {
      // System extensions are optional, always valid
      return true
    },
  },
  {
    component: ExtraArgsStep,
    isValid: () => {
      // Extra args are optional, always valid
      return true
    },
  },
  {
    component: ConfirmationStep,
    isValid: () => {
      // Confirmation step is always valid
      return true
    },
  },
]

/**
 * Step configurations for the SBC flow
 */
export const sbcFlowSteps: StepConfig[] = [
  {
    component: TalosVersionStep,
    isValid: (formState: FormState) => {
      return !!formState.talosVersion && !!formState.joinToken
    },
  },
  {
    component: SBCTypeStep,
    isValid: (formState: FormState) => {
      return !!formState.sbcType
    },
  },
  {
    component: SystemExtensionsStep,
    isValid: () => {
      // System extensions are optional, always valid
      return true
    },
  },
  {
    component: ExtraArgsStep,
    isValid: () => {
      // Extra args are optional, always valid
      return true
    },
  },
  {
    component: ConfirmationStep,
    isValid: () => {
      // Confirmation step is always valid
      return true
    },
  },
]

export type HardwareType = 'metal' | 'cloud' | 'sbc'

/**
 * Get step configurations for a given hardware type
 */
export function getStepsForHardwareType(hardwareType: HardwareType | undefined): StepConfig[] {
  if (!hardwareType) {
    return []
  }

  switch (hardwareType) {
    case 'metal':
      return metalFlowSteps
    case 'cloud':
      return cloudFlowSteps
    case 'sbc':
      return sbcFlowSteps
    default:
      return []
  }
}
