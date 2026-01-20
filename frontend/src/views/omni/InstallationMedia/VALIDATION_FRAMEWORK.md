# Installation Media Wizard Validation & Defaults Framework

This document describes the validation and defaults framework for the Installation Media wizard.

## Overview

The framework provides:
1. **Step Validation**: Each step can define required fields and validation logic
2. **Smart Navigation**: Users can only navigate to steps when all intermediate steps are valid
3. **Auto-Skip**: Steps with only one option are automatically skipped
4. **State Management**: Dependent steps are automatically reset when their dependencies change
5. **Default Values**: Steps can define default values to be applied

## Architecture

### Core Components

#### `stepConfig.ts`
Defines the validation framework and step configurations.

**Key Types:**
- `StepValidationResult`: Result of validating a step
  - `isValid`: Whether all required fields are filled
  - `shouldSkip`: Whether the step should be auto-skipped
  - `errorMessage`: Optional validation error message
  - `availableOptions`: Number of available options (for auto-skip logic)

- `StepConfig`: Configuration for a wizard step
  - `name`: Unique route name for the step
  - `validate`: Function to validate the step
  - `applyDefaults`: Optional function to apply default values
  - `dependencies`: Fields that this step depends on (for reset logic)

**Key Functions:**
- `getStepValidation()`: Get validation result for a step
- `areIntermediateStepsValid()`: Check if all steps up to a target are valid
- `findFirstInvalidStep()`: Find the first invalid step in a flow
- `resetDependentSteps()`: Reset steps that depend on changed fields

#### `useWizardStep.ts`
Composable for step components to integrate with the validation framework.

**Usage in step components:**
```typescript
import { useWizardStep } from './useWizardStep'

const { validation, checkAutoSkip, shouldAutoSkip } = useWizardStep(
  'InstallationMediaCreateMachineArch',
  formState,
  availableArchitectures
)
```

#### `InstallationMediaCreate.vue`
Main wizard component that orchestrates validation and navigation.

**Key Features:**
- Validates current step before allowing navigation
- Disables "Next" button when current step is invalid
- Prevents jumping to invalid steps via stepper
- Watches for dependency changes and resets affected steps
- Redirects to first invalid step when dependencies change

## Step Validators

### Required Steps
These steps require user input before proceeding:

1. **Entry** (`validateEntry`)
   - Requires: `hardwareType`
   - Dependencies: None

2. **TalosVersion** (`validateTalosVersion`)
   - Requires: `talosVersion`, `joinToken`
   - Dependencies: `hardwareType`

3. **CloudProvider** (`validateCloudProvider`)
   - Requires: `cloudPlatform`
   - Dependencies: `talosVersion`
   - Auto-skip: When only one provider is compatible with selected Talos version

4. **SBCType** (`validateSBCType`)
   - Requires: `sbcType`
   - Dependencies: `talosVersion`
   - Auto-skip: When only one SBC type is compatible with selected Talos version

5. **MachineArch** (`validateMachineArch`)
   - Requires: `machineArch`
   - Dependencies: `hardwareType`, `cloudPlatform`, `sbcType`
   - Auto-skip: When platform supports only one architecture

### Optional Steps
These steps are always valid (optional selections):

6. **SystemExtensions** (`validateSystemExtensions`)
   - Optional: `systemExtensions`
   - Dependencies: `talosVersion`

7. **ExtraArgs** (`validateExtraArgs`)
   - Optional: `cmdline`, `overlayOptions`, `bootloader`
   - Dependencies: None

8. **Confirmation** (`validateConfirmation`)
   - Always valid (summary step)

## Auto-Skip Logic

Steps can be automatically skipped when there's only one available option:

1. Step component determines available options based on current form state
2. Validator is called with available options array
3. If only one option exists:
   - `shouldSkip` is set to `true`
   - The option is auto-selected
   - Component can navigate to next step automatically

