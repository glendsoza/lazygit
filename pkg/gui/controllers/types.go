package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/gui/popup"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type IGuiCommon interface {
	popup.IPopupHandler

	LogAction(action string)
	LogCommand(cmdStr string, isCommandLine bool)
	Refresh(types.RefreshOptions) error
	RunSubprocessAndRefresh(oscommands.ICmdObj) error
	PushContext(context types.Context) error
	PopContext() error
}
