package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/nyaosorg/go-readline-ny"
)

type Tty = readline.MinimumTty

func getline(out io.Writer, prompt string, defaultStr string) (string, error) {
	editor := readline.Editor{
		Writer:  out,
		Default: defaultStr,
		Cursor:  65535,
		Prompt: func() (int, error) {
			fmt.Fprintf(out, "\r\x1B[0;33;40;1m%s%s", prompt, _ANSI_ERASE_LINE)
			return 2, nil
		},
		LineFeed: func(readline.Result) {},
	}
	defer io.WriteString(out, _ANSI_CURSOR_OFF)
	editor.BindKeySymbol(readline.K_ESCAPE, readline.F_INTR)
	text, err := editor.ReadLine(context.Background())
	if err == readline.CtrlC {
		return "", errors.New("Canceled")
	}
	return text, err
}

func yesNo(tty1 Tty, out io.Writer, message string) bool {
	fmt.Fprintf(out, "%s\r%s%s", _ANSI_YELLOW, message, _ANSI_ERASE_LINE)
	ch, err := readline.GetKey(tty1)
	return err == nil && ch == "y"
}
