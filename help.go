package cli

import (
	"flag"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
)

// Help prints help information for the command.
func (c *Command) Help() {
	out := c.outWriter
	fmt.Fprintf(out, "Usage: %s\n", c.usageLine())
	if c.Long != "" {
		fmt.Fprintf(out, "\n%s\n", c.Long)
	} else if c.Short != "" {
		fmt.Fprintf(out, "\n%s\n", c.Short)
	}
	if c.Example != "" {
		fmt.Fprintf(out, "\nExamples:\n%s\n", c.Example)
	}
	if c.Deprecated != "" {
		fmt.Fprintf(out, "\nDeprecated: %s\n", c.Deprecated)
	}
	// Flags: only show sections if there are any flags defined (not just help flags)
	if c.hasFlags(c.localFlags) {
		fmt.Fprintf(out, "\nFlags:\n")
		c.printFlags(out, c.localFlags)
	}
	if c.hasFlags(c.persistentFlags) {
		fmt.Fprintf(out, "\nPersistent Flags:\n")
		c.printFlags(out, c.persistentFlags)
	}
	// Subcommands grouped
	visible := c.visibleCommands()
	if len(visible) > 0 {
		fmt.Fprintf(out, "\nAvailable Commands:\n")
		groups := c.groupCommands(visible)
		w := tabwriter.NewWriter(out, 0, 8, 2, ' ', 0)
		for _, group := range groups {
			if group.Title != "" {
				fmt.Fprintf(w, "  %s:\n", group.Title)
			}
			for _, cmd := range group.Commands {
				line := fmt.Sprintf("    %s\t%s", cmd.Name(), cmd.Short)
				if cmd.Deprecated != "" {
					line += " (deprecated)"
				}
				fmt.Fprintln(w, line)
			}
		}
		w.Flush()
	}
}

func (c *Command) usageLine() string {
	if c.Use == "" {
		return c.Name()
	}
	return c.Use
}

func (c *Command) printFlags(w io.Writer, fs *flag.FlagSet) {
	fs.VisitAll(func(f *flag.Flag) {
		// skip help flags to avoid duplication
		if f.Name == "help" || f.Name == "h" {
			return
		}
		fmt.Fprintf(w, "  -%s", f.Name)
		if f.DefValue != "" {
			fmt.Fprintf(w, "=%s", f.DefValue)
		}
		fmt.Fprintf(w, "\n    \t%s\n", f.Usage)
	})
}

// hasFlags returns true if the flag set contains any flags other than help.
func (c *Command) hasFlags(fs *flag.FlagSet) bool {
	has := false
	fs.VisitAll(func(f *flag.Flag) {
		if f.Name != "help" && f.Name != "h" {
			has = true
		}
	})
	return has
}

type groupedCommands struct {
	Title    string
	Commands []*Command
}

func (c *Command) groupCommands(cmds []*Command) []groupedCommands {
	groups := make(map[string][]*Command)
	groupTitles := make(map[string]string)
	// Add groups from CommandGroups (parent's groups)
	for _, g := range c.CommandGroups {
		groupTitles[g.ID] = g.Title
	}
	// Collect commands by GroupID
	for _, cmd := range cmds {
		gid := cmd.GroupID
		if gid == "" {
			gid = "_default"
			groupTitles["_default"] = ""
		}
		groups[gid] = append(groups[gid], cmd)
	}

	// Collect group IDs
	var groupIDs []string
	for id := range groups {
		groupIDs = append(groupIDs, id)
	}

	// Sort groups: first those with non-empty title, then those with empty title (excluding _default), then _default
	sort.Slice(groupIDs, func(i, j int) bool {
		id1, id2 := groupIDs[i], groupIDs[j]

		// _default always last
		if id1 == "_default" {
			return false
		}
		if id2 == "_default" {
			return true
		}

		title1 := groupTitles[id1]
		title2 := groupTitles[id2]

		// Non-empty titles come before empty titles
		if title1 != "" && title2 == "" {
			return true
		}
		if title1 == "" && title2 != "" {
			return false
		}

		// Both have titles or both have empty titles – sort by title (or ID for empty titles)
		if title1 != title2 {
			return title1 < title2
		}
		return id1 < id2
	})

	result := make([]groupedCommands, 0, len(groups))
	for _, id := range groupIDs {
		cmds := groups[id]
		// sort commands by name
		sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name() < cmds[j].Name() })
		result = append(result, groupedCommands{
			Title:    groupTitles[id],
			Commands: cmds,
		})
	}
	return result
}

// visibleCommands returns subcommands that are not hidden.
func (c *Command) visibleCommands() []*Command {
	var vis []*Command
	for _, sub := range c.children {
		if !sub.Hidden {
			vis = append(vis, sub)
		}
	}
	return vis
}
