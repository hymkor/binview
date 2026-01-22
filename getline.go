package bine

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/nyaosorg/go-readline-ny"
	"github.com/nyaosorg/go-readline-ny/keys"

	"github.com/nyaosorg/go-ttyadapter"
)

type Tty = ttyadapter.Tty

func getline(out io.Writer, prompt string, defaultStr string, history readline.IHistory) (string, error) {
	editor := readline.Editor{
		Writer:  out,
		Default: defaultStr,
		Cursor:  65535,
		PromptWriter: func(w io.Writer) (int, error) {
			fmt.Fprintf(w, "\r\x1B[0;33;40;1m%s%s", prompt, _ANSI_ERASE_LINE)
			return 2, nil
		},
		LineFeedWriter: func(readline.Result, io.Writer) (int, error) { return 0, nil },
		History:        history,
	}
	defer io.WriteString(out, _ANSI_CURSOR_OFF)
	editor.BindKey(keys.Escape, readline.CmdInterrupt)
	text, err := editor.ReadLine(context.Background())
	if err == readline.CtrlC {
		return "", errors.New("Canceled")
	}
	return text, err
}

func yesNo(tty1 Tty, out io.Writer, message string) bool {
	fmt.Fprintf(out, "%s\r%s%s", _ANSI_YELLOW, message, _ANSI_ERASE_LINE)
	ch, err := tty1.GetKey()
	if err == nil && (ch == "y" || ch == "Y") {
		fmt.Fprintf(out, " %s ", ch)
		return true
	}
	return false
}
