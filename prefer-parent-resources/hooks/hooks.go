package hooks

import (
	vclustersdksyncer "github.com/loft-sh/vcluster-sdk/syncer"
	vclustersdksyncercontext "github.com/loft-sh/vcluster-sdk/syncer/context"
)

// GetAllHooks returns all hook objects to register.
func GetAllHooks(ctx *vclustersdksyncercontext.RegisterContext) []vclustersdksyncer.Base {
	return []vclustersdksyncer.Base{
		NewPreferParentConfigmapsHook(ctx),
		NewPreferParentSecretsHook(ctx),
	}
}
