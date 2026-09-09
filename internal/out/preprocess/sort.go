package preprocess

import (
	"github.com/tasticolly/gitfame/internal/statistics"
	"sort"
	"strings"
)

func less(params1 [3]int, person1 string, params2 [3]int, person2 string) bool {
	for i := 0; i < len(params1); i++ {
		if params1[i] != params2[i] {
			return params1[i] > params2[i]
		}
	}
	return strings.Compare(person1, person2) < 0
}

func Sort(data []statistics.EntityFrame, orderBy string) {
	lessEntity := func(i, j int) bool {
		switch orderBy {

		case "lines":
			return less(
				[3]int{data[i].LinesCount, data[i].CommitsCount, data[i].FilesCount}, data[i].PersonName,
				[3]int{data[j].LinesCount, data[j].CommitsCount, data[j].FilesCount}, data[j].PersonName,
			)

		case "commits":
			return less(
				[3]int{data[i].CommitsCount, data[i].LinesCount, data[i].FilesCount}, data[i].PersonName,
				[3]int{data[j].CommitsCount, data[j].LinesCount, data[j].FilesCount}, data[j].PersonName,
			)
		case "files":
			return less(
				[3]int{data[i].FilesCount, data[i].LinesCount, data[i].CommitsCount}, data[i].PersonName,
				[3]int{data[j].FilesCount, data[j].LinesCount, data[j].CommitsCount}, data[j].PersonName,
			)
		default:
			panic("orderBy do not validate: " + orderBy)
		}
	}

	sort.Slice(data, lessEntity)
}
