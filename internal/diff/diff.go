package diff

import "github.com/ksaegusa/ConfdiffStudio/internal/model"

func Compute(before, after []string) model.Diff {
	beforeCount := make(map[string]int)
	afterCount := make(map[string]int)

	for _, line := range before {
		beforeCount[line]++
	}
	for _, line := range after {
		afterCount[line]++
	}

	result := model.Diff{}

	for line, bcount := range beforeCount {
		acount := afterCount[line]
		for i := 0; i < bcount-acount; i++ {
			result.Removed = append(result.Removed, line)
		}
	}

	for line, acount := range afterCount {
		bcount := beforeCount[line]
		for i := 0; i < acount-bcount; i++ {
			result.Added = append(result.Added, line)
		}
	}

	return result
}
