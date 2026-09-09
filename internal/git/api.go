package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

type CommitInfo struct {
	Person     string
	NumOfLines int
}

func GetFileNames(ctx context.Context, repository, revision string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-tree", "-r", "--name-only", revision)
	cmd.Dir = repository
	out, err := cmd.Output()
	if err != nil {
		return nil, errors.New("ls-tree failed: " + err.Error())
	}
	fileNames := strings.TrimSpace(string(out))
	if len(fileNames) == 0 {
		fmt.Fprint(os.Stderr, "warning: no files found\n")
		return nil, nil
	}
	return strings.Split(fileNames, "\n"), nil
}

func IsRevisionExists(ctx context.Context, repository, revision string) bool {
	cmd := exec.CommandContext(ctx, "git", "cat-file", "-e", revision)
	cmd.Dir = repository
	_ = cmd.Start()
	if err := cmd.Wait(); err != nil {
		return false
	}
	return true
}

func lastChangePerson(ctx context.Context, repository, revision, filename string, useCommitter bool) ([]string, error) {
	personFormat := "%an"
	if useCommitter {
		personFormat = "%cn"
	}
	prettyFormat := fmt.Sprintf("--pretty='%s%%n%%H'", personFormat)
	cmd := exec.CommandContext(ctx, "git", "log", "-n1", prettyFormat, revision, "--", filename)
	cmd.Dir = repository
	out, err := cmd.Output()
	if err != nil {
		return nil, errors.New("git log failed: " + err.Error())
	}
	return strings.Split(string(out)[1:len(out)-2], "\n"), nil
}

func GetInfo(ctx context.Context, repository, revision, filename string, useCommitter bool) (map[string]*CommitInfo, error) {
	cmd := exec.CommandContext(ctx, "git", "blame", "--incremental", revision, filename)
	cmd.Dir = repository

	out, err := cmd.Output()

	if err != nil {
		return nil, errors.New("git blame failed: " + err.Error())
	}

	if len(out) == 0 {
		personAndHash, err := lastChangePerson(ctx, repository, revision, filename, useCommitter)
		if err != nil {
			return nil, err
		}
		return map[string]*CommitInfo{
			personAndHash[1]: {Person: personAndHash[0], NumOfLines: 0},
		}, nil
	}

	person := "author"
	if useCommitter {
		person = "committer"
	}

	commitToInfo := make(map[string]*CommitInfo)

	blameString := strings.Split(string(out), "\n")
	hashRegex, _ := regexp.Compile(`^[a-z0-9]{40}`)
	numOfLinesRegex, _ := regexp.Compile(`\d+$`)

	findPersonMode := false

	var hash string

	for _, line := range blameString {
		if !findPersonMode {
			hash = hashRegex.FindString(line)
			if hash == "" {
				continue
			}

			numOfLines, err := strconv.Atoi(numOfLinesRegex.FindString(line))
			if err != nil {
				return nil, errors.New("error in parsing blame: " + err.Error())
			}

			info, ok := commitToInfo[hash]
			if !ok {
				commitToInfo[hash] = &CommitInfo{NumOfLines: numOfLines}
				findPersonMode = true
			} else {
				info.NumOfLines += numOfLines
			}

		} else {
			if !strings.HasPrefix(line, person) {
				continue
			}
			findPersonMode = false
			commitToInfo[hash].Person = line[len(person)+1:]
		}
	}
	return commitToInfo, nil
}
