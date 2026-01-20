// Copyright (c) 2025 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

import type { FormState } from './InstallationMediaCreate.vue'

/**
 * Result of a step validation
 */
export interface StepValidationResult {
  /** Whether the step is valid (all required fields are filled) */
  isValid: boolean
  /** Whether the step should be skipped (e.g., only one option available) */
  shouldSkip: boolean
  /** Optional validation error message */
  errorMessage?: string
  /** Number of available options for this step (used for auto-skip logic) */
  availableOptions?: number
}

/**
 * Configuration for a wizard step
 */
export interface StepConfig {
  /** Unique route name for this step */
  name: string
  /** Function to validate the current form state for this step */
  validate: (formState: FormState) => StepValidationResult
  /** Function to apply default values for this step */
  applyDefaults?: (formState: FormState) => void
  /** Fields that this step depends on (for reset logic) */
  dependencies?: (keyof FormState)[]
}

/**
 * Validates the Entry step (hardware type selection)
 */
export function validateEntry(formState: FormState): StepValidationResult {
  const isValid = !!formState.hardwareType
  return {
    isValid,
    shouldSkip: false,
    errorMessage: isValid ? undefined : 'Hardware type is required',
  }
}

/**
 * Validates the TalosVersion step
 */
export function validateTalosVersion(formState: FormState): StepValidationResult {
  const isValid = !!formState.talosVersion && !!formState.joinToken
  return {
    isValid,
    shouldSkip: false,
    errorMessage: isValid ? undefined : 'Talos version and join token are required',
  }
}

/**
 * Validates the CloudProvider step
 * @param formState Current form state
 * @param availableProviders Optional array of available provider IDs for auto-skip logic
 * 
 * Note: This function intentionally mutates formState to auto-select the provider
 * when only one option is available. This is the desired behavior for the wizard
 * to automatically skip single-option steps.
 */
export function validateCloudProvider(
  formState: FormState,
  availableProviders?: string[],
): StepValidationResult {
  const isValid = !!formState.cloudPlatform
  const shouldSkip = availableProviders !== undefined && availableProviders.length === 1

  // Auto-select if only one option available
  if (shouldSkip && !formState.cloudPlatform && availableProviders.length === 1) {
    formState.cloudPlatform = availableProviders[0]
  }

  return {
    isValid: isValid || shouldSkip,
    shouldSkip,
    errorMessage: isValid || shouldSkip ? undefined : 'Cloud platform is required',
    availableOptions: availableProviders?.length,
  }
}

/**
 * Validates the SBCType step
 * @param formState Current form state
 * @param availableSBCs Optional array of available SBC type IDs for auto-skip logic
 * 
 * Note: This function intentionally mutates formState to auto-select the SBC type
 * when only one option is available. This is the desired behavior for the wizard
 * to automatically skip single-option steps.
 */
export function validateSBCType(
  formState: FormState,
  availableSBCs?: string[],
): StepValidationResult {
  const isValid = !!formState.sbcType
  const shouldSkip = availableSBCs !== undefined && availableSBCs.length === 1

  // Auto-select if only one option available
  if (shouldSkip && !formState.sbcType && availableSBCs.length === 1) {
    formState.sbcType = availableSBCs[0]
  }

  return {
    isValid: isValid || shouldSkip,
    shouldSkip,
    errorMessage: isValid || shouldSkip ? undefined : 'SBC type is required',
    availableOptions: availableSBCs?.length,
  }
}

/**
 * Validates the MachineArch step
 * @param formState Current form state
 * @param availableArchitectures Optional array of available architectures for auto-skip logic
 * 
 * Note: This function intentionally mutates formState to auto-select the architecture
 * when only one option is available. This is the desired behavior for the wizard
 * to automatically skip single-option steps.
 */
export function validateMachineArch(
  formState: FormState,
  availableArchitectures?: number[],
): StepValidationResult {
  const isValid = !!formState.machineArch
  const shouldSkip = availableArchitectures !== undefined && availableArchitectures.length === 1

  // Auto-select if only one option available
  if (shouldSkip && !formState.machineArch && availableArchitectures.length === 1) {
    formState.machineArch = availableArchitectures[0]
  }

  return {
    isValid: isValid || shouldSkip,
    shouldSkip,
    errorMessage: isValid || shouldSkip ? undefined : 'Machine architecture is required',
    availableOptions: availableArchitectures?.length,
  }
}

/**
 * Validates the SystemExtensions step
 * This step is always valid as extensions are optional
 */
