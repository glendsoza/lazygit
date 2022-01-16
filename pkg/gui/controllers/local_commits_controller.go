package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/popup"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type LocalCommitsController struct {
	// I've said publicly that I'm against single-letter variable names but in this
	// case I would actually prefer a _zero_ letter variable name in the form of
	// struct embedding, but Go does not allow hiding public fields in an embedded struct
	// to the client
	c                               *ControllerCommon
	git                             *commands.GitCommand
	getSelectedLocalCommit          func() *models.Commit
	getCommits                      func() []*models.Commit
	getSelectedLocalCommitIdx       func() int
	handleMidRebaseCommand          func(string) (bool, error)
	handleGenericMergeCommandResult func(error) error
	pullFiles                       func() error
}

var _ IController = &LocalCommitsController{}

func NewLocalCommitsController(
	c *ControllerCommon,
	git *commands.GitCommand,
	getSelectedLocalCommit func() *models.Commit,
	getCommits func() []*models.Commit,
	getSelectedLocalCommitIdx func() int,
	handleMidRebaseCommand func(string) (bool, error),
	handleGenericMergeCommandResult func(error) error,
	pullFiles func() error,
) *LocalCommitsController {
	return &LocalCommitsController{
		c:                               c,
		git:                             git,
		getSelectedLocalCommit:          getSelectedLocalCommit,
		getCommits:                      getCommits,
		getSelectedLocalCommitIdx:       getSelectedLocalCommitIdx,
		handleMidRebaseCommand:          handleMidRebaseCommand,
		handleGenericMergeCommandResult: handleGenericMergeCommandResult,
		pullFiles:                       pullFiles,
	}
}

func (self *LocalCommitsController) Keybindings(
	getKey func(key string) interface{},
	config config.KeybindingConfig,
	guards types.KeybindingGuards,
) []*types.Binding {
	return []*types.Binding{
		{
			Key:         getKey(config.Commits.SquashDown),
			Handler:     guards.OutsideFilterMode(self.squashDown),
			Description: self.c.Tr.LcSquashDown,
		},
		{
			Key:         getKey(config.Commits.MarkCommitAsFixup),
			Handler:     guards.OutsideFilterMode(self.fixup),
			Description: self.c.Tr.LcFixupCommit,
		},
		{
			Key:         getKey(config.Commits.RenameCommit),
			Handler:     guards.OutsideFilterMode(self.reword),
			Description: self.c.Tr.LcRewordCommit,
		},
		{
			Key:         getKey(config.Commits.RenameCommitWithEditor),
			Handler:     guards.OutsideFilterMode(self.rewordEditor),
			Description: self.c.Tr.LcRenameCommitEditor,
		},
		{
			Key:         getKey(config.Universal.Remove),
			Handler:     guards.OutsideFilterMode(self.drop),
			Description: self.c.Tr.LcDeleteCommit,
		},
		{
			Key:         getKey(config.Universal.Edit),
			Handler:     guards.OutsideFilterMode(self.edit),
			Description: self.c.Tr.LcEditCommit,
		},
		{
			Key:         getKey(config.Commits.PickCommit),
			Handler:     guards.OutsideFilterMode(self.pick),
			Description: self.c.Tr.LcPickCommit,
		},
	}
}

func (self *LocalCommitsController) squashDown() error {
	if len(self.getCommits()) <= 1 {
		return self.c.ErrorMsg(self.c.Tr.YouNoCommitsToSquash)
	}

	applied, err := self.handleMidRebaseCommand("squash")
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	return self.c.Ask(popup.AskOpts{
		Title:  self.c.Tr.Squash,
		Prompt: self.c.Tr.SureSquashThisCommit,
		HandleConfirm: func() error {
			return self.c.WithWaitingStatus(self.c.Tr.SquashingStatus, func() error {
				self.c.LogAction(self.c.Tr.Actions.SquashCommitDown)
				return self.interactiveRebase("squash")
			})
		},
	})
}

func (self *LocalCommitsController) fixup() error {
	if len(self.getCommits()) <= 1 {
		return self.c.ErrorMsg(self.c.Tr.YouNoCommitsToSquash)
	}

	applied, err := self.handleMidRebaseCommand("fixup")
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	return self.c.Ask(popup.AskOpts{
		Title:  self.c.Tr.Fixup,
		Prompt: self.c.Tr.SureFixupThisCommit,
		HandleConfirm: func() error {
			return self.c.WithWaitingStatus(self.c.Tr.FixingStatus, func() error {
				self.c.LogAction(self.c.Tr.Actions.FixupCommit)
				return self.interactiveRebase("fixup")
			})
		},
	})
}

func (self *LocalCommitsController) reword() error {
	applied, err := self.handleMidRebaseCommand("reword")
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	commit := self.getSelectedLocalCommit()
	if commit == nil {
		return nil
	}

	message, err := self.git.Commit.GetCommitMessage(commit.Sha)
	if err != nil {
		return self.c.Error(err)
	}

	// TODO: use the commit message panel here
	return self.c.Prompt(popup.PromptOpts{
		Title:          self.c.Tr.LcRewordCommit,
		InitialContent: message,
		HandleConfirm: func(response string) error {
			self.c.LogAction(self.c.Tr.Actions.RewordCommit)
			if err := self.git.Rebase.RewordCommit(self.getCommits(), self.getSelectedLocalCommitIdx(), response); err != nil {
				return self.c.Error(err)
			}

			return self.c.Refresh(types.RefreshOptions{Mode: types.ASYNC})
		},
	})
}

func (self *LocalCommitsController) rewordEditor() error {
	applied, err := self.handleMidRebaseCommand("reword")
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	self.c.LogAction(self.c.Tr.Actions.RewordCommit)
	subProcess, err := self.git.Rebase.RewordCommitInEditor(
		self.getCommits(), self.getSelectedLocalCommitIdx(),
	)
	if err != nil {
		return self.c.Error(err)
	}
	if subProcess != nil {
		return self.c.RunSubprocessAndRefresh(subProcess)
	}

	return nil
}

func (self *LocalCommitsController) drop() error {
	applied, err := self.handleMidRebaseCommand("drop")
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	return self.c.Ask(popup.AskOpts{
		Title:  self.c.Tr.DeleteCommitTitle,
		Prompt: self.c.Tr.DeleteCommitPrompt,
		HandleConfirm: func() error {
			return self.c.WithWaitingStatus(self.c.Tr.DeletingStatus, func() error {
				self.c.LogAction(self.c.Tr.Actions.DropCommit)
				return self.interactiveRebase("drop")
			})
		},
	})
}

func (self *LocalCommitsController) edit() error {
	applied, err := self.handleMidRebaseCommand("edit")
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	return self.c.WithWaitingStatus(self.c.Tr.RebasingStatus, func() error {
		self.c.LogAction(self.c.Tr.Actions.EditCommit)
		return self.interactiveRebase("edit")
	})
}

func (self *LocalCommitsController) pick() error {
	applied, err := self.handleMidRebaseCommand("pick")
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	// at this point we aren't actually rebasing so we will interpret this as an
	// attempt to pull. We might revoke this later after enabling configurable keybindings
	return self.pullFiles()
}

func (self *LocalCommitsController) interactiveRebase(action string) error {
	err := self.git.Rebase.InteractiveRebase(self.getCommits(), self.getSelectedLocalCommitIdx(), action)
	return self.handleGenericMergeCommandResult(err)
}
