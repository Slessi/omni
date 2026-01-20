# Installation Media Wizard Validation Framework

This directory contains the implementation of the validation framework for the Installation Media Wizard, which addresses [issue #2004](https://github.com/siderolabs/omni/issues/2004).

## Overview

The framework provides a flexible and extensible way to manage wizard steps with:
- **Step validation**: Each step can validate its own state
- **Default values**: Steps can apply defaults when entered
- **Auto-skip logic**: Steps with no options or single options can be automatically skipped
- **Forward navigation**: Users can jump to future steps if all intermediate steps are valid
- **Dependency management**: Steps can reset when their dependencies change

## Key Files

### `useWizardSteps.ts`
The core composable that manages wizard navigation and validation.

**Exports:**
- `StepConfig` interface: Defines the configuration for each step
- `useWizardSteps()` composable: Provides navigation and validation logic

**Key Features:**
- Validates current step
- Checks if steps are accessible
- Handles navigation with skip logic
- Manages default values
- Tracks dependency changes

### `stepConfigurations.ts`
Defines the validation and behavior for each step in all three flows:
- **Metal flow**: Bare-metal machine installation
- **Cloud flow**: Cloud platform installation
- **SBC flow**: Single-board computer installation

**Key Features:**
- Step validation rules
- Skip conditions
- Default value application
- Dependency change handlers

### `stepHelpers.ts`
Helper functions for common validation and configuration logic:
- `getAvailableArchitectures()`: Get architectures for current config
- `shouldSkipArchitectureStep()`: Determine if architecture step should be skipped
- `applyDefaultArchitecture()`: Apply default architecture selection
- `resetArchitectureIfInvalid()`: Reset architecture when no longer valid
- `isCloudPlatformCompatible()`: Check platform compatibility
- `resetCloudPlatformIfIncompatible()`: Reset platform when incompatible

### `InstallationMediaCreate.vue`
The main wizard component, updated to use the validation framework.

**Key Changes:**
- Uses `useWizardSteps` composable
- Disables "Next" button when current step is invalid
- Allows clicking on future steps in the stepper if they're accessible
- Handles state reset on dependency changes

### `Stepper.vue`
Enhanced to support:
- Accessibility checking for each step
- Click handlers with validation
- Visual feedback for accessible/inaccessible steps

## Usage

### Adding a New Step

1. Create the step component in `Steps/`
2. Add validation configuration in `stepConfigurations.ts`:

```typescript
{
  component: YourNewStep,
  isValid: (formState: FormState) => {
    // Return true if step is valid
    return !!formState.someRequiredField
  },
  shouldSkip: (formState: FormState) => {
    // Return true to skip this step
    return formState.someCondition === true
  },
  applyDefaults: (formState: FormState) => {
    // Set default values
    formState.someField ??= 'default-value'
  },
  onDependencyChange: (formState: FormState) => {
    // Reset state when dependencies change
    if (/* some condition */) {
      formState.someField = undefined
    }
  },
}
```

### Defining Validation

Each step can define an `isValid` function:

```typescript
isValid: (formState: FormState) => {
  // Required field validation
  if (!formState.requiredField) {
    return false
  }
  
  // Complex validation
  if (formState.someField && !isValidFormat(formState.someField)) {
    return false
  }
  
  return true
}
```

### Auto-Skip Logic

Steps can be automatically skipped when they have:
- No available options
- Only one option (auto-selected)

```typescript
shouldSkip: (formState: FormState) => {
  const options = getOptionsForStep(formState)
  return options.length <= 1
}
```

### Default Values

Apply defaults when entering a step:

```typescript
applyDefaults: (formState: FormState) => {
  // Set default if not already set
  formState.field ??= getDefaultValue()
  
  // Auto-select if only one option
  const options = getAvailableOptions(formState)
  if (options.length === 1) {
    formState.field = options[0]
  }
}
```

### Dependency Management

Reset state when dependencies change:

```typescript
onDependencyChange: (formState: FormState) => {
  // Reset if dependent field changed
  const isStillValid = checkValidity(formState)
  if (!isStillValid) {
    formState.dependentField = undefined
  }
}
```

## Testing

Tests are located in `useWizardSteps.spec.ts` and cover:
- Step validation
- Skip logic
- Navigation
- Default application
- Accessibility checking

Run tests with:
```bash
npm test
```

## Future Enhancements

Potential improvements to the framework:
1. **Async validation**: Support for asynchronous validation (e.g., API calls)
2. **Progress persistence**: Save validation state across page refreshes
3. **Validation messages**: Display specific error messages for invalid steps
4. **Batch validation**: Validate multiple steps at once
5. **Conditional flows**: Dynamic flow based on selections (beyond current skip logic)
6. **Step prefetching**: Load data for upcoming steps in advance
7. **Undo/redo**: Support for reverting changes
8. **A/B testing**: Framework for testing different validation strategies

## Architecture Decisions

### Why a Composable?

The validation logic is encapsulated in a composable (`useWizardSteps`) for:
- **Reusability**: Can be used in other wizards
- **Testability**: Easy to unit test in isolation
- **Separation of concerns**: Logic separate from UI
- **Type safety**: Full TypeScript support

### Why Separate Configuration Files?

Step configurations are in separate files for:
- **Maintainability**: Easy to find and update step logic
- **Readability**: Clear separation of each step's behavior
- **Modularity**: Steps can be added/removed easily
- **Testing**: Configurations can be tested independently

### Why Helper Functions?

Helper functions in `stepHelpers.ts` provide:
- **Code reuse**: Common logic used across multiple steps
- **Centralized logic**: Single source of truth for validation rules
- **Easy mocking**: Simple to mock in tests
- **Future API integration**: Easy to replace with real API calls

## Migration Notes

The framework is backward compatible with existing wizard behavior:
- Steps without validation are always considered valid
- Steps without skip logic are never skipped
- Steps without defaults work as before
- Stepper works without accessibility checking

No changes to existing step components are required unless you want to add validation.
