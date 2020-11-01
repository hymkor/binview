package main

import (
	"context"
	"fmt"
	"github.com/zetamatta/go-readline-ny"
	"io"
)

func getline(out io.Writer, prompt string, defaultStr string) (string, error) {
	editor := readline.Editor{
		Writer:  out,
		Default: defaultStr,
		Cursor:  65535,
		Prompt: func() (int, error) {
			fmt.Fprintf(out, "\r\x1B[0;33;40;1m%s%s", prompt, ERASE_LINE)
			return 2, nil
		},
		LineFeed: func(readline.Result) {},
	}
	defer io.WriteString(out, _ANSI_CURSOR_OFF)
	editor.BindKeySymbol(readline.K_ESCAPE, readline.F_INTR)
	return editor.ReadLine(context.Background())
}
