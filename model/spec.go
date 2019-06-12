package model

import log "github.com/sirupsen/logrus"

type Spec struct {
	Config    *ConfigSpec `yaml:"CONFIG"`
	Inventory Inventory   `yaml:"INVENTORY"`
	Wallets   Wallets     `yaml:"WALLETS"`
	Contracts Contracts   `yaml:"CONTRACTS"`
	Targets   Targets     `yaml:"TARGETS"`

	ViewCmds  ViewCmds  `yaml:"VIEW"`
	WriteCmds WriteCmds `yaml:"WRITE"`
	CallCmds  CallCmds  `yaml:"CALL"`

	uniqueNames map[string]struct{} `yaml:"-"`
}

func (spec *Spec) Validate(ctx AppContext) bool {
	validateLog := log.WithFields(log.Fields{
		"model": "Spec",
		"func":  "Validate",
	})
	if spec.Config == nil {
		spec.Config = DefaultConfigSpec
	} else if !spec.Config.Validate() {
		validateLog.Errorln("config spec validation failed")
		return false
	}
	if len(ctx.AppCommand()) > 0 {
		if spec.Inventory == nil {
			validateLog.Errorln("spec must contain INVENTORY section")
			return false
		} else if !spec.Inventory.Validate(ctx, spec) {
			validateLog.Errorln("inventory spec validation failed")
			return false
		}
	}
	if spec.ViewCmds == nil && spec.WriteCmds == nil && spec.CallCmds == nil {
		validateLog.Errorln("spec must contain at least one of VIEW, WRITE or CALL sections")
		return false
	}
	if spec.Wallets != nil {
		if !spec.Wallets.Validate(ctx, spec) {
			validateLog.Errorln("wallets spec validation failed")
			return false
		}
	} else if spec.WriteCmds != nil || spec.CallCmds != nil {
		validateLog.Errorln("spec must contain the WALLET section, if WRITE or CALL sections are provided")
		return false
	}
	if spec.Contracts != nil {
		if !spec.Contracts.Validate(ctx, spec) {
			validateLog.Errorln("contracts spec validation failed")
			return false
		}
	}
	spec.uniqueNames = make(map[string]struct{})
	if spec.CallCmds != nil {
		if !spec.CallCmds.Validate(ctx, spec) {
			validateLog.Errorln("call cmds spec validation failed")
			return false
		}
	}
	if spec.ViewCmds != nil {
		if !spec.ViewCmds.Validate(ctx, spec) {
			validateLog.Errorln("view cmds spec validation failed")
			return false
		}
	}
	if spec.WriteCmds != nil {
		if !spec.WriteCmds.Validate(ctx, spec) {
			validateLog.Errorln("write cmds spec validation failed")
			return false
		}
	}
	if spec.Targets != nil {
		if !spec.Targets.Validate(ctx, spec) {
			validateLog.Errorln("targets spec validation failed")
			return false
		}
	}
	return true
}

func (spec *Spec) CountArgsUsing(set map[int]struct{}, name string) {
	if cmd, ok := spec.CallCmds[name]; ok {
		cmd.CountArgsUsing(set)
	} else if cmd, ok := spec.ViewCmds[name]; ok {
		cmd.CountArgsUsing(set)
	} else if cmd, ok := spec.WriteCmds[name]; ok {
		cmd.CountArgsUsing(set)
	}
}

func (spec *Spec) ArgCount(name string) int {
	if cmd, ok := spec.CallCmds[name]; ok {
		return cmd.ArgCount()
	} else if cmd, ok := spec.ViewCmds[name]; ok {
		return cmd.ArgCount()
	} else if cmd, ok := spec.WriteCmds[name]; ok {
		return cmd.ArgCount()
	}
	return 0
}

type FieldName string
