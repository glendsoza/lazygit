package gui

import (
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers"
	"github.com/jesseduffield/lazygit/pkg/gui/popup"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// hacking this by including the gui struct for now until we split more things out
type guiCommon struct {
	gui *Gui
	popup.IPopupHandler
}

var _ controllers.IGuiCommon = &guiCommon{}

func (self *guiCommon) LogAction(msg string) {
	self.gui.LogAction(msg)
}

func (self *guiCommon) LogCommand(cmdStr string, isCommandLine bool) {
	self.gui.LogCommand(cmdStr, isCommandLine)
}

func (self *guiCommon) Refresh(opts types.RefreshOptions) error {
	return self.gui.refreshSidePanels(opts)
}

func (self *guiCommon) RunSubprocessAndRefresh(cmdObj oscommands.ICmdObj) error {
	return self.gui.runSubprocessWithSuspenseAndRefresh(cmdObj)
}
