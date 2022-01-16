package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type IController interface {
	Keybindings(
		getKey func(key string) interface{},
		config config.KeybindingConfig,
		guards types.KeybindingGuards,
	) []*types.Binding
}
