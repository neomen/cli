package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpOutput(t *testing.T) {
	cmd := NewCommand("test", "short", "long\nlong line")
	cmd.Example = "example text"
	cmd.Deprecated = "use newcmd instead"
	cmd.Flags().String("flag1", "default", "flag1 usage")
	cmd.PersistentFlags().Bool("pflag", false, "persistent flag")
	sub := NewCommand("sub", "sub short", "")
	sub.GroupID = "group1"
	cmd.AddCommand(sub)
	cmd.CommandGroups = []*Group{{ID: "group1", Title: "Group One"}}

	oldOut := cmd.outWriter
	buf := &bytes.Buffer{}
	cmd.outWriter = buf
	defer func() { cmd.outWriter = oldOut }()

	cmd.Help()

	out := buf.String()
	checks := []string{
		"Usage: test",
		"long\nlong line",
		"Examples:\nexample text",
		"Deprecated: use newcmd instead",
		"Flags:",
		"-flag1=default",
		"flag1 usage",
		"Persistent Flags:",
		"-pflag",
		"persistent flag",
		"Available Commands:",
		"Group One:",
		"sub  sub short", // было "sub\tsub short"
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("help output missing: %q\n%s", check, out)
		}
	}
}

func TestHiddenCommands(t *testing.T) {
	cmd := NewCommand("root", "", "")
	visible := NewCommand("visible", "desc", "")
	hidden := NewCommand("hidden", "", "")
	hidden.Hidden = true
	cmd.AddCommand(visible, hidden)
	vis := cmd.visibleCommands()
	if len(vis) != 1 || vis[0] != visible {
		t.Errorf("visibleCommands returned wrong list: %v", vis)
	}
}

func TestGroupCommands(t *testing.T) {
	cmd := NewCommand("root", "", "")
	cmd.CommandGroups = []*Group{
		{ID: "g1", Title: "Group 1"},
		{ID: "g2", Title: "Group 2"},
	}
	cmd1 := NewCommand("cmd1", "", "")
	cmd1.GroupID = "g2"
	cmd2 := NewCommand("cmd2", "", "")
	cmd2.GroupID = "g1"
	cmd3 := NewCommand("cmd3", "", "")
	cmd3.GroupID = "unknown"
	cmd.AddCommand(cmd1, cmd2, cmd3)
	groups := cmd.groupCommands(cmd.visibleCommands())
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	// Первая группа: Title "Group 1", команда cmd2
	if groups[0].Title != "Group 1" || groups[0].Commands[0].Name() != "cmd2" {
		t.Errorf("group 1 order wrong: got title=%q, cmd=%q", groups[0].Title, groups[0].Commands[0].Name())
	}
	// Вторая группа: Title "Group 2", команда cmd1
	if groups[1].Title != "Group 2" || groups[1].Commands[0].Name() != "cmd1" {
		t.Errorf("group 2 order wrong: got title=%q, cmd=%q", groups[1].Title, groups[1].Commands[0].Name())
	}
	// Третья группа: пустой Title, команда cmd3
	if groups[2].Title != "" || groups[2].Commands[0].Name() != "cmd3" {
		t.Errorf("default group wrong: got title=%q, cmd=%q", groups[2].Title, groups[2].Commands[0].Name())
	}
}
