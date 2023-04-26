package main_test

import (
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/libsql/libsql-shell-go/internal/cmd"
	"github.com/libsql/libsql-shell-go/test/utils"
)

func TestRootCommandFlags_WhenAllFlagsAreProvided_ExpectSQLStatementsExecutedWithoutError(t *testing.T) {
	c := qt.New(t)

	dbPath := c.TempDir() + `\test.sqlite`
	rootCmd := cmd.NewRootCmd()

	_, _, err := utils.ExecuteCobraCommand(t, rootCmd, "--exec", "CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT);", dbPath)

	c.Assert(err, qt.IsNil)
}

func TestRootCommandFlags_WhenDbIsMissing_ExpectErrorReturned(t *testing.T) {
	c := qt.New(t)

	rootCmd := cmd.NewRootCmd()

	_, _, err := utils.ExecuteCobraCommand(t, rootCmd, "--exec", "CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT);")

	c.Assert(err.Error(), qt.Equals, `accepts 1 arg(s), received 0`)
}

func TestRootCommandFlags_GivenEmptyStatements_ExpectErrorReturned(t *testing.T) {
	c := qt.New(t)

	dbPath := c.TempDir() + `\test.sqlite`
	rootCmd := cmd.NewRootCmd()

	_, _, err := utils.ExecuteCobraCommand(t, rootCmd, "--exec", "", dbPath)

	c.Assert(err.Error(), qt.IsNotNil)
}
