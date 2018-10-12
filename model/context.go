package model

import (
	"context"

	"github.com/AtlantPlatform/ethfw"
)

type AppContext struct {
	context.Context
}

func NewAppContext(ctx context.Context, specDir string, keycache ethfw.KeyCache) AppContext {
	ctx = context.WithValue(ctx, "prefix", "playbook")
	ctx = context.WithValue(ctx, "specdir", specDir)
	ctx = context.WithValue(ctx, "keycache", keycache)
	return AppContext{ctx}
}

func (ctx AppContext) SpecDir() string {
	return ctx.Value("specdir").(string)
}

func (ctx AppContext) KeyCache() ethfw.KeyCache {
	return ctx.Value("keycache").(ethfw.KeyCache)
}
