package utils

import (
	"bytes"
	"strings"
	"testing"

	"github.com/libsql/libsql-shell-go/internal/db"
	"github.com/spf13/cobra"
)

func ExecuteCobraCommand(t *testing.T, c *cobra.Command, args ...string) (string, string, error) {
	return ExecuteCobraCommandWithInitialInput(t, c, "", args...)
}

func ExecuteCobraCommandWithInitialInput(t *testing.T, c *cobra.Command, initialInput string, args ...string) (string, string, error) {
	t.Helper()

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	bufIn := new(bytes.Buffer)

	_, err := bufIn.Write([]byte(initialInput))
	if err != nil {
		return "", "", err
	}

	c.SetOut(bufOut)
	c.SetErr(bufErr)
	c.SetIn(bufIn)
	c.SetArgs(args)

	err = c.Execute()
	return strings.TrimSpace(bufOut.String()), strings.TrimSpace(bufErr.String()), err
}

func GetPrintTableOutput(header []string, data [][]string) string {
	buf := new(bytes.Buffer)

	db.PrintTable(buf, header, data)

	return strings.TrimSpace(buf.String())
}
