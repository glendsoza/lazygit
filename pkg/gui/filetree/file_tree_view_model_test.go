package filetree

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
)

func TestFilterAction(t *testing.T) {
	scenarios := []struct {
		name     string
		filter   FileTreeDisplayFilter
		files    []*models.File
		expected []*models.File
	}{
		{
			name:   "filter files with unstaged changes",
			filter: DisplayUnstaged,
			files: []*models.File{
				{Name: "dir2/dir2/file4", ShortStatus: "M ", HasUnstagedChanges: true},
				{Name: "dir2/file5", ShortStatus: "M ", HasStagedChanges: true},
				{Name: "file1", ShortStatus: "M ", HasUnstagedChanges: true},
			},
			expected: []*models.File{
				{Name: "dir2/dir2/file4", ShortStatus: "M ", HasUnstagedChanges: true},
				{Name: "file1", ShortStatus: "M ", HasUnstagedChanges: true},
			},
		},
		{
			name:   "filter files with staged changes",
			filter: DisplayStaged,
			files: []*models.File{
				{Name: "dir2/dir2/file4", ShortStatus: "M ", HasStagedChanges: true},
				{Name: "dir2/file5", ShortStatus: "M ", HasStagedChanges: false},
				{Name: "file1", ShortStatus: "M ", HasStagedChanges: true},
			},
			expected: []*models.File{
				{Name: "dir2/dir2/file4", ShortStatus: "M ", HasStagedChanges: true},
				{Name: "file1", ShortStatus: "M ", HasStagedChanges: true},
			},
		},
		{
			name:   "filter all files",
			filter: DisplayAll,
			files: []*models.File{
				{Name: "dir2/dir2/file4", ShortStatus: "M ", HasUnstagedChanges: true},
				{Name: "dir2/file5", ShortStatus: "M ", HasUnstagedChanges: true},
				{Name: "file1", ShortStatus: "M ", HasUnstagedChanges: true},
			},
			expected: []*models.File{
				{Name: "dir2/dir2/file4", ShortStatus: "M ", HasUnstagedChanges: true},
				{Name: "dir2/file5", ShortStatus: "M ", HasUnstagedChanges: true},
				{Name: "file1", ShortStatus: "M ", HasUnstagedChanges: true},
			},
		},
	}

	for _, s := range scenarios {
		s := s
		t.Run(s.name, func(t *testing.T) {
			mngr := &FileTreeViewModel{files: s.files, filter: s.filter}
			result := mngr.GetFilesForDisplay()
			assert.EqualValues(t, s.expected, result)
		})
	}
}