export function validateSystemExtensions(_formState: FormState): StepValidationResult {
  return {
    isValid: true,
    shouldSkip: false,
  }
}

/**
 * Validates the ExtraArgs step
 * This step is always valid as extra args are optional
 */
export function validateExtraArgs(_formState: FormState): StepValidationResult {
  return {
    isValid: true,
    shouldSkip: false,
  }
}

/**
 * Validates the Confirmation step
 * This step is always valid if we've reached it
 */
export function validateConfirmation(_formState: FormState): StepValidationResult {
  return {
    isValid: true,
    shouldSkip: false,
  }
}

/**
 * Step configurations for all wizard steps
 */
export const stepConfigs: Record<string, StepConfig> = {
  InstallationMediaCreateEntry: {
    name: 'InstallationMediaCreateEntry',
    validate: validateEntry,
  },
  InstallationMediaCreateTalosVersion: {
    name: 'InstallationMediaCreateTalosVersion',
    validate: validateTalosVersion,
    dependencies: ['hardwareType'],
  },
  InstallationMediaCreateCloudProvider: {
    name: 'InstallationMediaCreateCloudProvider',
    validate: validateCloudProvider,
    dependencies: ['talosVersion'],
  },
  InstallationMediaCreateSBCType: {
    name: 'InstallationMediaCreateSBCType',
    validate: validateSBCType,
    dependencies: ['talosVersion'],
  },
  InstallationMediaCreateMachineArch: {
    name: 'InstallationMediaCreateMachineArch',
    validate: validateMachineArch,
    dependencies: ['hardwareType', 'cloudPlatform', 'sbcType'],
  },
  InstallationMediaCreateSystemExtensions: {
    name: 'InstallationMediaCreateSystemExtensions',
    validate: validateSystemExtensions,
    dependencies: ['talosVersion'],
  },
  InstallationMediaCreateExtraArgs: {
    name: 'InstallationMediaCreateExtraArgs',
    validate: validateExtraArgs,
  },
  InstallationMediaCreateConfirmation: {
    name: 'InstallationMediaCreateConfirmation',
    validate: validateConfirmation,
  },
}

/**
 * Get the validation result for a specific step
 */
export function getStepValidation(stepName: string, formState: FormState): StepValidationResult {
  const config = stepConfigs[stepName]
  if (!config) {
    return { isValid: false, shouldSkip: false, errorMessage: 'Unknown step' }
  }
  return config.validate(formState)
}

/**
 * Check if all steps up to (but not including) the target step are valid
 */
export function areIntermediateStepsValid(
  steps: string[],
  targetStepIndex: number,
  formState: FormState,
): boolean {
  for (let i = 0; i < targetStepIndex; i++) {
    const validation = getStepValidation(steps[i], formState)
    if (!validation.isValid) {
      return false
    }
  }
  return true
}

/**
 * Find the first invalid step in a flow
 */
export function findFirstInvalidStep(steps: string[], formState: FormState): number {
  for (let i = 0; i < steps.length; i++) {
    const validation = getStepValidation(steps[i], formState)
    if (!validation.isValid) {
      return i
    }
  }
  return -1 // All steps are valid
}

/**
 * Reset form state fields that depend on changed fields
 */
export function resetDependentSteps(
  changedField: keyof FormState,
  formState: FormState,
  allSteps: string[],
): void {
  // Find all steps that depend on this field
  for (const stepName of allSteps) {
    const config = stepConfigs[stepName]
    if (config?.dependencies?.includes(changedField)) {
      // Reset fields affected by this step
      resetStepFields(stepName, formState)
    }
  }
}

/**
 * Reset form state fields for a specific step
 * 
 * Note: This uses a switch statement for simplicity. For a larger number of steps,
 * consider moving reset logic to the StepConfig object with a resetFields property.
 */
function resetStepFields(stepName: string, formState: FormState): void {
  switch (stepName) {
    case 'InstallationMediaCreateTalosVersion':
      delete formState.talosVersion
      delete formState.joinToken
      break
    case 'InstallationMediaCreateCloudProvider':
      delete formState.cloudPlatform
      break
    case 'InstallationMediaCreateSBCType':
      delete formState.sbcType
      break
    case 'InstallationMediaCreateMachineArch':
      delete formState.machineArch
      delete formState.secureBoot
      break
    case 'InstallationMediaCreateSystemExtensions':
      delete formState.systemExtensions
      break
    case 'InstallationMediaCreateExtraArgs':
      delete formState.cmdline
      delete formState.overlayOptions
      delete formState.bootloader
      break
  }
}
