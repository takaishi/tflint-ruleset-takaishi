package rules

import (
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

// ModuleCircularDependencyRule prevents circular dependencies between modules
type ModuleCircularDependencyRule struct {
	tflint.DefaultRule
}

// NewModuleCircularDependencyRule creates a new rule instance
func NewModuleCircularDependencyRule() *ModuleCircularDependencyRule {
	return &ModuleCircularDependencyRule{}
}

// Name returns the rule name
func (r *ModuleCircularDependencyRule) Name() string {
	return "module_circular_dependency"
}

// Enabled returns whether the rule is enabled
func (r *ModuleCircularDependencyRule) Enabled() bool {
	return false
}

// Severity returns the rule severity
func (r *ModuleCircularDependencyRule) Severity() tflint.Severity {
	return tflint.ERROR
}

// Link returns a link to detailed information about the rule
func (r *ModuleCircularDependencyRule) Link() string {
	return "https://github.com/takaishi/tflint-ruleset-takaishi"
}

// Check executes the rule checking process
func (r *ModuleCircularDependencyRule) Check(runner tflint.Runner) error {
	// Collect module definitions
	modules, err := r.collectModules(runner)
	if err != nil {
		return err
	}

	// Build dependency relationships between modules
	dependencies, err := r.buildDependencies(runner, modules)
	if err != nil {
		return err
	}

	// Detect circular dependencies
	circularDeps := r.detectCircularDependencies(dependencies)

	// Report errors
	for _, dep := range circularDeps {
		var message string
		if dep.CyclePath != "" {
			// For indirect circular dependencies, show the entire cycle path
			message = fmt.Sprintf("Circular dependency detected between modules: %s ↔ %s (path: %s)", dep.ModuleA, dep.ModuleB, dep.CyclePath)
		} else {
			// For direct circular dependencies
			message = fmt.Sprintf("Circular dependency detected between modules: %s ↔ %s", dep.ModuleA, dep.ModuleB)
		}

		err := runner.EmitIssue(
			r,
			message,
			dep.Range,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// ModuleInfo holds module information
type ModuleInfo struct {
	Name string
}

// Dependency represents a dependency relationship between modules
type Dependency struct {
	From  string
	To    string
	Range hcl.Range
}

// CircularDependency represents a circular dependency
type CircularDependency struct {
	ModuleA   string
	ModuleB   string
	Range     hcl.Range
	CyclePath string // Path of the entire cycle (for indirect circular dependencies)
}

// collectModules collects all module definitions
func (r *ModuleCircularDependencyRule) collectModules(runner tflint.Runner) (map[string]ModuleInfo, error) {
	modules := make(map[string]ModuleInfo)

	files, err := runner.GetFiles()
	if err != nil {
		return nil, err
	}

	// Sort by filename for deterministic order
	var fileNames []string
	for fileName := range files {
		fileNames = append(fileNames, fileName)
	}
	sort.Strings(fileNames)

	for _, fileName := range fileNames {
		file := files[fileName]
		body, ok := file.Body.(*hclsyntax.Body)
		if !ok {
			continue
		}

		for _, block := range body.Blocks {
			if block.Type == "module" && len(block.Labels) > 0 {
				moduleName := block.Labels[0]

				modules[moduleName] = ModuleInfo{
					Name: moduleName,
				}
			}
		}
	}

	return modules, nil
}

// buildDependencies builds dependency relationships between modules
func (r *ModuleCircularDependencyRule) buildDependencies(runner tflint.Runner, modules map[string]ModuleInfo) ([]Dependency, error) {
	var dependencies []Dependency
	seenDeps := make(map[string]bool) // Map to prevent duplicates

	files, err := runner.GetFiles()
	if err != nil {
		return nil, err
	}

	// Sort by filename for deterministic order
	var fileNames []string
	for fileName := range files {
		fileNames = append(fileNames, fileName)
	}
	sort.Strings(fileNames)

	for _, fileName := range fileNames {
		file := files[fileName]
		body, ok := file.Body.(*hclsyntax.Body)
		if !ok {
			continue
		}

		// Sort blocks for deterministic order
		var blocks []*hclsyntax.Block
		for _, block := range body.Blocks {
			blocks = append(blocks, block)
		}

		// Sort blocks by position (by line number)
		sort.Slice(blocks, func(i, j int) bool {
			return blocks[i].Range().Start.Line < blocks[j].Range().Start.Line
		})

		for _, block := range blocks {
			if block.Type == "module" && len(block.Labels) > 0 {
				moduleName := block.Labels[0]

				// Sort attributes for deterministic order
				var attrs []*hclsyntax.Attribute
				for _, attr := range block.Body.Attributes {
					attrs = append(attrs, attr)
				}

				// Sort attributes by position (by line number)
				sort.Slice(attrs, func(i, j int) bool {
					return attrs[i].Range().Start.Line < attrs[j].Range().Start.Line
				})

				for _, attr := range attrs {
					deps := r.findModuleReferences(attr.Expr, modules)
					for _, dep := range deps {
						// Create key for duplicate checking
						depKey := moduleName + "->" + dep
						if !seenDeps[depKey] {
							seenDeps[depKey] = true
							dependencies = append(dependencies, Dependency{
								From:  moduleName,
								To:    dep,
								Range: attr.Range(),
							})
						}
					}
				}
			}
		}
	}

	return dependencies, nil
}

// findModuleReferences searches for module references in expressions
func (r *ModuleCircularDependencyRule) findModuleReferences(expr hcl.Expression, modules map[string]ModuleInfo) []string {
	var references []string

	switch e := expr.(type) {
	case *hclsyntax.ScopeTraversalExpr:
		// Check format: module.module_name.output_name
		if len(e.Traversal) >= 2 {
			if root, ok := e.Traversal[0].(hcl.TraverseRoot); ok {
				if root.Name == "module" && len(e.Traversal) >= 2 {
					if attr, ok := e.Traversal[1].(hcl.TraverseAttr); ok {
						if _, exists := modules[attr.Name]; exists {
							references = append(references, attr.Name)
						}
					}
				}
			}
		}

	case *hclsyntax.TemplateExpr:
		// Check references in template expressions
		for _, part := range e.Parts {
			refs := r.findModuleReferences(part, modules)
			references = append(references, refs...)
		}

	case *hclsyntax.TupleConsExpr:
		// Check references in tuple expressions
		for _, expr := range e.Exprs {
			refs := r.findModuleReferences(expr, modules)
			references = append(references, refs...)
		}

	case *hclsyntax.ObjectConsExpr:
		// Check references in object expressions
		for _, item := range e.Items {
			if item.ValueExpr != nil {
				refs := r.findModuleReferences(item.ValueExpr, modules)
				references = append(references, refs...)
			}
		}

	case *hclsyntax.FunctionCallExpr:
		// Check references in function calls
		for _, arg := range e.Args {
			refs := r.findModuleReferences(arg, modules)
			references = append(references, refs...)
		}

	case *hclsyntax.ConditionalExpr:
		// Check references in conditional expressions
		if e.TrueResult != nil {
			refs := r.findModuleReferences(e.TrueResult, modules)
			references = append(references, refs...)
		}
		if e.FalseResult != nil {
			refs := r.findModuleReferences(e.FalseResult, modules)
			references = append(references, refs...)
		}

	case *hclsyntax.ForExpr:
		// Check references in for expressions
		if e.CollExpr != nil {
			refs := r.findModuleReferences(e.CollExpr, modules)
			references = append(references, refs...)
		}
		if e.KeyExpr != nil {
			refs := r.findModuleReferences(e.KeyExpr, modules)
			references = append(references, refs...)
		}
		if e.ValExpr != nil {
			refs := r.findModuleReferences(e.ValExpr, modules)
			references = append(references, refs...)
		}
		if e.CondExpr != nil {
			refs := r.findModuleReferences(e.CondExpr, modules)
			references = append(references, refs...)
		}
	}

	return references
}

// detectCircularDependencies detects circular dependencies
func (r *ModuleCircularDependencyRule) detectCircularDependencies(dependencies []Dependency) []CircularDependency {
	var circularDeps []CircularDependency
	reportedCycles := make(map[string]bool) // Track reported cycles

	// Build dependency map
	depMap := make(map[string][]string)
	depRangeMap := make(map[string]map[string]hcl.Range)
	for _, dep := range dependencies {
		depMap[dep.From] = append(depMap[dep.From], dep.To)
		if depRangeMap[dep.From] == nil {
			depRangeMap[dep.From] = make(map[string]hcl.Range)
		}
		depRangeMap[dep.From][dep.To] = dep.Range
	}

	// Sort module names for deterministic order
	var modules []string
	for module := range depMap {
		modules = append(modules, module)
	}
	sort.Strings(modules)

	// Sort dependencies for deterministic order
	for from := range depMap {
		sort.Strings(depMap[from])
	}

	// First detect direct circular dependencies (A → B → A)
	for _, module := range modules {
		if deps, exists := depMap[module]; exists {
			for _, dep := range deps {
				// Check reverse dependency
				if reverseDeps, exists := depMap[dep]; exists {
					for _, reverseDep := range reverseDeps {
						if reverseDep == module {
							// Found direct circular dependency
							cycleKey := r.normalizeCycle([]string{module, dep})

							// Check if cycle already reported
							if reportedCycles[cycleKey] {
								continue
							}

							reportedCycles[cycleKey] = true

							rangeToUse := hcl.Range{}
							if depRangeMap[module] != nil && depRangeMap[module][dep].Filename != "" {
								rangeToUse = depRangeMap[module][dep]
							}

							circularDeps = append(circularDeps, CircularDependency{
								ModuleA: module,
								ModuleB: dep,
								Range:   rangeToUse,
							})
						}
					}
				}
			}
		}
	}

	// Next detect indirect circular dependencies (3 or more modules)
	for _, module := range modules {
		visited := make(map[string]bool)
		recStack := make(map[string]bool)
		path := []string{}

		// Detect circular dependency (only the first one found)
		if cycle := r.findCycle(module, depMap, visited, recStack, &path); cycle != nil {
			// Create unique key for cycle (normalize order)
			cycleKey := r.normalizeCycle(cycle)

			// Check if cycle already reported
			if reportedCycles[cycleKey] {
				continue
			}

			reportedCycles[cycleKey] = true

			// If circular dependency found, report the entire cycle path
			for i := 0; i < len(cycle); i++ {
				moduleA := cycle[i]
				moduleB := cycle[(i+1)%len(cycle)] // Next module (return to first if last)

				// Rangeを取得
				rangeToUse := hcl.Range{}
				if depRangeMap[moduleA] != nil && depRangeMap[moduleA][moduleB].Filename != "" {
					rangeToUse = depRangeMap[moduleA][moduleB]
				}

				// Include entire cycle path in message
				cyclePath := ""
				for j, mod := range cycle {
					if j > 0 {
						cyclePath += " → "
					}
					cyclePath += mod
				}
				cyclePath += " → " + cycle[0] // Return to first module

				circularDeps = append(circularDeps, CircularDependency{
					ModuleA:   moduleA,
					ModuleB:   moduleB,
					Range:     rangeToUse,
					CyclePath: cyclePath, // Add entire cycle path
				})
			}
		}
	}

	return circularDeps
}

// normalizeCycle normalizes a cycle to create a unique key
func (r *ModuleCircularDependencyRule) normalizeCycle(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}

	// Rotate to start with the smallest module name
	minIndex := 0
	for i, module := range cycle {
		if module < cycle[minIndex] {
			minIndex = i
		}
	}

	// Rotate the cycle
	normalized := make([]string, len(cycle))
	for i := 0; i < len(cycle); i++ {
		normalized[i] = cycle[(minIndex+i)%len(cycle)]
	}

	// Join as string
	result := ""
	for _, module := range normalized {
		result += module + "→"
	}
	return result
}

// findCycle detects circular dependencies using depth-first search and returns the cycle
func (r *ModuleCircularDependencyRule) findCycle(module string, depMap map[string][]string, visited map[string]bool, recStack map[string]bool, path *[]string) []string {
	if recStack[module] {
		// Found circular dependency - find the start of the cycle
		cycleStart := -1
		for i, m := range *path {
			if m == module {
				cycleStart = i
				break
			}
		}
		if cycleStart >= 0 {
			return (*path)[cycleStart:]
		}
		return nil
	}

	if visited[module] {
		return nil
	}

	visited[module] = true
	recStack[module] = true
	*path = append(*path, module)

	for _, dep := range depMap[module] {
		if cycle := r.findCycle(dep, depMap, visited, recStack, path); cycle != nil {
			return cycle
		}
	}

	recStack[module] = false
	*path = (*path)[:len(*path)-1]
	return nil
}
