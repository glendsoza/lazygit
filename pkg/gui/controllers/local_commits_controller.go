package controllers

import (
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/popup"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

type LocalCommitsController struct {
	// I've said publicly that I'm against single-letter variable names but in this
	// case I would actually prefer a _zero_ letter variable name in the form of
	// struct embedding, but Go does not allow hiding public fields in an embedded struct
	// to the client
	c                               *ControllerCommon
	os                              *oscommands.OSCommand
	git                             *commands.GitCommand
	getSelectedLocalCommit          func() *models.Commit
	getCommits                      func() []*models.Commit
	getSelectedLocalCommitIdx       func() int
	handleGenericMergeCommandResult func(error) error
	pullFiles                       func() error
}

var _ IController = &LocalCommitsController{}

func NewLocalCommitsController(
	c *ControllerCommon,
	os *oscommands.OSCommand,
	git *commands.GitCommand,
	getSelectedLocalCommit func() *models.Commit,
	getCommits func() []*models.Commit,
	getSelectedLocalCommitIdx func() int,
	handleGenericMergeCommandResult func(error) error,
	pullFiles func() error,
) *LocalCommitsController {
	return &LocalCommitsController{
		c:                               c,
		os:                              os,
		git:                             git,
		getSelectedLocalCommit:          getSelectedLocalCommit,
		getCommits:                      getCommits,
		getSelectedLocalCommitIdx:       getSelectedLocalCommitIdx,
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

// handleMidRebaseCommand sees if the selected commit is in fact a rebasing
// commit meaning you are trying to edit the todo file rather than actually
// begin a rebase. It then updates the todo file with that action
func (self *LocalCommitsController) handleMidRebaseCommand(action string) (bool, error) {
	selectedCommit := self.getSelectedLocalCommit()
	if selectedCommit.Status != "rebasing" {
		return false, nil
	}

	// for now we do not support setting 'reword' because it requires an editor
	// and that means we either unconditionally wait around for the subprocess to ask for
	// our input or we set a lazygit client as the EDITOR env variable and have it
	// request us to edit the commit message when prompted.
	if action == "reword" {
		return true, self.c.ErrorMsg(self.c.Tr.LcRewordNotSupported)
	}

	self.c.LogAction("Update rebase TODO")
	self.c.LogCommand(
		fmt.Sprintf("Updating rebase action of commit %s to '%s'", selectedCommit.ShortSha(), action),
		false,
	)

	if err := self.git.Rebase.EditRebaseTodo(
		self.getSelectedLocalCommitIdx(), action,
	); err != nil {
		return false, self.c.Error(err)
	}

	return true, self.c.Refresh(types.RefreshOptions{
		Mode: types.SYNC, Scope: []types.RefreshableView{types.REBASE_COMMITS},
	})
}

func (self *LocalCommitsController) handleCommitMoveDown() error {
	index := gui.State.Panels.Commits.SelectedLineIdx
	selectedCommit := gui.State.Commits[index]
	if selectedCommit.Status == "rebasing" {
		if gui.State.Commits[index+1].Status != "rebasing" {
			return nil
		}

		// logging directly here because MoveTodoDown doesn't have enough information
		// to provide a useful log
		self.c.LogAction(self.c.Tr.Actions.MoveCommitDown)
		self.c.LogCommand(fmt.Sprintf("Moving commit %s down", selectedCommit.ShortSha()), false)

		if err := self.git.Rebase.MoveTodoDown(index); err != nil {
			return self.c.Error(err)
		}
		gui.State.Panels.Commits.SelectedLineIdx++
		return self.c.Refresh(types.RefreshOptions{
			Mode: types.SYNC, Scope: []types.RefreshableView{types.REBASE_COMMITS},
		})
	}

	return self.c.WithWaitingStatus(self.c.Tr.MovingStatus, func() error {
		self.c.LogAction(self.c.Tr.Actions.MoveCommitDown)
		err := self.git.Rebase.MoveCommitDown(gui.State.Commits, index)
		if err == nil {
			gui.State.Panels.Commits.SelectedLineIdx++
		}
		return self.handleGenericMergeCommandResult(err)
	})
}

func (self *LocalCommitsController) handleCommitMoveUp() error {
	index := gui.State.Panels.Commits.SelectedLineIdx
	if index == 0 {
		return nil
	}

	selectedCommit := gui.State.Commits[index]
	if selectedCommit.Status == "rebasing" {
		// logging directly here because MoveTodoDown doesn't have enough information
		// to provide a useful log
		self.c.LogAction(self.c.Tr.Actions.MoveCommitUp)
		self.c.LogCommand(
			fmt.Sprintf("Moving commit %s up", selectedCommit.ShortSha()),
			false,
		)

		if err := self.git.Rebase.MoveTodoDown(index - 1); err != nil {
			return self.c.Error(err)
		}
		gui.State.Panels.Commits.SelectedLineIdx--
		return self.c.Refresh(types.RefreshOptions{
			Mode: types.SYNC, Scope: []types.RefreshableView{types.REBASE_COMMITS},
		})
	}

	return self.c.WithWaitingStatus(self.c.Tr.MovingStatus, func() error {
		self.c.LogAction(self.c.Tr.Actions.MoveCommitUp)
		err := self.git.Rebase.MoveCommitDown(gui.State.Commits, index-1)
		if err == nil {
			gui.State.Panels.Commits.SelectedLineIdx--
		}
		return self.handleGenericMergeCommandResult(err)
	})
}

func (self *LocalCommitsController) handleCommitAmendTo() error {
	return self.c.Ask(popup.AskOpts{
		Title:  self.c.Tr.AmendCommitTitle,
		Prompt: self.c.Tr.AmendCommitPrompt,
		HandleConfirm: func() error {
			return self.c.WithWaitingStatus(self.c.Tr.AmendingStatus, func() error {
				self.c.LogAction(self.c.Tr.Actions.AmendCommit)
				err := self.git.Rebase.AmendTo(gui.State.Commits[gui.State.Panels.Commits.SelectedLineIdx].Sha)
				return self.handleGenericMergeCommandResult(err)
			})
		},
	})
}

func (self *LocalCommitsController) handleCommitRevert() error {
	commit := self.getSelectedLocalCommit()

	if commit.IsMerge() {
		return self.createRevertMergeCommitMenu(commit)
	} else {
		self.c.LogAction(self.c.Tr.Actions.RevertCommit)
		if err := self.git.Commit.Revert(commit.Sha); err != nil {
			return self.c.Error(err)
		}
		return self.afterRevertCommit()
	}
}

func (self *LocalCommitsController) createRevertMergeCommitMenu(commit *models.Commit) error {
	menuItems := make([]*popup.MenuItem, len(commit.Parents))
	for i, parentSha := range commit.Parents {
		i := i
		message, err := self.git.Commit.GetCommitMessageFirstLine(parentSha)
		if err != nil {
			return self.c.Error(err)
		}

		menuItems[i] = &popup.MenuItem{
			DisplayString: fmt.Sprintf("%s: %s", utils.SafeTruncate(parentSha, 8), message),
			OnPress: func() error {
				parentNumber := i + 1
				self.c.LogAction(self.c.Tr.Actions.RevertCommit)
				if err := self.git.Commit.RevertMerge(commit.Sha, parentNumber); err != nil {
					return self.c.Error(err)
				}
				return self.afterRevertCommit()
			},
		}
	}

	return self.c.Menu(popup.CreateMenuOptions{Title: self.c.Tr.SelectParentCommitForMerge, Items: menuItems})
}

func (self *LocalCommitsController) afterRevertCommit() error {
	gui.State.Panels.Commits.SelectedLineIdx++
	return self.c.Refresh(types.RefreshOptions{
		Mode: types.BLOCK_UI, Scope: []types.RefreshableView{types.COMMITS, types.BRANCHES},
	})
}

func (self *LocalCommitsController) handleViewCommitFiles() error {
	commit := self.getSelectedLocalCommit()
	if commit == nil {
		return nil
	}

	return self.switchToCommitFilesContext(commit.Sha, true, gui.State.Contexts.BranchCommits, "commits")
}

func (self *LocalCommitsController) handleCreateFixupCommit() error {
	commit := self.getSelectedLocalCommit()
	if commit == nil {
		return nil
	}

	prompt := utils.ResolvePlaceholderString(
		self.c.Tr.SureCreateFixupCommit,
		map[string]string{
			"commit": commit.Sha,
		},
	)

	return self.c.Ask(popup.AskOpts{
		Title:  self.c.Tr.CreateFixupCommit,
		Prompt: prompt,
		HandleConfirm: func() error {
			self.c.LogAction(self.c.Tr.Actions.CreateFixupCommit)
			if err := self.git.Commit.CreateFixupCommit(commit.Sha); err != nil {
				return self.c.Error(err)
			}

			return self.c.Refresh(types.RefreshOptions{Mode: types.ASYNC})
		},
	})
}

func (self *LocalCommitsController) handleSquashAllAboveFixupCommits() error {
	commit := self.getSelectedLocalCommit()
	if commit == nil {
		return nil
	}

	prompt := utils.ResolvePlaceholderString(
		self.c.Tr.SureSquashAboveCommits,
		map[string]string{
			"commit": commit.Sha,
		},
	)

	return self.c.Ask(popup.AskOpts{
		Title:  self.c.Tr.SquashAboveCommits,
		Prompt: prompt,
		HandleConfirm: func() error {
			return self.c.WithWaitingStatus(self.c.Tr.SquashingStatus, func() error {
				self.c.LogAction(self.c.Tr.Actions.SquashAllAboveFixupCommits)
				err := self.git.Rebase.SquashAllAboveFixupCommits(commit.Sha)
				return self.handleGenericMergeCommandResult(err)
			})
		},
	})
}

func (self *LocalCommitsController) handleTagCommit() error {
	commit := self.getSelectedLocalCommit()
	if commit == nil {
		return nil
	}

	return self.createTagMenu(commit.Sha)
}

func (self *LocalCommitsController) createTagMenu(commitSha string) error {
	return self.c.Menu(popup.CreateMenuOptions{
		Title: self.c.Tr.TagMenuTitle,
		Items: []*popup.MenuItem{
			{
				DisplayString: self.c.Tr.LcLightweightTag,
				OnPress: func() error {
					return self.handleCreateLightweightTag(commitSha)
				},
			},
			{
				DisplayString: self.c.Tr.LcAnnotatedTag,
				OnPress: func() error {
					return self.handleCreateAnnotatedTag(commitSha)
				},
			},
		},
	})
}

func (self *LocalCommitsController) afterTagCreate() error {
	self.State.Panels.Tags.SelectedLineIdx = 0 // Set to the top
	return self.c.Refresh(types.RefreshOptions{
		Mode: types.ASYNC, Scope: []types.RefreshableView{types.COMMITS, types.TAGS},
	})
}

func (self *LocalCommitsController) handleCreateAnnotatedTag(commitSha string) error {
	return self.c.Prompt(popup.PromptOpts{
		Title: self.c.Tr.TagNameTitle,
		HandleConfirm: func(tagName string) error {
			return self.c.Prompt(popup.PromptOpts{
				Title: self.c.Tr.TagMessageTitle,
				HandleConfirm: func(msg string) error {
					self.c.LogAction(self.c.Tr.Actions.CreateAnnotatedTag)
					if err := self.git.Tag.CreateAnnotated(tagName, commitSha, msg); err != nil {
						return self.c.Error(err)
					}
					return self.afterTagCreate()
				},
			})
		},
	})
}

func (self *LocalCommitsController) handleCreateLightweightTag(commitSha string) error {
	return self.c.Prompt(popup.PromptOpts{
		Title: self.c.Tr.TagNameTitle,
		HandleConfirm: func(tagName string) error {
			self.c.LogAction(self.c.Tr.Actions.CreateLightweightTag)
			if err := self.git.Tag.CreateLightweight(tagName, commitSha); err != nil {
				return self.c.Error(err)
			}
			return self.afterTagCreate()
		},
	})
}

func (self *LocalCommitsController) handleCheckoutCommit() error {
	commit := self.getSelectedLocalCommit()
	if commit == nil {
		return nil
	}

	return self.c.Ask(popup.AskOpts{
		Title:  self.c.Tr.LcCheckoutCommit,
		Prompt: self.c.Tr.SureCheckoutThisCommit,
		HandleConfirm: func() error {
			self.c.LogAction(self.c.Tr.Actions.CheckoutCommit)
			return self.handleCheckoutRef(commit.Sha, handleCheckoutRefOptions{})
		},
	})
}

func (self *LocalCommitsController) handleCreateCommitResetMenu() error {
	commit := self.getSelectedLocalCommit()
	if commit == nil {
		return self.c.ErrorMsg(self.c.Tr.NoCommitsThisBranch)
	}

	return self.createResetMenu(commit.Sha)
}

func (self *LocalCommitsController) handleOpenSearchForCommitsPanel(string) error {
	// we usually lazyload these commits but now that we're searching we need to load them now
	if self.State.Panels.Commits.LimitCommits {
		self.State.Panels.Commits.LimitCommits = false
		if err := self.c.Refresh(types.RefreshOptions{Mode: types.ASYNC, Scope: []types.RefreshableView{types.COMMITS}}); err != nil {
			return err
		}
	}

	return self.handleOpenSearch("commits")
}

func (self *LocalCommitsController) handleGotoBottomForCommitsPanel() error {
	// we usually lazyload these commits but now that we're searching we need to load them now
	if self.State.Panels.Commits.LimitCommits {
		self.State.Panels.Commits.LimitCommits = false
		if err := self.c.Refresh(types.RefreshOptions{Mode: types.SYNC, Scope: []types.RefreshableView{types.COMMITS}}); err != nil {
			return err
		}
	}

	for _, context := range self.getListContexts() {
		if context.GetViewName() == "commits" {
			return context.handleGotoBottom()
		}
	}

	return nil
}

func (self *LocalCommitsController) handleCopySelectedCommitMessageToClipboard() error {
	commit := self.getSelectedLocalCommit()
	if commit == nil {
		return nil
	}

	message, err := self.git.Commit.GetCommitMessage(commit.Sha)
	if err != nil {
		return self.c.Error(err)
	}

	self.c.LogAction(self.c.Tr.Actions.CopyCommitMessageToClipboard)
	if err := self.os.CopyToClipboard(message); err != nil {
		return self.c.Error(err)
	}

	self.c.Toast(self.c.Tr.CommitMessageCopiedToClipboard)

	return nil
}

func (self *LocalCommitsController) handleOpenLogMenu() error {
	return self.c.Menu(popup.CreateMenuOptions{
		Title: self.c.Tr.LogMenuTitle,
		Items: []*popup.MenuItem{
			{
				DisplayString: self.c.Tr.ToggleShowGitGraphAll,
				OnPress: func() error {
					self.ShowWholeGitGraph = !self.ShowWholeGitGraph

					if self.ShowWholeGitGraph {
						self.State.Panels.Commits.LimitCommits = false
					}

					return self.c.WithWaitingStatus(self.c.Tr.LcLoadingCommits, func() error {
						return self.c.Refresh(types.RefreshOptions{Mode: types.SYNC, Scope: []types.RefreshableView{types.COMMITS}})
					})
				},
			},
			{
				DisplayString: self.c.Tr.ShowGitGraph,
				OpensMenu:     true,
				OnPress: func() error {
					onPress := func(value string) func() error {
						return func() error {
							self.c.UserConfig.Git.Log.ShowGraph = value
							return nil
						}
					}
					return self.c.Menu(popup.CreateMenuOptions{
						Title: self.c.Tr.LogMenuTitle,
						Items: []*popup.MenuItem{
							{
								DisplayString: "always",
								OnPress:       onPress("always"),
							},
							{
								DisplayString: "never",
								OnPress:       onPress("never"),
							},
							{
								DisplayString: "when maximised",
								OnPress:       onPress("when-maximised"),
							},
						},
					})
				},
			},
			{
				DisplayString: self.c.Tr.SortCommits,
				OpensMenu:     true,
				OnPress: func() error {
					onPress := func(value string) func() error {
						return func() error {
							self.c.UserConfig.Git.Log.Order = value
							return self.c.WithWaitingStatus(self.c.Tr.LcLoadingCommits, func() error {
								return self.c.Refresh(types.RefreshOptions{Mode: types.SYNC, Scope: []types.RefreshableView{types.COMMITS}})
							})
						}
					}

					return self.c.Menu(popup.CreateMenuOptions{
						Title: self.c.Tr.LogMenuTitle,
						Items: []*popup.MenuItem{
							{
								DisplayString: "topological (topo-order)",
								OnPress:       onPress("topo-order"),
							},
							{
								DisplayString: "date-order",
								OnPress:       onPress("date-order"),
							},
							{
								DisplayString: "author-date-order",
								OnPress:       onPress("author-date-order"),
							},
						},
					})
				},
			},
		},
	})
}

func (self *LocalCommitsController) handleOpenCommitInBrowser() error {
	commit := self.getSelectedLocalCommit()
	if commit == nil {
		return nil
	}

	hostingServiceMgr := self.getHostingServiceMgr()

	url, err := hostingServiceMgr.GetCommitURL(commit.Sha)
	if err != nil {
		return self.c.Error(err)
	}

	self.c.LogAction(self.c.Tr.Actions.OpenCommitInBrowser)
	if err := self.os.OpenLink(url); err != nil {
		return self.c.Error(err)
	}

	return nil
}
