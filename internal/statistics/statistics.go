package statistics

import (
	"github.com/tasticolly/gitfame/internal/git"
)

type EntityFrame struct {
	PersonName   string `json:"name"`
	LinesCount   int    `json:"lines"`
	CommitsCount int    `json:"commits"`
	FilesCount   int    `json:"files"`
}

type EntityFrameWriter interface {
	WriteFrames(data []EntityFrame) error
}

type internalStats struct {
	FilesCount int
	LinesCount int
	CommitSet  map[string]struct{}
}

func newInternalStats() *internalStats {
	return &internalStats{CommitSet: make(map[string]struct{})}
}

func CalculateStatistic(repository, revision string, fileNames []string, useCommitter bool) ([]EntityFrame, error) {
	PersonToStats := make(map[string]*internalStats)

	for _, filename := range fileNames {
		commitsInfoFromFile, err := git.GetInfo(repository, revision, filename, useCommitter)
		if err != nil {
			return nil, err
		}

		currentPersons := make(map[string]struct{})
		for commitHash, currentCommitInfo := range commitsInfoFromFile {
			currentPersons[currentCommitInfo.Person] = struct{}{}

			stats, ok := PersonToStats[currentCommitInfo.Person]
			if !ok {
				stats = newInternalStats()
				PersonToStats[currentCommitInfo.Person] = stats
			}
			stats.CommitSet[commitHash] = struct{}{}
			stats.LinesCount += currentCommitInfo.NumOfLines
		}
		for person := range currentPersons {
			PersonToStats[person].FilesCount++
		}
	}

	result := make([]EntityFrame, 0, len(PersonToStats))

	for person, internalStatPtr := range PersonToStats {
		result = append(result, EntityFrame{
			PersonName:   person,
			LinesCount:   internalStatPtr.LinesCount,
			CommitsCount: len(internalStatPtr.CommitSet),
			FilesCount:   internalStatPtr.FilesCount,
		})
	}
	return result, nil
}
