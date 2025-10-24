# My TFLint Ruleset

A custom TFLint ruleset plugin for Terraform that detects circular dependencies between modules.

## Overview

This ruleset automatically detects and reports circular dependencies between Terraform modules as errors. A circular dependency occurs when module A depends on module B, while module B also depends on module A.

## Installation

### 1. Configure TFLint

Add the following configuration to your `.tflint.hcl` file:

```hcl
plugin "takaishi" {
  enabled = true
  version = "0.0.1"
  source  = "github.com/takaishi/tflint-ruleset-takaishi"
}
```

### 2. init

Run init command:

```
tflint --init
```

## Rules

### module_circular_dependency

A rule that detects circular dependencies between modules.

#### Configuration

```hcl
rule "module_circular_dependency" {
  enabled = true
}
```

#### Detection Examples

**Direct circular dependency:**

```hcl
module "module_a" {
  source = "./modules/a"
  input = module.module_b.output
}

module "module_b" {
  source = "./modules/b"
  input = module.module_a.output
}
```

**Indirect circular dependency (3 or more modules):**

```hcl
module "module_a" {
  source = "./modules/a"
  input = module.module_b.output
}

module "module_b" {
  source = "./modules/b"
  input = module.module_c.output
}

module "module_c" {
  source = "./modules/c"
  input = module.module_a.output
}
```

#### Supported Expression Types

This rule can detect module references in the following Terraform expressions:

- Direct reference: `module.module_name.output`
- Template expression: `"${module.module_name.output}-suffix"`
- Object expression: `{ value = module.module_name.output }`
- List expression: `[module.module_name.output]`
- Function call: `concat(module.module_name.output, ...)`
- Conditional expression: `condition ? module.module_name.output : ...`
- For expression: `[for item in module.module_name.output : ...]`
