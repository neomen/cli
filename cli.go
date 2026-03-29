package cli

import (
	"context"
	"flag"
	"io"
	"os"
	"strconv"
	"strings"
)

// PositionalArgs defines a function to validate positional arguments.
type PositionalArgs func(cmd *Command, args []string) error

// Group represents a group of commands in help output.
type Group struct {
	ID    string
	Title string
}

// flagMeta stores metadata about a flag that has both long and short forms.
type flagMeta struct {
	longName  string
	shortName string
	usage     string
	value     flag.Value // the underlying flag.Value
	defValue  string     // default value as string
}

// Command represents a CLI command with subcommands, flags, and an action.
type Command struct {
	// Use is the one-line usage message. Recommended syntax: "cmd [flags] [arg...]"
	Use string
	// Short is the short description shown in help.
	Short string
	// Long is the long message shown in help.
	Long string
	// Example is an optional field to show usage examples.
	Example string
	// Deprecated marks this command as deprecated and prints the given message.
	Deprecated string
	// Version defines the version for this command. If set on the root command,
	// a --version flag will be added automatically.
	Version string
	// Annotations are arbitrary key-value pairs attached to the command.
	Annotations map[string]string

	// Run is the function that executes the command. If nil, the command is considered a group.
	// The command's context is available via cmd.Context().
	Run func(cmd *Command, args []string) error

	// Aliases is a list of alternative names for this command.
	Aliases []string
	// SuggestFor is a list of command names for which this command will be suggested.
	SuggestFor []string
	// Args defines validation for positional arguments.
	Args PositionalArgs
	// ValidArgs is a list of valid positional arguments for shell completion.
	ValidArgs []string
	// ArgAliases is a list of aliases for ValidArgs (accepted but not suggested).
	ArgAliases []string

	// GroupID identifies the command group in help output.
	GroupID string
	// CommandGroups is a list of groups for subcommands (used on parent).
	CommandGroups []*Group

	// Hidden, if true, hides the command from help output.
	Hidden bool
	// SilenceErrors, if true, suppresses automatic error printing.
	SilenceErrors bool
	// SilenceUsage, if true, suppresses usage printing when an error occurs.
	SilenceUsage bool
	// DisableFlagParsing disables flag parsing entirely; all arguments are treated as positional.
	DisableFlagParsing bool
	// DisableSuggestions disables command name suggestions for unknown commands.
	DisableSuggestions bool
	// SuggestionsMinimumDistance sets the minimum Levenshtein distance for suggestions.
	SuggestionsMinimumDistance int
	// TraverseChildren, if true, parses flags on all parents before executing child commands.
	TraverseChildren bool

	// PersistentPreRunE is executed before any children commands (in parent order).
	PersistentPreRunE func(cmd *Command, args []string) error
	// PreRunE is executed before the command's Run function.
	PreRunE func(cmd *Command, args []string) error
	// PostRunE is executed after the command's Run function.
	PostRunE func(cmd *Command, args []string) error
	// PersistentPostRunE is executed after all children commands (in child order).
	PersistentPostRunE func(cmd *Command, args []string) error

	// localFlags is the flag set for flags that only apply to this command.
	localFlags *flag.FlagSet
	// persistentFlags is the flag set for flags that apply to this command and all its children.
	persistentFlags *flag.FlagSet

	// parent is the parent command.
	parent *Command
	// children is the list of subcommands.
	children []*Command
	// childrenMap is a map for fast lookup by name or alias.
	childrenMap map[string]*Command

	// ctx is the context for this command execution.
	ctx context.Context

	// I/O streams, can be overridden for testing.
	inReader  io.Reader
	outWriter io.Writer
	errWriter io.Writer

	// NEW fields for flag parsing results
	parsedFlags  *flag.FlagSet // combined set used for parsing
	leftoverArgs []string      // arguments after flag parsing

	// flagMetaMap maps each flag name (both long and short) to its metadata.
	flagMetaMap map[string]*flagMeta
}

// NewCommand creates a new command with the given name and description.
func NewCommand(use, short, long string) *Command {
	cmd := &Command{
		Use:                        use,
		Short:                      short,
		Long:                       long,
		SuggestionsMinimumDistance: 2,
		inReader:                   os.Stdin,
		outWriter:                  os.Stdout,
		errWriter:                  os.Stderr,
		flagMetaMap:                make(map[string]*flagMeta),
	}
	cmd.localFlags = flag.NewFlagSet(use, flag.ContinueOnError)
	cmd.persistentFlags = flag.NewFlagSet(use, flag.ContinueOnError)

	// Add help flags to both sets so that --help can be detected.
	cmd.localFlags.Bool("help", false, "help for "+use)
	cmd.localFlags.Bool("h", false, "help for "+use)
	cmd.persistentFlags.Bool("help", false, "help for "+use)
	cmd.persistentFlags.Bool("h", false, "help for "+use)

	// If this command has no parent (i.e., it's the root), add the hidden completion commands.
	if cmd.parent == nil {
		cmd.AddCompletionCommand()
	}
	return cmd
}

