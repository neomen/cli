package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestNewCommand(t *testing.T) {
	cmd := NewCommand("test", "short", "long")
	if cmd.Name() != "test" {
		t.Errorf("expected name 'test', got %q", cmd.Name())
	}
	if cmd.Short != "short" {
		t.Errorf("expected short 'short', got %q", cmd.Short)
	}
	if cmd.Long != "long" {
		t.Errorf("expected long 'long', got %q", cmd.Long)
	}
	if cmd.parent != nil {
		t.Error("root command should have nil parent")
	}
	if cmd.localFlags == nil {
		t.Error("localFlags should be initialized")
	}
	if cmd.persistentFlags == nil {
		t.Error("persistentFlags should be initialized")
	}
	helpFlag := cmd.localFlags.Lookup("help")
	if helpFlag == nil {
		t.Error("help flag not added to localFlags")
	}
	helpFlagPersistent := cmd.persistentFlags.Lookup("help")
	if helpFlagPersistent == nil {
		t.Error("help flag not added to persistentFlags")
	}
	if len(cmd.children) != 2 {
		t.Errorf("expected 2 completion subcommands, got %d", len(cmd.children))
	}
}

func TestAddCommand(t *testing.T) {
	root := NewCommand("root", "", "")
	child := NewCommand("child", "", "")
	root.AddCommand(child)
	if len(root.children) != 3 { // +2 completion
		t.Errorf("expected 3 children, got %d", len(root.children))
	}
	if child.parent != root {
		t.Error("child parent not set")
	}
	if root.Command("child") != child {
		t.Error("command lookup by name failed")
	}
	child.Aliases = []string{"c"}
	root.AddCommand(child) // adding again should not duplicate but update map
	if root.Command("c") != child {
		t.Error("alias lookup failed")
	}
}

func TestCommandName(t *testing.T) {
	cmd := NewCommand("test cmd", "", "")
	if cmd.Name() != "test" {
		t.Errorf("expected 'test', got %q", cmd.Name())
	}
	cmd.Use = "single"
	if cmd.Name() != "single" {
		t.Errorf("expected 'single', got %q", cmd.Name())
	}
}

func TestFlagsAndPersistentFlags(t *testing.T) {
	cmd := NewCommand("test", "", "")
	cmd.Flags().String("foo", "", "foo flag")
	cmd.PersistentFlags().Int("bar", 0, "bar flag")
	if cmd.Flags().Lookup("foo") == nil {
		t.Error("local flag not found")
	}
	if cmd.PersistentFlags().Lookup("bar") == nil {
		t.Error("persistent flag not found")
	}
}

func TestExecuteSimpleCommand(t *testing.T) {
	var called bool
	cmd := NewCommand("test", "", "")
	cmd.Run = func(c *Command, args []string) error {
		called = true
		return nil
	}
	oldOut := cmd.outWriter
	defer func() { cmd.outWriter = oldOut }()
	cmd.outWriter = &bytes.Buffer{}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"test"}

	err := cmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("Run not called")
	}
}

func TestExecuteWithFlags(t *testing.T) {
	var foo string
	cmd := NewCommand("test", "", "")
	cmd.Flags().StringVar(&foo, "foo", "", "foo")
	cmd.Run = func(c *Command, args []string) error {
		if foo != "bar" {
			t.Errorf("expected foo=bar, got %q", foo)
		}
		return nil
	}
	oldOut := cmd.outWriter
	defer func() { cmd.outWriter = oldOut }()
	cmd.outWriter = &bytes.Buffer{}

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"test", "--foo=bar"}

	err := cmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExecuteHelpFlag(t *testing.T) {
	cmd := NewCommand("test", "short desc", "")
	oldOut := cmd.outWriter
	buf := &bytes.Buffer{}
	cmd.outWriter = buf
	defer func() { cmd.outWriter = oldOut }()

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"test", "--help"}

	err := cmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "Usage: test") {
		t.Errorf("help output missing: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "short desc") {
		t.Errorf("short description missing")
	}
}

func TestArgsValidation(t *testing.T) {
	cmd := NewCommand("test", "", "")
	// Clear the hidden completion commands so the command is a leaf.
	cmd.children = []*Command{}
	cmd.childrenMap = nil

	cmd.Args = func(c *Command, args []string) error {
		if len(args) != 1 {
			return errors.New("need exactly one arg")
		}
		return nil
	}
	cmd.Run = func(c *Command, args []string) error { return nil }
	oldOut := cmd.outWriter
	defer func() { cmd.outWriter = oldOut }()
	cmd.outWriter = &bytes.Buffer{}

	err := cmd.ExecuteContext(context.Background(), []string{"foo"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err = cmd.ExecuteContext(context.Background(), []string{})
	if err == nil || err.Error() != "need exactly one arg" {
		t.Errorf("expected validation error, got %v", err)
	}
}
