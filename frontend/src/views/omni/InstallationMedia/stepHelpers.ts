// Copyright (c) 2025 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.

/**
 * Helper functions for step validation and configuration
 * These would ideally use actual API data, but for demonstration
 * purposes we use simplified logic
 */

import { PlatformConfigSpecArch } from '@/api/omni/specs/virtual.pb'

import type { FormState } from './InstallationMediaCreate.vue'

/**
 * Get available architectures for the current hardware configuration
 * This is a simplified version - real implementation would fetch from API
 */
export function getAvailableArchitectures(formState: FormState): PlatformConfigSpecArch[] {
  if (formState.hardwareType === 'sbc') {
    // SBCs only support ARM64
    return [PlatformConfigSpecArch.ARM64]
  }

  // Metal and cloud typically support both, but this would come from API
  return [PlatformConfigSpecArch.AMD64, PlatformConfigSpecArch.ARM64]
}

/**
 * Check if the architecture step should be skipped
 * Skip if only one architecture is available
 */
export function shouldSkipArchitectureStep(formState: FormState): boolean {
  const availableArchs = getAvailableArchitectures(formState)
  return availableArchs.length === 1
}

/**
 * Apply default architecture based on available options
 */
export function applyDefaultArchitecture(formState: FormState): void {
  const availableArchs = getAvailableArchitectures(formState)
  
  // If only one architecture is available, auto-select it
  if (availableArchs.length === 1) {
    formState.machineArch = availableArchs[0]
  } else if (!formState.machineArch && availableArchs.includes(PlatformConfigSpecArch.AMD64)) {
    // Default to AMD64 if available and nothing is selected
    formState.machineArch = PlatformConfigSpecArch.AMD64
  }
}

/**
 * Reset architecture if it's no longer valid for the current configuration
 */
export function resetArchitectureIfInvalid(formState: FormState): void {
  if (!formState.machineArch) {
    return
  }

  const availableArchs = getAvailableArchitectures(formState)
  if (!availableArchs.includes(formState.machineArch)) {
    formState.machineArch = undefined
  }
}

/**
 * Check if the cloud platform is still compatible with the selected Talos version
 * This is a simplified check - real implementation would use actual API data
 */
export function isCloudPlatformCompatible(formState: FormState): boolean {
  // In a real implementation, this would check the cloud platform's min_version
  // against the selected Talos version
  return true
}

/**
 * Reset cloud platform if it's no longer compatible
 */
export function resetCloudPlatformIfIncompatible(formState: FormState): void {
  if (formState.cloudPlatform && !isCloudPlatformCompatible(formState)) {
    formState.cloudPlatform = undefined
  }
}