// Flags returns the local flag set for this command.
func (c *Command) Flags() *flag.FlagSet {
	return c.localFlags
}

// PersistentFlags returns the persistent flag set for this command.
func (c *Command) PersistentFlags() *flag.FlagSet {
	return c.persistentFlags
}

// AddCommand adds one or more subcommands to this command.
func (c *Command) AddCommand(children ...*Command) {
	for _, child := range children {
		child.parent = c
		c.children = append(c.children, child)
		if c.childrenMap == nil {
			c.childrenMap = make(map[string]*Command)
		}
		c.childrenMap[child.Use] = child
		for _, alias := range child.Aliases {
			c.childrenMap[alias] = child
		}
	}
}

// Commands returns a list of subcommands.
func (c *Command) Commands() []*Command {
	return c.children
}

// Command returns the subcommand with the given name or alias, or nil if not found.
func (c *Command) Command(name string) *Command {
	if c.childrenMap == nil {
		return nil
	}
	return c.childrenMap[name]
}

// Parent returns the parent command.
func (c *Command) Parent() *Command {
	return c.parent
}

// Name returns the command name (the first word of Use).
func (c *Command) Name() string {
	name := strings.Fields(c.Use)[0]
	return name
}

// Context returns the context associated with this command.
// It is set by ExecuteContext or SetContext.
func (c *Command) Context() context.Context {
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

// SetContext sets the context for this command.
func (c *Command) SetContext(ctx context.Context) {
	c.ctx = ctx
}

// InReader returns the input reader.
func (c *Command) InReader() io.Reader {
	return c.inReader
}

// SetInReader sets the input reader.
func (c *Command) SetInReader(r io.Reader) {
	c.inReader = r
}

// OutWriter returns the output writer.
func (c *Command) OutWriter() io.Writer {
	return c.outWriter
}

// SetOutWriter sets the output writer.
func (c *Command) SetOutWriter(w io.Writer) {
	c.outWriter = w
}

// ErrWriter returns the error writer.
func (c *Command) ErrWriter() io.Writer {
	return c.errWriter
}

// SetErrWriter sets the error writer.
func (c *Command) SetErrWriter(w io.Writer) {
	c.errWriter = w
}

// persistentFlagSets returns a slice of persistent flag sets from the root down to this command.
func (c *Command) persistentFlagSets() []*flag.FlagSet {
	var sets []*flag.FlagSet
	for cmd := c; cmd != nil; cmd = cmd.parent {
		sets = append([]*flag.FlagSet{cmd.persistentFlags}, sets...)
	}
	return sets
}

// ---------- Combined Flag Methods ----------

// IntVarP defines an integer flag with both long and short names.
// It sets the variable p to the value given on the command line.
func (c *Command) IntVarP(p *int, long, short string, value int, usage string) {
	// Register long flag (visible)
	c.localFlags.IntVar(p, long, value, usage)
	// Register short flag (hidden by skipping in help)
	c.localFlags.IntVar(p, short, value, usage)
	// Store metadata for help merging
	meta := &flagMeta{
		longName:  long,
		shortName: short,
		usage:     usage,
		value:     c.localFlags.Lookup(long).Value,
		defValue:  flagValueToString(value),
	}
	c.flagMetaMap[long] = meta
	c.flagMetaMap[short] = meta
}

// StringVarP defines a string flag with both long and short names.
func (c *Command) StringVarP(p *string, long, short string, value string, usage string) {
	c.localFlags.StringVar(p, long, value, usage)
	c.localFlags.StringVar(p, short, value, usage)
	meta := &flagMeta{
		longName:  long,
		shortName: short,
		usage:     usage,
		value:     c.localFlags.Lookup(long).Value,
		defValue:  value,
	}
	c.flagMetaMap[long] = meta
	c.flagMetaMap[short] = meta
}

// BoolVarP defines a boolean flag with both long and short names.
func (c *Command) BoolVarP(p *bool, long, short string, value bool, usage string) {
	c.localFlags.BoolVar(p, long, value, usage)
	c.localFlags.BoolVar(p, short, value, usage)
	meta := &flagMeta{
		longName:  long,
		shortName: short,
		usage:     usage,
		value:     c.localFlags.Lookup(long).Value,
		defValue:  flagValueToString(value),
	}
	c.flagMetaMap[long] = meta
	c.flagMetaMap[short] = meta
}

// helper to convert a flag's default value to a string representation
func flagValueToString(v interface{}) string {
	switch val := v.(type) {
	case int:
		return strconv.Itoa(val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case string:
		return val
	default:
		return ""
	}
}
