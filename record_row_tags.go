package mwanachamaagency

import (
	"encoding/json"
	"slices"
	"strings"

	"gorm.io/gorm"
)

const AllRowsTag = "all"

func cleanTags(tags []string) ([]string, error) {
	out := []string{}
	for _, tag := range tags {
		if strings.TrimSpace(tag) == "" {
			return nil, invalidRow("a tag is not blank")
		}
		if !slices.Contains(out, tag) {
			out = append(out, tag)
		}
	}
	return out, nil
}

func reachesEveryRow(tags []string) bool {
	return len(tags) == 0 || slices.Contains(tags, AllRowsTag)
}

func tagFilter(q *gorm.DB, tags []string) *gorm.DB {
	if reachesEveryRow(tags) {
		return q
	}
	clauses := make([]string, 0, len(tags))
	needles := make([]any, 0, len(tags))
	for _, tag := range tags {
		quoted, _ := json.Marshal(tag)
		clauses = append(clauses, "tags LIKE ? ESCAPE '\\'")
		needles = append(needles, "%"+likeEscaper.Replace(string(quoted))+"%")
	}
	return q.Where("("+strings.Join(clauses, " OR ")+")", needles...)
}

func carriesAny(rowTags, tags []string) bool {
	for _, tag := range rowTags {
		if slices.Contains(tags, tag) {
			return true
		}
	}
	return false
}
