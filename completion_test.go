package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestAddCompletionCommand(t *testing.T) {
	root := NewCommand("root", "", "")
	// root already has completion commands from NewCommand
	if root.Command("__complete") == nil {
		t.Error("__complete command not added")
	}
	if root.Command("__completeNoDesc") == nil {
		t.Error("__completeNoDesc command not added")
	}
}

func TestCompleteRoot(t *testing.T) {
	root := NewCommand("root", "", "")
	sub1 := NewCommand("serve", "start server", "")
	sub2 := NewCommand("build", "build project", "")
	root.AddCommand(sub1, sub2)

	oldOut := root.outWriter
	buf := &bytes.Buffer{}
	root.outWriter = buf
	defer func() { root.outWriter = oldOut }()

	err := root.complete([]string{}, true)
	if err != nil {
		t.Fatalf("complete error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "serve\tstart server") {
		t.Errorf("missing serve: %s", output)
	}
	if !strings.Contains(output, "build\tbuild project") {
		t.Errorf("missing build: %s", output)
	}
}

func TestCompleteWithPartialPath(t *testing.T) {
	root := NewCommand("root", "", "")
	sub := NewCommand("serve", "", "")
	root.AddCommand(sub)

	buf := &bytes.Buffer{}
	oldOut := root.outWriter
	root.outWriter = buf
	defer func() { root.outWriter = oldOut }()

	// complete with path ["se"] – should suggest "serve"
	err := root.complete([]string{"se"}, true)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if !strings.Contains(buf.String(), "serve") {
		t.Errorf("expected serve suggestion, got %q", buf.String())
	}
}

func TestCompleteFlags(t *testing.T) {
	cmd := NewCommand("test", "", "")
	cmd.Flags().String("config", "", "config file")
	cmd.PersistentFlags().Bool("verbose", false, "verbose mode")

	buf := &bytes.Buffer{}
	oldOut := cmd.outWriter
	cmd.outWriter = buf
	defer func() { cmd.outWriter = oldOut }()

	err := cmd.complete([]string{"", "--"}, true)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "--config") || !strings.Contains(out, "config file") {
		t.Errorf("flag completion missing: %s", out)
	}
	if !strings.Contains(out, "--verbose") || !strings.Contains(out, "verbose mode") {
		t.Errorf("persistent flag completion missing: %s", out)
	}
}

func TestCompleteValidArgs(t *testing.T) {
	cmd := NewCommand("test", "", "")
	cmd.ValidArgs = []string{"start", "stop", "restart"}
	cmd.Run = func(c *Command, args []string) error { return nil }

	buf := &bytes.Buffer{}
	oldOut := cmd.outWriter
	cmd.outWriter = buf
	defer func() { cmd.outWriter = oldOut }()

	err := cmd.complete([]string{""}, false)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	out := buf.String()
	for _, arg := range cmd.ValidArgs {
		if !strings.Contains(out, arg) {
			t.Errorf("valid arg %q missing from completion", arg)
		}
	}
}

func TestGenBashCompletion(t *testing.T) {
	root := NewCommand("myapp", "", "")
	buf := &bytes.Buffer{}
	err := root.GenBashCompletion(buf)
	if err != nil {
		t.Fatalf("GenBashCompletion error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "complete -F __myapp_completion myapp") {
		t.Errorf("bash completion script incomplete: %s", out)
	}
}

func TestGenZshCompletion(t *testing.T) {
	root := NewCommand("myapp", "", "")
	buf := &bytes.Buffer{}
	err := root.GenZshCompletion(buf)
	if err != nil {
		t.Fatalf("GenZshCompletion error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "compdef _myapp myapp") {
		t.Errorf("zsh completion script incomplete: %s", out)
	}
}

func TestGenFishCompletion(t *testing.T) {
	root := NewCommand("myapp", "", "")
	buf := &bytes.Buffer{}
	err := root.GenFishCompletion(buf)
	if err != nil {
		t.Fatalf("GenFishCompletion error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "complete -c myapp -f -a \"(__myapp_completion)\"") {
		t.Errorf("fish completion script incomplete: %s", out)
	}
}

func TestGenPowerShellCompletion(t *testing.T) {
	root := NewCommand("myapp", "", "")
	buf := &bytes.Buffer{}
	err := root.GenPowerShellCompletion(buf)
	if err != nil {
		t.Fatalf("GenPowerShellCompletion error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Register-ArgumentCompleter -Native -CommandName myapp") {
		t.Errorf("powershell completion script incomplete: %s", out)
	}
}
