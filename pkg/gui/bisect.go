package gui

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

func (gui *Gui) handleOpenBisectMenu() error {
	if ok, err := gui.validateNotInFilterMode(); err != nil || !ok {
		return err
	}

	title := gui.Tr.Bisect.BisectMenuTitle
	// no shame in getting this directly rather than using the cached value
	// given how cheap it is to obtain
	info := gui.Git.Bisect.GetInfo()
	selectedSha := gui.getSelectedLocalCommit().Sha
	if info.Started() {
		currentSha := info.GetCurrentSha()

		menuItems := []*menuItem{{
			displayString: fmt.Sprintf("%s (%s)", gui.Tr.Bisect.MarkSkipSelected, utils.ShortSha(selectedSha)),
			opensMenu:     true,
			onPress: func() error {
				return gui.openBisectMenuForRef(info, selectedSha)
			},
		}}
		if currentSha != "" && currentSha != selectedSha {
			menuItems = append(menuItems, &menuItem{
				displayString: fmt.Sprintf("%s (%s)", gui.Tr.Bisect.MarkSkipCurrent, utils.ShortSha(currentSha)),
				opensMenu:     true,
				onPress: func() error {
					return gui.openBisectMenuForRef(info, currentSha)
				},
			})
		}
		menuItems = append(menuItems, &menuItem{
			displayString: gui.Tr.Bisect.ResetOption,
			onPress: func() error {
				return gui.resetBisect()
			},
		})

		return gui.createMenu(
			title,
			menuItems,
			createMenuOptions{showCancel: true},
		)
	} else {
		return gui.createMenu(
			title,
			[]*menuItem{{
				displayString: gui.Tr.Bisect.MarkStart,
				opensMenu:     true,
				onPress: func() error {
					return gui.openStartBisectMenuForRef(info, selectedSha)
				},
			}},
			createMenuOptions{showCancel: true},
		)
	}
}

func (gui *Gui) openBisectMenuForRef(info *git_commands.BisectInfo, ref string) error {
	return gui.createMenu(
		gui.Tr.Bisect.MarkSkip,
		[]*menuItem{
			{
				displayString: info.NewTerm(),
				onPress: func() error {
					gui.logAction(gui.Tr.Actions.BisectMark)
					if err := gui.Git.Bisect.Mark(ref, info.NewTerm()); err != nil {
						return gui.surfaceError(err)
					}

					return gui.afterMark()
				},
			},
			{
				displayString: info.OldTerm(),
				onPress: func() error {
					gui.logAction(gui.Tr.Actions.BisectMark)
					if err := gui.Git.Bisect.Mark(ref, info.OldTerm()); err != nil {
						return gui.surfaceError(err)
					}

					return gui.afterMark()
				},
			},
			{
				displayString: "skip", // not i18n'ing because this is what it is in the CLI
				onPress: func() error {
					gui.logAction(gui.Tr.Actions.BisectSkip)
					if err := gui.Git.Bisect.Skip(ref); err != nil {
						return gui.surfaceError(err)
					}

					return gui.afterMark()
				},
			},
		},
		createMenuOptions{showCancel: true},
	)
}

func (gui *Gui) openStartBisectMenuForRef(info *git_commands.BisectInfo, ref string) error {
	return gui.createMenu(
		gui.Tr.Bisect.Mark,
		[]*menuItem{
			{
				displayString: info.NewTerm(),
				onPress: func() error {
					gui.logAction(gui.Tr.Actions.StartBisect)
					if err := gui.Git.Bisect.Start(); err != nil {
						return gui.surfaceError(err)
					}

					if err := gui.Git.Bisect.Mark(ref, info.NewTerm()); err != nil {
						return gui.surfaceError(err)
					}

					return gui.postBisectCommandRefresh()
				},
			},
			{
				displayString: info.OldTerm(),
				onPress: func() error {
					gui.logAction(gui.Tr.Actions.StartBisect)
					if err := gui.Git.Bisect.Start(); err != nil {
						return gui.surfaceError(err)
					}

					if err := gui.Git.Bisect.Mark(ref, info.OldTerm()); err != nil {
						return gui.surfaceError(err)
					}

					return gui.postBisectCommandRefresh()
				},
			},
		},
		createMenuOptions{showCancel: true},
	)
}

func (gui *Gui) resetBisect() error {
	return gui.ask(askOpts{
		title:  gui.Tr.Bisect.ResetTitle,
		prompt: gui.Tr.Bisect.ResetPrompt,
		handleConfirm: func() error {
			gui.logAction(gui.Tr.Actions.ResetBisect)
			if err := gui.Git.Bisect.Reset(); err != nil {
				return gui.surfaceError(err)
			}

			return gui.postBisectCommandRefresh()
		},
	})
}

func (gui *Gui) showBisectCompleteMessage(candidateShas []string) error {
	prompt := gui.Tr.Bisect.CompletePrompt
	if len(candidateShas) > 1 {
		prompt = gui.Tr.Bisect.CompletePromptIndeterminate
	}

	formattedCommits, err := gui.Git.Commit.GetCommitsOneline(candidateShas)
	if err != nil {
		return gui.surfaceError(err)
	}

	return gui.ask(askOpts{
		title:  gui.Tr.Bisect.CompleteTitle,
		prompt: fmt.Sprintf(prompt, strings.TrimSpace(formattedCommits)),
		handleConfirm: func() error {
			gui.logAction(gui.Tr.Actions.ResetBisect)
			if err := gui.Git.Bisect.Reset(); err != nil {
				return gui.surfaceError(err)
			}

			return gui.postBisectCommandRefresh()
		},
	})
}

func (gui *Gui) postBisectCommandRefresh() error {
	return gui.refreshSidePanels(refreshOptions{mode: ASYNC, scope: []RefreshableView{}})
}

func (gui *Gui) afterMark() error {
	done, candidateShas, err := gui.Git.Bisect.IsDone()
	if err != nil {
		return gui.surfaceError(err)
	}

	if done {
		return gui.showBisectCompleteMessage(candidateShas)
	}

	return gui.postBisectCommandRefresh()
}
