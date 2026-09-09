package selection

import (
	"github.com/tasticolly/gitfame/internal/config"
	"path"
	"path/filepath"
)

func SelectFiles(files, extensions, languages, excludePatterns, restrictToPatterns []string) ([]string, error) {
	files, err := selectByExtensions(files, extensions, languages)
	if err != nil {
		return nil, err
	}
	files, err = selectByGlob(files, excludePatterns, restrictToPatterns)
	if err != nil {
		return nil, err
	}
	return files, nil

}

func getActualExtension(extensions, languages []string) (map[string]struct{}, error) {
	langsToExtensions, err := config.LanguageExtensionsToMap()

	if err != nil {
		return nil, err
	}

	//do it so that extra memory is not wasted
	//we no longer need to store all of them
	config.FreeLanguageExtensions()

	actualExtensions := make(map[string]struct{})

	for _, extension := range extensions {
		actualExtensions[extension] = struct{}{}
	}

	for _, language := range languages {
		currentExtensions := langsToExtensions[language]
		for _, extension := range currentExtensions {
			actualExtensions[extension] = struct{}{}
		}
	}
	return actualExtensions, nil
}

func selectByExtensions(files, extensions, languages []string) ([]string, error) {
	if len(extensions) == 0 && len(languages) == 0 {
		return files, nil
	}

	extensionsSet, err := getActualExtension(extensions, languages)
	if err != nil {
		return nil, err
	}

	suitableFiles := make([]string, 0, len(files))

	for _, file := range files {
		_, ok := extensionsSet[path.Ext(file)]
		if ok {
			suitableFiles = append(suitableFiles, file)
		}
	}
	return suitableFiles, nil
}

func selectByGlob(files, excludePatterns, restrictToPatterns []string) ([]string, error) {
	if len(excludePatterns) == 0 && len(restrictToPatterns) == 0 {
		return files, nil
	}

	suitableFiles := make([]string, 0, len(files))

	for _, file := range files {
		ok := isFileSuitable(file, excludePatterns, restrictToPatterns)
		if ok {
			suitableFiles = append(suitableFiles, file)
		}
	}
	return suitableFiles, nil
}

func isFileSuitable(file string, excludePatterns, restrictToPatterns []string) bool {
	excluded := isFileExcluded(file, excludePatterns)
	if excluded {
		return false
	}

	if len(restrictToPatterns) == 0 {
		return true
	}

	for _, pattern := range restrictToPatterns {
		if ok, _ := filepath.Match(pattern, file); ok {
			return true
		}
	}
	return false
}

func isFileExcluded(file string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		if ok, _ := filepath.Match(pattern, file); ok {
			return true
		}
	}
	return false
}
