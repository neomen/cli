package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

// AddCompletionCommand adds the __complete and __completeNoDesc subcommands to the root command.
// This is automatically called for root commands, but can be called manually if needed.
func (c *Command) AddCompletionCommand() {
	if c.parent != nil {
		// Only root commands should have completion commands.
		return
	}
	// __complete command
	completeCmd := &Command{
		Use:    "__complete",
		Hidden: true,
		Run: func(cmd *Command, args []string) error {
			return c.complete(args, true)
		},
	}
	// __completeNoDesc command
	completeNoDescCmd := &Command{
		Use:    "__completeNoDesc",
		Hidden: true,
		Run: func(cmd *Command, args []string) error {
			return c.complete(args, false)
		},
	}
	c.AddCommand(completeCmd, completeNoDescCmd)
}

// complete handles the completion logic for __complete and __completeNoDesc.
// It expects args to contain the command line arguments that are being completed,
// with the last argument being the current word being completed.
// The output should be one completion per line, optionally with a description separated by tab.
func (c *Command) complete(args []string, withDesc bool) error {
	// If there are no arguments, we are completing at the root level.
	if len(args) == 0 {
		for _, sub := range c.visibleCommands() {
			if withDesc {
				fmt.Fprintf(c.outWriter, "%s\t%s\n", sub.Name(), sub.Short)
			} else {
				fmt.Fprintln(c.outWriter, sub.Name())
			}
		}
		return nil
	}

	// The last argument is the current word being completed.
	currentWord := args[len(args)-1]
	// The preceding arguments are the command path (possibly incomplete).
	path := args[:len(args)-1]

	// Traverse to the command where completion is happening.
	cmd := c
	for i, arg := range path {
		sub := cmd.Command(arg)
		if sub == nil {
			// No such command, so maybe it's a flag or the user is typing a command.
			// If we are at the last part of the path, we can suggest commands that match the prefix.
			if i == len(path)-1 {
				prefix := arg
				for _, sub := range cmd.visibleCommands() {
					if strings.HasPrefix(sub.Name(), prefix) {
						if withDesc {
							fmt.Fprintf(c.outWriter, "%s\t%s\n", sub.Name(), sub.Short)
						} else {
							fmt.Fprintln(c.outWriter, sub.Name())
						}
					}
				}
				// Also suggest flags
				suggestFlags(cmd, prefix, withDesc)
			}
			return nil
		}
		cmd = sub
	}

	// At this point, we are at the command that is being completed.
	if currentWord == "" {
		// Suggest subcommands
		for _, sub := range cmd.visibleCommands() {
			if withDesc {
				fmt.Fprintf(c.outWriter, "%s\t%s\n", sub.Name(), sub.Short)
			} else {
				fmt.Fprintln(c.outWriter, sub.Name())
			}
		}
		// Suggest valid positional arguments
		if len(cmd.ValidArgs) > 0 {
			for _, arg := range cmd.ValidArgs {
				if withDesc {
					fmt.Fprintf(c.outWriter, "%s\t\n", arg)
				} else {
					fmt.Fprintln(c.outWriter, arg)
				}
			}
		}
		// Suggest flags
		suggestFlags(cmd, "", withDesc)
		return nil
	}

	// Suggest subcommands that match the prefix
	for _, sub := range cmd.visibleCommands() {
		if strings.HasPrefix(sub.Name(), currentWord) {
			if withDesc {
				fmt.Fprintf(c.outWriter, "%s\t%s\n", sub.Name(), sub.Short)
			} else {
				fmt.Fprintln(c.outWriter, sub.Name())
			}
		}
	}
	// Suggest valid positional arguments that match the prefix
	if len(cmd.ValidArgs) > 0 {
		for _, arg := range cmd.ValidArgs {
			if strings.HasPrefix(arg, currentWord) {
				if withDesc {
					fmt.Fprintf(c.outWriter, "%s\t\n", arg)
				} else {
					fmt.Fprintln(c.outWriter, arg)
				}
			}
		}
	}
	// Suggest flags that match the prefix
	suggestFlags(cmd, currentWord, withDesc)

	return nil
}

