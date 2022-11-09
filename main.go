package main

import (
	"github.com/carlmontanari/vcluster-plugin-prefer-parent-resources/hooks"
	"github.com/loft-sh/vcluster-sdk/plugin"
)

func main() {
	ctx := plugin.MustInit("prefer-parent-resources-hooks")
	plugin.MustRegister(hooks.NewPreferParentConfigmapsHook(ctx))
	plugin.MustRegister(hooks.NewPreferParentSecretsHook(ctx))
	plugin.MustStart()
}
