package watchman

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hansmi/baamhackl/internal/cmdemu"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var fakeCommand = cmdemu.Command{
	Name: "watchman",
	Execute: func(args []string) error {
		fs := flag.NewFlagSet("", flag.PanicOnError)
		_ = fs.Parse(args)

		if fs.NArg() == 1 {
			switch fs.Arg(0) {
			case "success":
				fmt.Println("{}")
				return nil

			case "error":
				fmt.Println(`{"error": "test error"}`)
				return nil

			case "exit-99":
				return cmdemu.ExitCodeError(99)

			case "badjson":
				fmt.Println(`{: [`)
				return nil
			}
		}

		if fs.NArg() > 1 && fs.Arg(0) == "call" {
			fmt.Println("{}")
			return nil
		}

		return errors.New("incorrect usage")
	},
}

func TestMain(m *testing.M) {
	w := cmdemu.New(flag.CommandLine)
	w.Register(fakeCommand)
	os.Exit(w.Main(m))
}

func TestRunClient(t *testing.T) {
	for _, tc := range []struct {
		args    []string
		wantErr *regexp.Regexp
	}{
		{
			wantErr: regexp.MustCompile(`^client failed:.*\bstatus \d+$`),
		},
		{
			args: []string{"success"},
		},
		{
			args:    []string{"error"},
			wantErr: regexp.MustCompile(`\btest error$`),
		},
		{
			args:    []string{"exit-99"},
			wantErr: regexp.MustCompile(`\bexit status 99$`),
		},
		{
			args:    []string{"badjson"},
			wantErr: regexp.MustCompile(`\bdecoding\b.*failed.*\binvalid character\b`),
		},
	} {
		t.Run(fmt.Sprint(tc.args), func(t *testing.T) {
			t.Cleanup(zap.ReplaceGlobals(zaptest.NewLogger(t)))

			args := fakeCommand.MakeArgs(tc.args...)

			if err := runClient(context.Background(), args, nil); tc.wantErr == nil {
				if err != nil {
					t.Errorf("runClient() failed: %v", err)
				}
			} else if err == nil {
				t.Errorf("runClient() succeeded and doesn't match %q", tc.wantErr.String())
			} else if !tc.wantErr.MatchString(err.Error()) {
				t.Errorf("runClient() failed and doesn't match %q: %v", tc.wantErr.String(), err)
			}
		})
	}
}

func TestCommandClient(t *testing.T) {
	t.Cleanup(zap.ReplaceGlobals(zaptest.NewLogger(t)))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := NewCommandClient(fakeCommand.MakeArgs("call"))

	if err := client.WatchSet(ctx, t.TempDir()); err != nil {
		t.Errorf("WatchSet() failed: %v", err)
	}

	if err := client.Recrawl(ctx, t.TempDir()); err != nil {
		t.Errorf("Recrawl() failed: %v", err)
	}

	if err := client.TriggerSet(ctx, t.TempDir(), nil); err != nil {
		t.Errorf("TriggerSet() failed: %v", err)
	}

	if err := client.TriggerDel(ctx, t.TempDir(), "name"); err != nil {
		t.Errorf("TriggerDel() failed: %v", err)
	}
}