**Example:**
```typescript
// In MachineArch.vue
const supportedArchitectures = computed(() => {
  // Get architectures from platform config
  return platform.value?.spec.architectures ?? []
})

// Validation will auto-skip if only one architecture
const validation = validateMachineArch(formState.value, supportedArchitectures.value)

if (validation.shouldSkip) {
  // Navigate to next step
}
```

## State Management

### Dependency Tracking

When a field changes, all steps that depend on it are automatically reset:

```typescript
// Example: Changing hardwareType resets:
// - talosVersion (depends on hardwareType)
// - cloudPlatform (if cloud flow)
// - sbcType (if sbc flow)
// - machineArch (depends on hardwareType/platform)
```

### Reset Behavior

The framework tracks these dependency chains:

- `hardwareType` → `talosVersion`, `joinToken`
- `talosVersion` → `cloudPlatform`, `sbcType`, `systemExtensions`
- `cloudPlatform` → `machineArch`, `secureBoot`
- `sbcType` → `machineArch`

When a field changes, any fields that depend on it are cleared to ensure consistency.

## Navigation Rules

### Forward Navigation
- User can click "Next" only if current step is valid
- User can jump forward via stepper only if all intermediate steps are valid
- Invalid jumps redirect to first invalid step

### Backward Navigation
- Always allowed (user can go back to fix earlier choices)
- Does not trigger validation or reset logic

### Entry Page
- Visiting entry page always resets the entire form
- This ensures clean state for new wizard sessions

## Testing

The framework includes comprehensive tests:

1. **Unit Tests** (`stepConfig.spec.ts`)
   - Tests for each validator
   - Tests for auto-skip logic
   - Tests for navigation helpers

2. **Integration Tests** (`InstallationMediaCreate.validation.spec.ts`)
   - Tests for validation integration with wizard component
   - Tests for navigation prevention
   - Tests for state reset behavior

3. **Component Tests** (`InstallationMediaCreate.spec.ts`)
   - Tests for existing wizard functionality
   - Tests for session storage persistence
   - Tests for stepper behavior

## Adding New Steps

To add a new step with validation:

1. **Define Validator** in `stepConfig.ts`:
```typescript
export function validateMyStep(
  formState: FormState,
  availableOptions?: string[]
): StepValidationResult {
  const isValid = !!formState.myField
  const shouldSkip = availableOptions?.length === 1
  
  if (shouldSkip && availableOptions) {
    formState.myField = availableOptions[0]
  }
  
  return {
    isValid: isValid || shouldSkip,
    shouldSkip,
    errorMessage: isValid || shouldSkip ? undefined : 'My field is required',
    availableOptions: availableOptions?.length,
  }
}
```

2. **Add to Step Configs**:
```typescript
export const stepConfigs: Record<string, StepConfig> = {
  // ...
  InstallationMediaCreateMyStep: {
    name: 'InstallationMediaCreateMyStep',
    validate: validateMyStep,
    dependencies: ['previousField'],
  },
}
```

3. **Update Reset Logic** if needed:
```typescript
function resetStepFields(stepName: string, formState: FormState): void {
  switch (stepName) {
    // ...
    case 'InstallationMediaCreateMyStep':
      delete formState.myField
      break
  }
}
```

4. **Add to Flow**:
```typescript
const flows: Record<HardwareType, string[]> = {
  metal: [
    // ...
    'InstallationMediaCreateMyStep',
    // ...
  ],
}
```

5. **Use in Component** (optional for auto-skip):
```typescript
import { useWizardStep } from './useWizardStep'

const { validation, checkAutoSkip } = useWizardStep(
  'InstallationMediaCreateMyStep',
  formState,
  availableOptions
)

watch(availableOptions, () => {
  checkAutoSkip(nextStepName)
})
```

## Future Enhancements

Potential improvements to consider:

1. **Validation State Persistence**: Save validation state to session storage
2. **Confirmation Dialogs**: Show confirmation when resetting dependent steps
3. **Progressive Validation**: Validate as user types, not just on navigation
4. **Custom Error Messages**: Allow steps to provide more specific error messages
5. **Async Validation**: Support validators that need to fetch data
6. **Step Metadata**: Additional step metadata (title, description, help text)
