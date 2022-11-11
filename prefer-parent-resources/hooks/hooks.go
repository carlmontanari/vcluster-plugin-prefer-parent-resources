package hooks

import (
	"github.com/loft-sh/vcluster-sdk/syncer"
	"github.com/loft-sh/vcluster-sdk/syncer/context"
)

// GetAllHooks returns all hook objects to register.
func GetAllHooks(ctx *context.RegisterContext) []syncer.Base {
	return []syncer.Base{
		NewPreferParentConfigmapsHook(ctx),
		NewPreferParentSecretsHook(ctx),
	}
}
