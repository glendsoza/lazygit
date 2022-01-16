package controllers

import (
	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type RemotesController struct {
	c       *ControllerCommon
	context types.IListContext
	git     *commands.GitCommand

	getSelectedRemote func() *models.Remote
	setRemoteBranches func([]*models.RemoteBranch)
	allContexts       context.ContextTree
}

var _ types.IController = &RemotesController{}

func NewRemotesController(
	c *ControllerCommon,
	context types.IListContext,
	git *commands.GitCommand,
	allContexts context.ContextTree,
	getSelectedRemote func() *models.Remote,
	setRemoteBranches func([]*models.RemoteBranch),
) *RemotesController {
	return &RemotesController{
		c:                 c,
		git:               git,
		allContexts:       allContexts,
		context:           context,
		getSelectedRemote: getSelectedRemote,
		setRemoteBranches: setRemoteBranches,
	}
}

func (self *RemotesController) Keybindings(getKey func(key string) interface{}, config config.KeybindingConfig, guards types.KeybindingGuards) []*types.Binding {
	bindings := []*types.Binding{
		{
			Key:     getKey(config.Universal.GoInto),
			Handler: self.checkSelected(self.enter),
		},
		{
			Key:     gocui.MouseLeft,
			Handler: func() error { return self.context.HandleClick(self.checkSelected(self.enter)) },
		},
	}

	return append(bindings, self.context.Keybindings(getKey, config, guards)...)
}

func (self *RemotesController) enter(remote *models.Remote) error {
	// naive implementation: get the branches from the remote and render them to the list, change the context
	self.setRemoteBranches(remote.Branches)

	newSelectedLine := 0
	if len(remote.Branches) == 0 {
		newSelectedLine = -1
	}
	self.allContexts.RemoteBranches.GetPanelState().SetSelectedLineIdx(newSelectedLine)

	return self.c.PushContext(self.allContexts.RemoteBranches)
}

func (self *RemotesController) checkSelected(callback func(*models.Remote) error) func() error {
	return func() error {
		file := self.getSelectedRemote()
		if file == nil {
			return nil
		}

		return callback(file)
	}
}

func (self *RemotesController) Context() types.Context {
	return self.context
}
