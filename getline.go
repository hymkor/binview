package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf16"

	"github.com/nyaosorg/go-readline-ny"
)

type Tty interface {
	Raw() (func() error, error)
	ReadRune() (rune, error)
	Buffered() bool
}

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
	text, err := editor.ReadLine(context.Background())
	if err == readline.CtrlC {
		return "", errors.New("Canceled")
	}
	return text, err
}

func getkey(tty1 Tty) (string, error) {
	clean, err := tty1.Raw()
	if err != nil {
		return "", err
	}
	defer clean()

	var buffer strings.Builder
	escape := false
	var surrogated rune = 0
	for {
		r, err := tty1.ReadRune()
		if err != nil {
			return "", err
		}
		if r == 0 {
			continue
		}
		if surrogated > 0 {
			r = utf16.DecodeRune(surrogated, r)
			surrogated = 0
		} else if utf16.IsSurrogate(r) { // surrogate pair first word.
			surrogated = r
			continue
		}
		buffer.WriteRune(r)
		if r == '\x1B' {
			escape = true
		}
		if !(escape && tty1.Buffered()) && buffer.Len() > 0 {
			return buffer.String(), nil
		}
	}
}

func yesNo(tty1 Tty, out io.Writer, message string) bool {
	fmt.Fprintf(out, "%s\r%s%s", _ANSI_YELLOW, message, ERASE_LINE)
	ch, err := getkey(tty1)
	return err == nil && ch == "y"
}
