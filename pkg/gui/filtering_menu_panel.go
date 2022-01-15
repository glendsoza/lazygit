package gui

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gui/popup"
)

func (gui *Gui) handleCreateFilteringMenuPanel() error {
	fileName := ""
	switch gui.currentSideListContext() {
	case gui.State.Contexts.Files:
		node := gui.getSelectedFileNode()
		if node != nil {
			fileName = node.GetPath()
		}
	case gui.State.Contexts.CommitFiles:
		node := gui.getSelectedCommitFileNode()
		if node != nil {
			fileName = node.GetPath()
		}
	}

	menuItems := []*popup.MenuItem{}

	if fileName != "" {
		menuItems = append(menuItems, &popup.MenuItem{
			DisplayString: fmt.Sprintf("%s '%s'", gui.Tr.LcFilterBy, fileName),
			OnPress: func() error {
				return gui.setFiltering(fileName)
			},
		})
	}

	menuItems = append(menuItems, &popup.MenuItem{
		DisplayString: gui.Tr.LcFilterPathOption,
		OnPress: func() error {
			return gui.PopupHandler.Prompt(popup.PromptOpts{
				FindSuggestionsFunc: gui.getFilePathSuggestionsFunc(),
				Title:               gui.Tr.EnterFileName,
				HandleConfirm: func(response string) error {
					return gui.setFiltering(strings.TrimSpace(response))
				},
			})
		},
	})

	if gui.State.Modes.Filtering.Active() {
		menuItems = append(menuItems, &popup.MenuItem{
			DisplayString: gui.Tr.LcExitFilterMode,
			OnPress:       gui.clearFiltering,
		})
	}

	return gui.PopupHandler.Menu(popup.CreateMenuOptions{Title: gui.Tr.FilteringMenuTitle, Items: menuItems})
}
