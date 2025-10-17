package main

import (
	"github.com/takaishi/tflint-ruleset-takaishi/rules"
	"github.com/terraform-linters/tflint-plugin-sdk/plugin"
	"github.com/terraform-linters/tflint-plugin-sdk/tflint"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		RuleSet: &tflint.BuiltinRuleSet{
			Name:    "takaishi",
			Version: "0.0.1",
			Rules: []tflint.Rule{
				rules.NewModuleCircularDependencyRule(),
			},
		},
	})
}
