package cmd

import (
	"fmt"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"

	"github.com/libsql/libsql-shell-go/pkg/shell"
	"github.com/libsql/libsql-shell-go/pkg/shell/enums"
)

type RootArgs struct {
	statements string
	quiet      bool
	authToken  string
	file       string
}

func NewRootCmd() *cobra.Command {
	var rootArgs RootArgs = RootArgs{}
	var rootCmd = &cobra.Command{
		SilenceUsage: true,
		Use:          "libsql-shell <DB>",
		Short:        "A cli for executing SQL statements on a libSQL or SQLite database",
		Args:         cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			shellConfig := shell.ShellConfig{
				DbUri:       args[0],
				InF:         cmd.InOrStdin(),
				OutF:        cmd.OutOrStdout(),
				ErrF:        cmd.ErrOrStderr(),
				HistoryMode: enums.PerDatabaseHistory,
				HistoryName: "libsql",
				QuietMode:   rootArgs.quiet,
				AuthToken:   rootArgs.authToken,
			}

			if cmd.Flag("exec").Changed {
				if len(rootArgs.statements) == 0 {
					return fmt.Errorf("no SQL command to execute")
				}

				return shell.RunShellLine(shellConfig, rootArgs.statements)
			}

			if cmd.Flag("from-file").Changed {
				if len(rootArgs.file) == 0 {
					return fmt.Errorf("file not provided")
				}

				file, err := os.Open(rootArgs.file)
				if err != nil {
					return err
				}
				defer file.Close()

				stat, err := file.Stat()
				if err != nil {
					return err
				}

				content := make([]byte, stat.Size())
				_, err = file.Read(content)
				if err != nil {
					return err
				}

				return shell.RunShellLine(shellConfig, strings.TrimSpace(string(content)))

			}

			return shell.RunShell(shellConfig)
		},
	}

	rootCmd.Flags().StringVarP(&rootArgs.statements, "exec", "e", "", "SQL statements separated by ;")
	rootCmd.Flags().StringVarP(&rootArgs.file, "from-file", "f", "", "Execute commands from a file")
	rootCmd.Flags().BoolVarP(&rootArgs.quiet, "quiet", "q", false, "Don't print welcome message")
	rootCmd.Flags().StringVar(&rootArgs.authToken, "auth", "", "Add a JWT Token.")

	return rootCmd
}

func Execute() {
	var rootCmd *cobra.Command = NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
