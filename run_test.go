package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestExecuteSubcommand(t *testing.T) {
	root := NewCommand("root", "", "")
	var subCalled bool
	sub := NewCommand("sub", "", "")
	sub.Run = func(c *Command, args []string) error {
		subCalled = true
		return nil
	}
	root.AddCommand(sub)

	err := root.ExecuteContext(context.Background(), []string{"sub"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !subCalled {
		t.Error("subcommand Run not called")
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	root := NewCommand("root", "", "")
	root.Run = func(c *Command, args []string) error { return nil }
	root.DisableSuggestions = true

	err := root.ExecuteContext(context.Background(), []string{"unknown"})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("expected unknown command error, got %v", err)
	}
}

func TestExecuteSuggestions(t *testing.T) {
	root := NewCommand("root", "", "")
	sub := NewCommand("serve", "", "")
	root.AddCommand(sub)
	root.SuggestionsMinimumDistance = 2

	err := root.ExecuteContext(context.Background(), []string{"server"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Did you mean") || !strings.Contains(err.Error(), "serve") {
		t.Errorf("suggestion missing: %v", err)
	}
}

func TestExecuteTraverseChildren(t *testing.T) {
	root := NewCommand("root", "", "")
	var flagParsed bool
	root.Flags().Bool("global", false, "")
	root.TraverseChildren = true
	sub := NewCommand("sub", "", "")
	sub.Run = func(c *Command, args []string) error {
		// Проверяем флаг через родителя
		globalFlag := c.Parent().Flags().Lookup("global")
		if globalFlag == nil || globalFlag.Value.String() != "true" {
			t.Error("global flag not parsed")
		}
		flagParsed = true
		return nil
	}
	root.AddCommand(sub)

	err := root.ExecuteContext(context.Background(), []string{"--global", "sub"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !flagParsed {
		t.Error("subcommand not executed")
	}
}

func TestExecuteDisableFlagParsing(t *testing.T) {
	root := NewCommand("root", "", "")
	root.children = []*Command{}
	root.childrenMap = nil

	var args []string
	root.Run = func(c *Command, a []string) error {
		args = a
		return nil
	}
	root.DisableFlagParsing = true

	err := root.ExecuteContext(context.Background(), []string{"--foo", "bar"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(args) != 2 || args[0] != "--foo" || args[1] != "bar" {
		t.Errorf("expected args [--foo bar], got %v", args)
	}
}

func TestHooks(t *testing.T) {
	var order []string
	root := NewCommand("root", "", "")
	root.PersistentPreRunE = func(c *Command, args []string) error {
		order = append(order, "rootPersistentPre")
		return nil
	}
	root.PersistentPostRunE = func(c *Command, args []string) error {
		order = append(order, "rootPersistentPost")
		return nil
	}
	sub := NewCommand("sub", "", "")
	sub.PreRunE = func(c *Command, args []string) error {
		order = append(order, "subPre")
		return nil
	}
	sub.PostRunE = func(c *Command, args []string) error {
		order = append(order, "subPost")
		return nil
	}
	sub.Run = func(c *Command, args []string) error {
		order = append(order, "subRun")
		return nil
	}
	root.AddCommand(sub)

	err := root.ExecuteContext(context.Background(), []string{"sub"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expected := []string{"rootPersistentPre", "subPre", "subRun", "subPost", "rootPersistentPost"}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i := range order {
		if order[i] != expected[i] {
			t.Errorf("hook order mismatch at %d: expected %q, got %q", i, expected[i], order[i])
		}
	}
}

func TestHooksError(t *testing.T) {
	root := NewCommand("root", "", "")
	root.PersistentPreRunE = func(c *Command, args []string) error {
		return errors.New("hook error")
	}
	sub := NewCommand("sub", "", "")
	sub.Run = func(c *Command, args []string) error { return nil }
	root.AddCommand(sub)

	err := root.ExecuteContext(context.Background(), []string{"sub"})
	if err == nil || err.Error() != "hook error" {
		t.Errorf("expected hook error, got %v", err)
	}
}

func TestSilenceErrors(t *testing.T) {
	cmd := NewCommand("test", "", "")
	cmd.Flags().Bool("foo", false, "")
	cmd.Run = func(c *Command, args []string) error { return errors.New("runtime error") }
	cmd.SilenceErrors = true
	oldErr := cmd.errWriter
	buf := &bytes.Buffer{}
	cmd.errWriter = buf
	defer func() { cmd.errWriter = oldErr }()

	err := cmd.ExecuteContext(context.Background(), []string{})
	if err == nil {
		t.Error("expected error")
	}
	if buf.Len() > 0 {
		t.Errorf("expected no error output, got %q", buf.String())
	}
}