func suggestFlags(cmd *Command, prefix string, withDesc bool) {
	flags := make(map[string]string)
	cmd.localFlags.VisitAll(func(f *flag.Flag) {
		if f.Name == "help" || f.Name == "h" {
			return
		}
		flags["-"+f.Name] = f.Usage
		if len(f.Name) > 1 {
			flags["--"+f.Name] = f.Usage
		}
	})
	for _, fs := range cmd.persistentFlagSets() {
		fs.VisitAll(func(f *flag.Flag) {
			if f.Name == "help" || f.Name == "h" {
				return
			}
			flags["-"+f.Name] = f.Usage
			if len(f.Name) > 1 {
				flags["--"+f.Name] = f.Usage
			}
		})
	}
	for name, usage := range flags {
		if strings.HasPrefix(name, prefix) {
			if withDesc {
				fmt.Fprintf(cmd.outWriter, "%s\t%s\n", name, usage)
			} else {
				fmt.Fprintln(cmd.outWriter, name)
			}
		}
	}
}

// GenBashCompletion writes bash completion script to the given writer.
func (c *Command) GenBashCompletion(w io.Writer) error {
	prog := c.Name()
	_, err := fmt.Fprintf(w, `#!/bin/bash
# bash completion for %s

__%s_completion() {
    local cur prev words cword split
    _init_completion -s || return

    local args=("${words[@]:1:$cword-1}")
    if [[ ${#args[@]} -eq $cword-1 ]]; then
        args+=("")
    fi
    COMPREPLY=($( "${words[0]}" __completeNoDesc "${args[@]}" 2>/dev/null ))
}

complete -F __%s_completion %s
`, prog, prog, prog, prog)
	return err
}

// GenZshCompletion writes zsh completion script to the given writer.
func (c *Command) GenZshCompletion(w io.Writer) error {
	prog := c.Name()
	_, err := fmt.Fprintf(w, `#compdef %s

_%s() {
    local -a words
    words=(${(z)BUFFER})
    local cur="${words[$CURRENT]}"
    local args=("${words[@]:1:$CURRENT-1}")
    if [[ ${#args[@]} -eq $CURRENT-1 ]]; then
        args+=("")
    fi
    reply=($(${words[1]} __completeNoDesc "${args[@]}" 2>/dev/null))
}

compdef _%s %s
`, prog, prog, prog, prog)
	return err
}

// GenFishCompletion writes fish completion script to the given writer.
func (c *Command) GenFishCompletion(w io.Writer) error {
	prog := c.Name()
	_, err := fmt.Fprintf(w, `# fish completion for %s

function __%s_completion
    set -l args (commandline -opc)
    set -e args[1]
    set -l cur (commandline -ct)
    if test (count $args) -eq (commandline -poc)
        set args $args ""
    end
    %s __completeNoDesc $args 2>/dev/null
end

complete -c %s -f -a "(__%s_completion)"
`, prog, prog, prog, prog, prog)
	return err
}

// GenPowerShellCompletion writes PowerShell completion script to the given writer.
func (c *Command) GenPowerShellCompletion(w io.Writer) error {
	prog := c.Name()
	_, err := fmt.Fprintf(w, `# PowerShell completion for %s

$script:__%s_completion = {
    param($wordToComplete, $commandAst, $cursorPosition)
    $args = $commandAst.CommandElements[1..$commandAst.CommandElements.Count] | ForEach-Object { $_.Value }
    if ($args.Count -eq $commandAst.CommandElements.Count - 1) {
        $args += ""
    }
    & $commandAst.CommandElements[0].Value __completeNoDesc $args 2>$null
}

Register-ArgumentCompleter -Native -CommandName %s -ScriptBlock $__%s_completion
`, prog, prog, prog, prog)
	return err
}
