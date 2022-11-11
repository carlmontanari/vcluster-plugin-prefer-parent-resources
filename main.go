package main

import (
	"github.com/carlmontanari/vcluster-plugin-prefer-parent-resources/prefer-parent-resources/hooks"
	vclustersdkplugin "github.com/loft-sh/vcluster-sdk/plugin"
)

func main() {
	ctx := vclustersdkplugin.MustInit()

	for _, hook := range hooks.GetAllHooks(ctx) {
		vclustersdkplugin.MustRegister(hook)
	}

	vclustersdkplugin.MustStart()
}
