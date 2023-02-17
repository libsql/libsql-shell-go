package lib

import (
	"database/sql"
	"fmt"
	"io"
	"strings"

	"github.com/chzyer/readline"
)

const QUIT_COMMAND = ".quit"
const WELCOME_MESSAGE = "Welcome to LibSQL shell!\n\nType \".quit\" to exit the shell.\n\n"

func NewReadline(in io.Reader, out io.Writer, err io.Writer) (*readline.Instance, error) {
	return readline.NewEx(&readline.Config{
		Prompt:          "→  ",
		InterruptPrompt: "^C",
		EOFPrompt:       QUIT_COMMAND,
		Stdin:           io.NopCloser(in),
		Stdout:          out,
		Stderr:          err,
	})
}

func RunShell(l *readline.Instance, db *sql.DB) error {
	defer l.Close()
	l.CaptureExitSignal()

	fmt.Print(WELCOME_MESSAGE)

shellLoop:
	for {
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		switch line {
		case QUIT_COMMAND:
			break shellLoop
		default:
			result, err := ExecuteStatements(db, line)
			if err != nil {
				fmt.Fprintf(l.Stderr(), "Error: %s\n", err.Error())
			} else {
				fmt.Fprintln(l.Stdout(), result)
			}
		}

	}
	return nil
}
