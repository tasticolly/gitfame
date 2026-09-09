package validaton

import (
	"context"
	"errors"
	"fmt"
	"github.com/tasticolly/gitfame/internal/config"
	"github.com/tasticolly/gitfame/internal/git"
	"os"
	"path"
)

func Flags(
	ctx context.Context,
	repositoryFlag, revisionFlag, orderFlag, formatFlag string,
	extensionsFlag, languagesFlag, excludeFlag, restrictToFlag []string,
) error {
	err := validateRepository(repositoryFlag)
	if err != nil {
		return err
	}

	err = validateRevision(ctx, repositoryFlag, revisionFlag)
	if err != nil {
		return err
	}

	err = validateOrderBy(orderFlag)
	if err != nil {
		return err
	}

	err = validateFormat(formatFlag)
	if err != nil {
		return err
	}

	err = validateExtensions(extensionsFlag)
	if err != nil {
		return err
	}

	err = validateLanguages(languagesFlag)
	if err != nil {
		return err
	}

	err = validateExclude(excludeFlag)
	if err != nil {
		return err
	}

	err = validateRestrictTo(restrictToFlag)
	if err != nil {
		return err
	}

	return nil
}

func isFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func validateRepository(repositoryPath string) error {
	exits, err := isFileExists(repositoryPath)
	if err != nil {
		return err
	}
	if !exits {
		return errors.New("git repository does not exist on path: " + repositoryPath)
	}
	return nil
}

func validateRevision(ctx context.Context, repository, revision string) error {
	ok := git.IsRevisionExists(ctx, repository, revision)
	if !ok {
		return errors.New("git revision does not exists: " + revision)
	}
	return nil
}

func validateOrderBy(order string) error {
	validOrder := map[string]struct{}{
		"lines":   {},
		"commits": {},
		"files":   {},
	}
	if _, ok := validOrder[order]; !ok {
		return errors.New("order not supported: " + order)
	}
	return nil
}

func validateFormat(outputFormat string) error {
	validFormat := map[string]struct{}{
		"tabular":    {},
		"csv":        {},
		"json":       {},
		"json-lines": {},
	}

	if _, ok := validFormat[outputFormat]; !ok {
		return errors.New("output format not supported: " + outputFormat)
	}
	return nil
}

func validateExtensions(extensions []string) error {
	for _, ext := range extensions {
		if ext[0] != '.' {
			return errors.New("extension must starts with '.' " + ext)
		}
	}
	return nil
}

func validateLanguages(languages []string) error {
	langsToExtensions, err := config.LanguageExtensionsToMap()
	if err != nil {
		return err
	}

	for _, lang := range languages {
		_, ok := langsToExtensions[lang]
		if !ok {
			fmt.Fprintf(os.Stderr, "warning: unknown language: %s\n", lang)
		}
	}

	return nil
}

func validateExclude(excludePatterns []string) error {
	return validateGlobPatterns(excludePatterns, "exclude")
}

func validateRestrictTo(restrictToPatterns []string) error {
	return validateGlobPatterns(restrictToPatterns, "restrict-to")
}

func validateGlobPatterns(patterns []string, flagName string) error {
	for _, pattern := range patterns {
		_, err := path.Match(pattern, "")
		if err != nil {
			return fmt.Errorf("uncorrect syntax in '%s' pattern: %s", flagName, pattern)
		}
	}
	return nil
}
