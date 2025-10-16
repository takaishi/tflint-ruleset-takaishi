package rules

import (
	"testing"

	"github.com/terraform-linters/tflint-plugin-sdk/helper"
)

func TestModuleCircularDependencyRule(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected helper.Issues
	}{
		{
			name: "no circular dependency",
			content: `
module "module_a" {
  source = "./modules/a"
  input = "value"
}

module "module_b" {
  source = "./modules/b"
  input = module.module_a.output
}

module "module_c" {
  source = "./modules/c"
  input = module.module_b.output
}`,
			expected: helper.Issues{},
		},
		{
			name: "circular dependency between two modules",
			content: `
module "module_a" {
  source = "./modules/a"
  input = module.module_b.output
}

module "module_b" {
  source = "./modules/b"
  input = module.module_a.output
}`,
			expected: helper.Issues{
				{
					Rule:    NewModuleCircularDependencyRule(),
					Message: "Circular dependency detected between modules: module_a ↔ module_b",
				},
			},
		},
		{
			name: "circular dependency in complex expression",
			content: `
module "module_a" {
  source = "./modules/a"
  input = "${module.module_b.output}-suffix"
}

module "module_b" {
  source = "./modules/b"
  input = module.module_a.output
}`,
			expected: helper.Issues{
				{
					Rule:    NewModuleCircularDependencyRule(),
					Message: "Circular dependency detected between modules: module_a ↔ module_b",
				},
			},
		},
		{
			name: "circular dependency in object",
			content: `
module "module_a" {
  source = "./modules/a"
  config = {
    value = module.module_b.output
  }
}

module "module_b" {
  source = "./modules/b"
  config = {
    value = module.module_a.output
  }
}`,
			expected: helper.Issues{
				{
					Rule:    NewModuleCircularDependencyRule(),
					Message: "Circular dependency detected between modules: module_a ↔ module_b",
				},
			},
		},
		{
			name: "circular dependency in list",
			content: `
module "module_a" {
  source = "./modules/a"
  values = [module.module_b.output]
}

module "module_b" {
  source = "./modules/b"
  values = [module.module_a.output]
}`,
			expected: helper.Issues{
				{
					Rule:    NewModuleCircularDependencyRule(),
					Message: "Circular dependency detected between modules: module_a ↔ module_b",
				},
			},
		},
		{
			name: "no circular dependency with multiple modules",
			content: `
module "module_a" {
  source = "./modules/a"
  input = "value"
}

module "module_b" {
  source = "./modules/b"
  input = module.module_a.output
}

module "module_c" {
  source = "./modules/c"
  input = module.module_a.output
}

module "module_d" {
  source = "./modules/d"
  input = module.module_b.output
}`,
			expected: helper.Issues{},
		},
		{
			name: "circular dependency with multiple references",
			content: `
module "module_a" {
  source = "./modules/a"
  input1 = module.module_b.output
  input2 = module.module_b.output
}

module "module_b" {
  source = "./modules/b"
  input = module.module_a.output
}`,
			expected: helper.Issues{
				{
					Rule:    NewModuleCircularDependencyRule(),
					Message: "Circular dependency detected between modules: module_a ↔ module_b",
				},
			},
		},
		{
			name: "complex circular dependency with three modules",
			content: `
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
}`,
			expected: helper.Issues{
				{
					Rule:    NewModuleCircularDependencyRule(),
					Message: "Circular dependency detected between modules: module_a ↔ module_b (path: module_a → module_b → module_c → module_a)",
				},
				{
					Rule:    NewModuleCircularDependencyRule(),
					Message: "Circular dependency detected between modules: module_b ↔ module_c (path: module_a → module_b → module_c → module_a)",
				},
				{
					Rule:    NewModuleCircularDependencyRule(),
					Message: "Circular dependency detected between modules: module_c ↔ module_a (path: module_a → module_b → module_c → module_a)",
				},
			},
		},
	}

	rule := NewModuleCircularDependencyRule()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := helper.TestRunner(t, map[string]string{"main.tf": test.content})
			if err := rule.Check(runner); err != nil {
				t.Fatalf("Unexpected error occurred: %s", err)
			}

			// Check expected error count
			if len(runner.Issues) != len(test.expected) {
				t.Errorf("Expected %d issues, got %d", len(test.expected), len(runner.Issues))
				for _, issue := range runner.Issues {
					t.Logf("Issue: %s", issue.Message)
				}
				return
			}

			// Check if circular dependency error message is included
			for i, expectedIssue := range test.expected {
				if i >= len(runner.Issues) {
					break
				}
				actualIssue := runner.Issues[i]

				// Check if message contains "Circular dependency detected between modules"
				if expectedIssue.Message != "" {
					// Allow either module_a ↔ module_b or module_b ↔ module_a
					containsCircular := false
					if actualIssue.Message == expectedIssue.Message {
						containsCircular = true
					} else {
						// Allow reverse order message
						reverseMsg := "Circular dependency detected between modules: module_b ↔ module_a"
						if expectedIssue.Message == "Circular dependency detected between modules: module_a ↔ module_b" && actualIssue.Message == reverseMsg {
							containsCircular = true
						}
					}

					if !containsCircular {
						t.Errorf("Expected message '%s', got '%s'", expectedIssue.Message, actualIssue.Message)
					}
				}
			}
		})
	}
}
