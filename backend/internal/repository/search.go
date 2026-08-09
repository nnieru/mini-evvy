package repository

import (
	"fmt"
	"strings"
)

func appendILikeOr(clause string, fields []string, q string, args []any) (string, []any) {
	if strings.TrimSpace(q) == "" {
		return clause, args
	}
	pos := len(args) + 1
	parts := make([]string, len(fields))
	for i, field := range fields {
		parts[i] = fmt.Sprintf("%s ILIKE $%d", field, pos)
	}
	return clause + " AND (" + strings.Join(parts, " OR ") + ")", append(args, SearchPattern(q))
}
