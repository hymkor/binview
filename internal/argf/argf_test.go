package argf

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"testing"
)

func TestArgf(t *testing.T) {
	args := []string{}
	for i := 1; i <= 5; i++ {
		fname := filepath.Join(os.TempDir(), fmt.Sprintf("%d.txt", i))
		writer, err := os.Create(fname)
		if err != nil {
			t.Fatalf("%s: %s", fname, err.Error())
		}
		args = append(args, fname)
		fmt.Fprintf(writer, "%d\n", i)
		writer.Close()
	}
	defer func() {
		for _, fname := range args {
			os.Remove(fname)
		}
	}()

	reader, err := New(args)
	if err != nil {
		t.Fatal(err.Error())
	}
	sc := bufio.NewScanner(reader)
	i := 1
	for sc.Scan() {
		line := sc.Text()
		expect := fmt.Sprintf("%d", i)
		if line != expect {
			t.Fatalf("'%s' != '%s'", line, expect)
		}
		i++
	}
	if i != 6 {
		t.Fatal("not read all data")
	}
}
