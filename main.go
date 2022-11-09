package main

import (
	"github.com/carlmontanari/vcluster-plugin-prefer-parent-resources/hooks"
	"github.com/loft-sh/vcluster-sdk/plugin"
)

func main() {
	ctx := plugin.MustInit()

	for _, hook := range hooks.GetAllHooks(ctx) {
		plugin.MustRegister(hook)
	}

	plugin.MustStart()
}
