package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
)

// parseFlags parses persistent flags (from all ancestors) and local flags for the command.
// It populates c.parsedFlags and c.leftoverArgs, then returns an error if any.
func (c *Command) parseFlags(args []string) error {
	if c.DisableFlagParsing {
		return nil
	}
	combined := flag.NewFlagSet(c.Use, flag.ContinueOnError)
	// Add persistent flags from ancestors
	for _, fs := range c.persistentFlagSets() {
		fs.VisitAll(func(f *flag.Flag) {
			if combined.Lookup(f.Name) == nil {
				combined.Var(f.Value, f.Name, f.Usage)
			}
		})
	}
	// Add local flags
	c.localFlags.VisitAll(func(f *flag.Flag) {
		if combined.Lookup(f.Name) == nil {
			combined.Var(f.Value, f.Name, f.Usage)
		}
	})
	if err := combined.Parse(args); err != nil {
		return err
	}
	c.parsedFlags = combined
	c.leftoverArgs = combined.Args()
	return nil
}

// helpRequested checks if the help flag was set in any flag set.
func (c *Command) helpRequested() bool {
	// If we have a parsed combined set, check there first
	if c.parsedFlags != nil {
		if help := c.parsedFlags.Lookup("help"); help != nil && help.Value.String() == "true" {
			return true
		}
		if h := c.parsedFlags.Lookup("h"); h != nil && h.Value.String() == "true" {
			return true
		}
	}
	// Fallback for when parsing hasn't happened
	help := false
	c.localFlags.Visit(func(f *flag.Flag) {
		if f.Name == "help" || f.Name == "h" {
			help = true
		}
	})
	if help {
		return true
	}
	for _, fs := range c.persistentFlagSets() {
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "help" || f.Name == "h" {
				help = true
			}
		})
		if help {
			break
		}
	}
	return help
}

// runHooks executes a list of hook functions in order.
func (c *Command) runHooks(hooks []func(*Command, []string) error, args []string) error {
	for _, hook := range hooks {
		if err := hook(c, args); err != nil {
			return err
		}
	}
	return nil
}

// execute runs the command's Run function after validating arguments and running hooks.
// The command's context is used when calling hooks and Run.
func (c *Command) execute(args []string) error {
	// Check if help was requested.
	if c.helpRequested() {
		c.Help()
		return nil
	}
	if c.Run == nil {
		// No action defined; show help.
		c.Help()
		return nil
	}

	// If args is nil, use the leftover from flag parsing (only if DisableFlagParsing is false)
	if args == nil && !c.DisableFlagParsing {
		args = c.leftoverArgs
	}

	// Validate positional arguments.
	if c.Args != nil {
		if err := c.Args(c, args); err != nil {
			return err
		}
	}

	// Prepare hooks
	persistentPreHooks := []func(*Command, []string) error{}
	for p := c.parent; p != nil; p = p.parent {
		if p.PersistentPreRunE != nil {
			persistentPreHooks = append([]func(*Command, []string) error{p.PersistentPreRunE}, persistentPreHooks...)
		}
	}
	if c.PersistentPreRunE != nil {
		persistentPreHooks = append(persistentPreHooks, c.PersistentPreRunE)
	}
	preHooks := []func(*Command, []string) error{}
	if c.PreRunE != nil {
		preHooks = append(preHooks, c.PreRunE)
	}
	postHooks := []func(*Command, []string) error{}
	if c.PostRunE != nil {
		postHooks = append(postHooks, c.PostRunE)
	}
	persistentPostHooks := []func(*Command, []string) error{}
	if c.PersistentPostRunE != nil {
		persistentPostHooks = append(persistentPostHooks, c.PersistentPostRunE)
	}
	for p := c.parent; p != nil; p = p.parent {
		if p.PersistentPostRunE != nil {
			persistentPostHooks = append(persistentPostHooks, p.PersistentPostRunE)
		}
	}

	// Run hooks
	if err := c.runHooks(persistentPreHooks, args); err != nil {
		return err
	}
	if err := c.runHooks(preHooks, args); err != nil {
		return err
	}
	// Run command
	if err := c.Run(c, args); err != nil {
		return err
	}
	if err := c.runHooks(postHooks, args); err != nil {
		return err
	}
	if err := c.runHooks(persistentPostHooks, args); err != nil {
		return err
	}
	return nil
}

