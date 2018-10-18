package model

import (
	"context"

	"github.com/AtlantPlatform/ethfw"
	"github.com/AtlantPlatform/ethfw/sol"
)

type AppContext struct {
	context.Context
}

func NewAppContext(ctx context.Context, appCommand string, appCommandArgs []string, nodeGroup string,
	specDir string, solcCompiler sol.Compiler, keycache ethfw.KeyCache) AppContext {
	ctx = context.WithValue(ctx, "prefix", "playbook")
	ctx = context.WithValue(ctx, "cmd", appCommand)
	ctx = context.WithValue(ctx, "args", appCommandArgs)
	ctx = context.WithValue(ctx, "group", nodeGroup)
	ctx = context.WithValue(ctx, "specdir", specDir)
	ctx = context.WithValue(ctx, "keycache", keycache)
	ctx = context.WithValue(ctx, "sol", solcCompiler)
	return AppContext{ctx}
}

func (ctx AppContext) AppCommand() string {
	return ctx.Value("cmd").(string)
}

func (ctx AppContext) AppCommandArgs() []string {
	return ctx.Value("args").([]string)
}

func (ctx AppContext) NodeGroup() string {
	return ctx.Value("group").(string)
}

func (ctx AppContext) SpecDir() string {
	return ctx.Value("specdir").(string)
}

func (ctx AppContext) SolcCompiler() sol.Compiler {
	return ctx.Value("sol").(sol.Compiler)
}

func (ctx AppContext) KeyCache() ethfw.KeyCache {
	return ctx.Value("keycache").(ethfw.KeyCache)
}
