// Copyright (c) 2025 Sidero Labs, Inc.
//
// Use of this software is governed by the Business Source License
// included in the LICENSE file.
import { describe, expect, test } from 'vitest'

import type { FormState } from './InstallationMediaCreate.vue'
import {
  areIntermediateStepsValid,
  findFirstInvalidStep,
  getStepValidation,
  validateCloudProvider,
  validateConfirmation,
  validateEntry,
  validateExtraArgs,
  validateMachineArch,
  validateSBCType,
  validateSystemExtensions,
  validateTalosVersion,
} from './stepConfig'

describe('stepConfig validators', () => {
  describe('validateEntry', () => {
    test('returns invalid when hardwareType is missing', () => {
      const formState: FormState = {}
      const result = validateEntry(formState)

      expect(result.isValid).toBe(false)
      expect(result.shouldSkip).toBe(false)
      expect(result.errorMessage).toBe('Hardware type is required')
    })

    test('returns valid when hardwareType is set', () => {
      const formState: FormState = { hardwareType: 'metal' }
      const result = validateEntry(formState)

      expect(result.isValid).toBe(true)
      expect(result.shouldSkip).toBe(false)
      expect(result.errorMessage).toBeUndefined()
    })
  })

  describe('validateTalosVersion', () => {
    test('returns invalid when talosVersion is missing', () => {
      const formState: FormState = { joinToken: 'token-123' }
      const result = validateTalosVersion(formState)

      expect(result.isValid).toBe(false)
      expect(result.errorMessage).toBe('Talos version and join token are required')
    })

    test('returns invalid when joinToken is missing', () => {
      const formState: FormState = { talosVersion: '1.8.0' }
      const result = validateTalosVersion(formState)

      expect(result.isValid).toBe(false)
      expect(result.errorMessage).toBe('Talos version and join token are required')
    })

    test('returns valid when both talosVersion and joinToken are set', () => {
      const formState: FormState = {
        talosVersion: '1.8.0',
        joinToken: 'token-123',
      }
      const result = validateTalosVersion(formState)

      expect(result.isValid).toBe(true)
      expect(result.shouldSkip).toBe(false)
    })
  })

  describe('validateCloudProvider', () => {
    test('returns invalid when cloudPlatform is missing', () => {
      const formState: FormState = {}
      const result = validateCloudProvider(formState)

      expect(result.isValid).toBe(false)
      expect(result.errorMessage).toBe('Cloud platform is required')
    })

    test('returns valid when cloudPlatform is set', () => {
      const formState: FormState = { cloudPlatform: 'aws' }
      const result = validateCloudProvider(formState)

      expect(result.isValid).toBe(true)
      expect(result.shouldSkip).toBe(false)
    })

    test('auto-skips when only one provider available', () => {
      const formState: FormState = {}
      const result = validateCloudProvider(formState, ['aws'])

      expect(result.shouldSkip).toBe(true)
      expect(result.isValid).toBe(true)
      expect(result.availableOptions).toBe(1)
      expect(formState.cloudPlatform).toBe('aws')
    })

    test('does not auto-skip when multiple providers available', () => {
      const formState: FormState = {}
      const result = validateCloudProvider(formState, ['aws', 'gcp'])

      expect(result.shouldSkip).toBe(false)
      expect(result.availableOptions).toBe(2)
    })
  })

  describe('validateSBCType', () => {
    test('returns invalid when sbcType is missing', () => {
      const formState: FormState = {}
      const result = validateSBCType(formState)

      expect(result.isValid).toBe(false)
      expect(result.errorMessage).toBe('SBC type is required')
    })

    test('returns valid when sbcType is set', () => {
      const formState: FormState = { sbcType: 'rpi4' }
      const result = validateSBCType(formState)

      expect(result.isValid).toBe(true)
      expect(result.shouldSkip).toBe(false)
    })

    test('auto-skips when only one SBC available', () => {
      const formState: FormState = {}
      const result = validateSBCType(formState, ['rpi4'])

      expect(result.shouldSkip).toBe(true)
      expect(result.isValid).toBe(true)
      expect(result.availableOptions).toBe(1)
      expect(formState.sbcType).toBe('rpi4')
    })

    test('does not auto-skip when multiple SBCs available', () => {
      const formState: FormState = {}
      const result = validateSBCType(formState, ['rpi4', 'rpi5'])

      expect(result.shouldSkip).toBe(false)
      expect(result.availableOptions).toBe(2)
    })
  })

  describe('validateMachineArch', () => {
    test('returns invalid when machineArch is missing', () => {
      const formState: FormState = {}
      const result = validateMachineArch(formState)

      expect(result.isValid).toBe(false)
      expect(result.errorMessage).toBe('Machine architecture is required')
    })

    test('returns valid when machineArch is set', () => {
      const formState: FormState = { machineArch: 0 } // AMD64
      const result = validateMachineArch(formState)

      expect(result.isValid).toBe(true)
      expect(result.shouldSkip).toBe(false)
    })

    test('auto-skips when only one architecture available', () => {
      const formState: FormState = {}
      const result = validateMachineArch(formState, [1]) // ARM64 only

      expect(result.shouldSkip).toBe(true)
      expect(result.isValid).toBe(true)
      expect(result.availableOptions).toBe(1)
      expect(formState.machineArch).toBe(1)
    })

    test('does not auto-skip when multiple architectures available', () => {
      const formState: FormState = {}
      const result = validateMachineArch(formState, [0, 1]) // AMD64 and ARM64

      expect(result.shouldSkip).toBe(false)
      expect(result.availableOptions).toBe(2)
    })
  })

  describe('validateSystemExtensions', () => {
    test('always returns valid as extensions are optional', () => {
      const formState: FormState = {}
      const result = validateSystemExtensions(formState)

      expect(result.isValid).toBe(true)
      expect(result.shouldSkip).toBe(false)
    })

    test('returns valid with extensions', () => {
      const formState: FormState = { systemExtensions: ['ext1', 'ext2'] }
      const result = validateSystemExtensions(formState)

      expect(result.isValid).toBe(true)
    })
  })

  describe('validateExtraArgs', () => {
    test('always returns valid as extra args are optional', () => {
      const formState: FormState = {}
      const result = validateExtraArgs(formState)

      expect(result.isValid).toBe(true)
      expect(result.shouldSkip).toBe(false)
    })

    test('returns valid with cmdline', () => {
      const formState: FormState = { cmdline: 'console=tty0' }
      const result = validateExtraArgs(formState)

      expect(result.isValid).toBe(true)
    })
  })

  describe('validateConfirmation', () => {
    test('always returns valid', () => {
      const formState: FormState = {}
      const result = validateConfirmation(formState)

      expect(result.isValid).toBe(true)
      expect(result.shouldSkip).toBe(false)
    })
  })

  describe('getStepValidation', () => {
    test('returns validation result for known step', () => {
      const formState: FormState = { hardwareType: 'metal' }
      const result = getStepValidation('InstallationMediaCreateEntry', formState)

      expect(result.isValid).toBe(true)
    })

    test('returns invalid for unknown step', () => {
      const formState: FormState = {}
      const result = getStepValidation('UnknownStep', formState)

      expect(result.isValid).toBe(false)
      expect(result.errorMessage).toBe('Unknown step')
    })
  })

  describe('areIntermediateStepsValid', () => {
    const metalFlow = [
      'InstallationMediaCreateEntry',
      'InstallationMediaCreateTalosVersion',
      'InstallationMediaCreateMachineArch',
      'InstallationMediaCreateSystemExtensions',
      'InstallationMediaCreateExtraArgs',
      'InstallationMediaCreateConfirmation',
    ]

    test('returns true when all intermediate steps are valid', () => {
      const formState: FormState = {
        hardwareType: 'metal',
        talosVersion: '1.8.0',
        joinToken: 'token-123',
        machineArch: 0,
      }

      // Check if we can navigate to step index 4 (SystemExtensions)
      const result = areIntermediateStepsValid(metalFlow, 3, formState)

      expect(result).toBe(true)
    })

    test('returns false when an intermediate step is invalid', () => {
      const formState: FormState = {
        hardwareType: 'metal',
        // talosVersion is missing, so TalosVersion step is invalid
        joinToken: 'token-123',
      }

      // Try to navigate to step index 2 (MachineArch)
      const result = areIntermediateStepsValid(metalFlow, 2, formState)

      expect(result).toBe(false)
    })

    test('returns true when target step is 0 (first step)', () => {
      const formState: FormState = {}
      const result = areIntermediateStepsValid(metalFlow, 0, formState)

      expect(result).toBe(true)
    })
  })

  describe('findFirstInvalidStep', () => {
    const metalFlow = [
      'InstallationMediaCreateEntry',
      'InstallationMediaCreateTalosVersion',
      'InstallationMediaCreateMachineArch',
      'InstallationMediaCreateSystemExtensions',
      'InstallationMediaCreateExtraArgs',
      'InstallationMediaCreateConfirmation',
    ]

    test('returns index of first invalid step', () => {
      const formState: FormState = {
        hardwareType: 'metal',
        // talosVersion is missing
      }

      const result = findFirstInvalidStep(metalFlow, formState)

      expect(result).toBe(1) // TalosVersion step is invalid
    })

    test('returns -1 when all steps are valid', () => {
      const formState: FormState = {
        hardwareType: 'metal',
        talosVersion: '1.8.0',
        joinToken: 'token-123',
        machineArch: 0,
      }

      const result = findFirstInvalidStep(metalFlow, formState)

      expect(result).toBe(-1)
    })

    test('returns 0 when first step is invalid', () => {
      const formState: FormState = {
        // hardwareType is missing
      }

      const result = findFirstInvalidStep(metalFlow, formState)

      expect(result).toBe(0) // Entry step is invalid
    })

    test('finds first invalid in cloud flow', () => {
      const cloudFlow = [
        'InstallationMediaCreateEntry',
        'InstallationMediaCreateTalosVersion',
        'InstallationMediaCreateCloudProvider',
        'InstallationMediaCreateMachineArch',
        'InstallationMediaCreateSystemExtensions',
        'InstallationMediaCreateExtraArgs',
        'InstallationMediaCreateConfirmation',
      ]

      const formState: FormState = {
        hardwareType: 'cloud',
        talosVersion: '1.8.0',
        joinToken: 'token-123',
        // cloudPlatform is missing
      }

      const result = findFirstInvalidStep(cloudFlow, formState)

      expect(result).toBe(2) // CloudProvider step is invalid
    })
  })
})
