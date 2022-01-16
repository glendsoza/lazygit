package gui

func (gui *Gui) toggleWhitespaceInDiffView() error {
	gui.IgnoreWhitespaceInDiffView = !gui.IgnoreWhitespaceInDiffView

	toastMessage := gui.Tr.ShowingWhitespaceInDiffView
	if gui.IgnoreWhitespaceInDiffView {
		toastMessage = gui.Tr.IgnoringWhitespaceInDiffView
	}
	gui.PopupHandler.Toast(toastMessage)

	return gui.refreshFilesAndSubmodules()
}
