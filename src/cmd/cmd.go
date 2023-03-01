package cmd

import (
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"

	"github.com/chiselstrike/libsql-shell/src/lib"
)

type RootArgs struct {
	statements string
}

func NewRootCmd() *cobra.Command {
	var rootArgs RootArgs = RootArgs{}
	var rootCmd = &cobra.Command{
		SilenceUsage: true,
		Use:          "libsql-shell",
		Short:        "A cli for executing SQL statements on a libSQL or SQLite database",
		Args:         cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := args[0]
			db, err := lib.NewDb(dbPath)
			if err != nil {
				return err
			}
			defer db.Close()

			if cmd.Flag("exec").Changed {
				if len(rootArgs.statements) == 0 {
					return fmt.Errorf("no SQL command to execute")
				}

				results, err := db.ExecuteStatements(rootArgs.statements)
				if err != nil {
					return err
				}
				lib.PrintStatementsResults(results, cmd.OutOrStdout(), false)
				return nil
			}

			shellConfig := lib.ShellConfig{
				InF:         cmd.InOrStdin(),
				OutF:        cmd.OutOrStdout(),
				ErrF:        cmd.ErrOrStderr(),
				HistoryFile: fmt.Sprintf("%s/.libsql_shell_history", os.Getenv("HOME")),
			}

			return db.RunShell(&shellConfig)
		},
	}

	rootCmd.Flags().StringVarP(&rootArgs.statements, "exec", "e", "", "SQL statements separated by ;")

	return rootCmd
}

func Init() {
	var rootCmd *cobra.Command = NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
