package controllers

import (
	"github.com/jesseduffield/gocui"
	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type FilesController struct {
	// I've said publicly that I'm against single-letter variable names but in this
	// case I would actually prefer a _zero_ letter variable name in the form of
	// struct embedding, but Go does not allow hiding public fields in an embedded struct
	// to the client
	c       *ControllerCommon
	context types.IListContext
	git     *commands.GitCommand

	getSelectedFileNode func() *filetree.FileNode
	allContexts         context.ContextTree
}

var _ types.IController = &FilesController{}

func NewFilesController(
	c *ControllerCommon,
	context types.IListContext,
	git *commands.GitCommand,
	getSelectedFileNode func() *filetree.FileNode,
	allContexts context.ContextTree,
) *FilesController {
	return &FilesController{
		c:                   c,
		context:             context,
		git:                 git,
		getSelectedFileNode: getSelectedFileNode,
		allContexts:         allContexts,
	}
}

func (self *FilesController) Keybindings(getKey func(key string) interface{}, config config.KeybindingConfig, guards types.KeybindingGuards) []*types.Binding {
	bindings := []*types.Binding{
		{
			Key:         getKey(config.Universal.Select),
			Handler:     self.checkSelectedFileNode(self.enter),
			Description: self.c.Tr.LcToggleStaged,
		},
		{
			Key:     gocui.MouseLeft,
			Handler: func() error { return self.context.HandleClick(self.checkSelectedFileNode(self.enter)) },
		},
	}

	return append(bindings, self.context.Keybindings(getKey, config, guards)...)
}

func (self *FilesController) enter(node *filetree.FileNode) error {
	if node.IsLeaf() {
		file := node.File

		if file.HasInlineMergeConflicts {
			return self.c.PushContext(self.allContexts.Merging)
		}

		if file.HasUnstagedChanges {
			self.c.LogAction(self.c.Tr.Actions.StageFile)
			if err := self.git.WorkingTree.StageFile(file.Name); err != nil {
				return self.c.Error(err)
			}
		} else {
			self.c.LogAction(self.c.Tr.Actions.UnstageFile)
			if err := self.git.WorkingTree.UnStageFile(file.Names(), file.Tracked); err != nil {
				return self.c.Error(err)
			}
		}
	} else {
		// if any files within have inline merge conflicts we can't stage or unstage,
		// or it'll end up with those >>>>>> lines actually staged
		if node.GetHasInlineMergeConflicts() {
			return self.c.ErrorMsg(self.c.Tr.ErrStageDirWithInlineMergeConflicts)
		}

		if node.GetHasUnstagedChanges() {
			self.c.LogAction(self.c.Tr.Actions.StageFile)
			if err := self.git.WorkingTree.StageFile(node.Path); err != nil {
				return self.c.Error(err)
			}
		} else {
			// pretty sure it doesn't matter that we're always passing true here
			self.c.LogAction(self.c.Tr.Actions.UnstageFile)
			if err := self.git.WorkingTree.UnStageFile([]string{node.Path}, true); err != nil {
				return self.c.Error(err)
			}
		}
	}

	if err := self.c.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.FILES}}); err != nil {
		return err
	}

	return self.context.HandleFocus()
}

func (self *FilesController) checkSelectedFileNode(callback func(*filetree.FileNode) error) func() error {
	return func() error {
		node := self.getSelectedFileNode()
		if node == nil {
			return nil
		}

		return callback(node)
	}
}

func (self *FilesController) checkSelectedFile(callback func(*models.File) error) func() error {
	return func() error {
		file := self.getSelectedFile()
		if file == nil {
			return nil
		}

		return callback(file)
	}
}

func (self *FilesController) Context() types.Context {
	return self.context
}

func (self *FilesController) getSelectedFile() *models.File {
	node := self.getSelectedFileNode()
	if node == nil {
		return nil
	}
	return node.File
}