// Execute runs the command with the given arguments (usually os.Args[1:]).
// It uses context.Background().
func (c *Command) Execute() error {
	return c.ExecuteContext(context.Background(), os.Args[1:])
}

// ExecuteContext runs the command with a context and arguments.
// It sets the command's context before execution.
func (c *Command) ExecuteContext(ctx context.Context, args []string) error {
	c.SetContext(ctx)

	// If this command has no children, just run it.
	if len(c.children) == 0 {
		if c.DisableFlagParsing {
			// No flag parsing, pass args directly
			return c.execute(args)
		}
		if err := c.parseFlags(args); err != nil {
			if !c.SilenceErrors {
				fmt.Fprintln(c.errWriter, err)
			}
			if !c.SilenceUsage {
				c.Help()
			}
			return err
		}
		return c.execute(nil)
	}

	// If flag parsing is disabled, run this command directly (no subcommand dispatch)
	if c.DisableFlagParsing {
		return c.execute(args)
	}

	// If no arguments, show help if this command is a group.
	if len(args) == 0 {
		if c.Run == nil {
			c.Help()
			return nil
		}
		return c.execute(args)
	}

	// TraverseChildren: parse flags on this command before looking for subcommand.
	if c.TraverseChildren {
		if err := c.parseFlags(args); err != nil {
			if !c.SilenceErrors {
				fmt.Fprintln(c.errWriter, err)
			}
			if !c.SilenceUsage {
				c.Help()
			}
			return err
		}
		// After parsing, leftover args (non-flag) are the command path.
		args = c.leftoverArgs
		if len(args) == 0 {
			// No subcommand, just run this command if it has Run.
			if c.Run != nil {
				return c.execute(nil)
			}
			c.Help()
			return nil
		}
	}

	// Try to find a subcommand.
	sub := c.Command(args[0])
	if sub != nil {
		return sub.ExecuteContext(ctx, args[1:])
	}

	// Unknown subcommand: maybe it's a flag, or maybe the user intended a command.
	// If we haven't parsed flags yet (because TraverseChildren is false), parse now.
	if !c.TraverseChildren {
		if err := c.parseFlags(args); err != nil {
			if !c.SilenceErrors {
				fmt.Fprintln(c.errWriter, err)
			}
			if !c.SilenceUsage {
				c.Help()
			}
			return err
		}
	}
	// After parsing, leftover arguments (non-flag) are in c.leftoverArgs.
	if len(c.leftoverArgs) == 0 {
		// No leftover arguments, so run this command (if it has a Run function).
		if c.Run == nil {
			c.Help()
			return nil
		}
		return c.execute(nil)
	}
	// There are leftover arguments, but we couldn't find a matching subcommand.
	if !c.DisableSuggestions {
		suggested := c.suggestions(args[0])
		if len(suggested) > 0 {
			return fmt.Errorf("unknown command %q for %q\n\nDid you mean this?\n\t%s",
				args[0], c.Name(), strings.Join(suggested, "\n\t"))
		}
	}
	return fmt.Errorf("unknown command %q for %q", args[0], c.Name())
}

// suggestions returns a list of similar command names for a given unknown command.
func (c *Command) suggestions(unknown string) []string {
	var suggestions []string
	for _, sub := range c.visibleCommands() {
		if levenshtein(sub.Name(), unknown) <= c.SuggestionsMinimumDistance {
			suggestions = append(suggestions, sub.Name())
		}
	}
	// Also check aliases.
	if c.childrenMap != nil {
		for name, sub := range c.childrenMap {
			if name != sub.Use && levenshtein(name, unknown) <= c.SuggestionsMinimumDistance {
				suggestions = append(suggestions, sub.Use)
			}
		}
	}
	// Also check SuggestFor.
	for _, sub := range c.children {
		for _, suggest := range sub.SuggestFor {
			if levenshtein(suggest, unknown) <= c.SuggestionsMinimumDistance {
				suggestions = append(suggestions, sub.Name())
			}
		}
	}
	// Remove duplicates.
	seen := make(map[string]bool)
	uniq := []string{}
	for _, s := range suggestions {
		if !seen[s] {
			seen[s] = true
			uniq = append(uniq, s)
		}
	}
	return uniq
}

// levenshtein distance for suggestions.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	d := make([][]int, la+1)
	for i := range d {
		d[i] = make([]int, lb+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			d[i][j] = min(
				d[i-1][j]+1,
				d[i][j-1]+1,
				d[i-1][j-1]+cost,
			)
		}
	}
	return d[la][lb]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
