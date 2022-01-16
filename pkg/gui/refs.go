package gui

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/popup"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

func (gui *Gui) CheckoutRef(ref string, options types.CheckoutRefOptions) error {
	waitingStatus := options.WaitingStatus
	if waitingStatus == "" {
		waitingStatus = gui.Tr.CheckingOutStatus
	}

	cmdOptions := git_commands.CheckoutOptions{Force: false, EnvVars: options.EnvVars}

	onSuccess := func() {
		gui.State.Panels.Branches.SelectedLineIdx = 0
		gui.State.Panels.Commits.SelectedLineIdx = 0
		// loading a heap of commits is slow so we limit them whenever doing a reset
		gui.State.Panels.Commits.LimitCommits = true
	}

	return gui.PopupHandler.WithWaitingStatus(waitingStatus, func() error {
		if err := gui.Git.Branch.Checkout(ref, cmdOptions); err != nil {
			// note, this will only work for english-language git commands. If we force git to use english, and the error isn't this one, then the user will receive an english command they may not understand. I'm not sure what the best solution to this is. Running the command once in english and a second time in the native language is one option

			if options.OnRefNotFound != nil && strings.Contains(err.Error(), "did not match any file(s) known to git") {
				return options.OnRefNotFound(ref)
			}

			if strings.Contains(err.Error(), "Please commit your changes or stash them before you switch branch") {
				// offer to autostash changes
				return gui.PopupHandler.Ask(popup.AskOpts{

					Title:  gui.Tr.AutoStashTitle,
					Prompt: gui.Tr.AutoStashPrompt,
					HandleConfirm: func() error {
						if err := gui.Git.Stash.Save(gui.Tr.StashPrefix + ref); err != nil {
							return gui.PopupHandler.Error(err)
						}
						if err := gui.Git.Branch.Checkout(ref, cmdOptions); err != nil {
							return gui.PopupHandler.Error(err)
						}

						onSuccess()
						if err := gui.Git.Stash.Pop(0); err != nil {
							if err := gui.Refresh(types.RefreshOptions{Mode: types.BLOCK_UI}); err != nil {
								return err
							}
							return gui.PopupHandler.Error(err)
						}
						return gui.Refresh(types.RefreshOptions{Mode: types.BLOCK_UI})
					},
				})
			}

			if err := gui.PopupHandler.Error(err); err != nil {
				return err
			}
		}
		onSuccess()

		return gui.Refresh(types.RefreshOptions{Mode: types.BLOCK_UI})
	})
}

func (gui *Gui) resetToRef(ref string, strength string, envVars []string) error {
	if err := gui.Git.Commit.ResetToCommit(ref, strength, envVars); err != nil {
		return gui.PopupHandler.Error(err)
	}

	gui.State.Panels.Commits.SelectedLineIdx = 0
	gui.State.Panels.ReflogCommits.SelectedLineIdx = 0
	// loading a heap of commits is slow so we limit them whenever doing a reset
	gui.State.Panels.Commits.LimitCommits = true

	if err := gui.pushContext(gui.State.Contexts.BranchCommits); err != nil {
		return err
	}

	if err := gui.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.FILES, types.BRANCHES, types.REFLOG, types.COMMITS}}); err != nil {
		return err
	}

	return nil
}

func (gui *Gui) CreateGitResetMenu(ref string) error {
	strengths := []string{"soft", "mixed", "hard"}
	menuItems := make([]*popup.MenuItem, len(strengths))
	for i, strength := range strengths {
		strength := strength
		menuItems[i] = &popup.MenuItem{
			DisplayStrings: []string{
				fmt.Sprintf("%s reset", strength),
				style.FgRed.Sprintf("reset --%s %s", strength, ref),
			},
			OnPress: func() error {
				gui.LogAction("Reset")
				return gui.resetToRef(ref, strength, []string{})
			},
		}
	}

	return gui.PopupHandler.Menu(popup.CreateMenuOptions{
		Title: fmt.Sprintf("%s %s", gui.Tr.LcResetTo, ref),
		Items: menuItems,
	})
}
