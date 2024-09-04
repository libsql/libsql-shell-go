package db

import (
	"net/url"
	"reflect"
	"strings"
	"unicode"
)

func IsUrl(uri string) bool {
	url, err := url.ParseRequestURI(uri)
	if err != nil {
		return false
	}
	return url.Scheme != ""
}

func IsValidSqldUrl(uri string) (bool, string) {
	url, err := url.ParseRequestURI(uri)
	if err != nil {
		return false, ""
	}
	return url.Scheme == "libsql" || url.Scheme == "wss" || url.Scheme == "ws" || url.Scheme == "http" || url.Scheme == "https", url.Scheme
}

func EscapeSingleQuotes(value string) string {
	return strings.Replace(value, "'", "''", -1)
}

func startsWithNumber(name string) bool {
	firstChar := rune(name[0])
	return unicode.IsNumber(firstChar)
}

func NeedsEscaping(name string) bool {
	if len(name) == 0 {
		return true
	}
	if startsWithNumber(name) {
		return true
	}
	for _, char := range name {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) && char != rune('_') {
			return true
		}
	}
	return false
}

var explainQueryPlanStatement = "EXPLAIN QUERY PLAN"
var explainQueryPlanColumnNames = []string{"id", "parent", "notused", "detail"}

func queryContainsExplainQueryPlanStatement(query string) bool {
	return strings.HasPrefix(
		strings.ToLower(query),
		strings.ToLower(explainQueryPlanStatement),
	)
}

func columnNamesMatchExplainQueryPlan(colNames []string) bool {
	return reflect.DeepEqual(colNames, explainQueryPlanColumnNames)
}

// "query" can be a string containing multiple queries separated by ";" or a single query
func IsResultComingFromExplainQueryPlan(statementResult StatementResult) bool {
	query := statementResult.Query
	columnNames := statementResult.ColumnNames
	return queryContainsExplainQueryPlanStatement(query) &&
		columnNamesMatchExplainQueryPlan(columnNames)
}
