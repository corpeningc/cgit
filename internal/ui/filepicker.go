package ui

import (
	"github.com/charmbracelet/huh"
)

func SelectFiles(files []string) ([]string, error) {
	var selectedFiles []string
	var options []huh.Option[string]

	for _, file := range files {
		options = append(options, huh.NewOption(file, file))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
			Title("Select files to add:").
			Options(options...).
			Value(&selectedFiles),
		),
	)

	err := form.Run()
	if err != nil {
		return nil, err
	}

	return selectedFiles, nil
}
