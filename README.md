# cli

A minimal, dependency-free CLI framework for building command-line applications in Go.

Inspired by [Cobra](https://github.com/spf13/cobra) but with a simpler design and zero external dependencies.

## Features

- **Commands & Subcommands** – Build nested command structures.
- **Flags** – Local and persistent flags using Go’s standard `flag` package.
- **Help System** – Automatic help generation with grouping.
- **Argument Validation** – Custom positional argument validators.
- **Hooks** – PersistentPreRun, PreRun, PostRun, PersistentPostRun.
- **Error Handling** – Silence errors/usage when needed.
- **Command Suggestions** – Levenshtein-based suggestions for mistyped commands.
- **Shell Completions** – Generate completions for Bash, Zsh, Fish, and PowerShell.
- **No Dependencies** – Only uses the Go standard library.

## Installation

```bash
go get github.com/neomen/cli
```

## Quick Start

Create a simple command with a flag:

```go
package main

import (
    "fmt"
    "log"

    "github.com/neomen/cli"
)

func main() {
    root := cli.NewCommand("myapp", "A simple CLI app", "")

    var name string
    root.Flags().StringVar(&name, "name", "World", "who to greet")

    root.Run = func(cmd *cli.Command, args []string) error {
        fmt.Printf("Hello, %s!\n", name)
        return nil
    }

    if err := root.Execute(); err != nil {
        log.Fatal(err)
    }
}
```

Run it:

```bash
$ go build -o myapp
$ ./myapp --name=Alice
Hello, Alice!
```

## Usage Examples

### Adding Subcommands

```go
serve := cli.NewCommand("serve", "Start the server", "Long description...")
serve.Flags().Int("port", 8080, "Port to listen on")
serve.Run = func(cmd *cli.Command, args []string) error {
    port, _ := cmd.Flags().GetInt("port")
    fmt.Printf("Starting server on port %d\n", port)
    return nil
}

root.AddCommand(serve)
```

### Persistent Flags

Persistent flags are available to the command and all its subcommands.

```go
root.PersistentFlags().Bool("verbose", false, "Enable verbose output")

// In a subcommand:
verbose, _ := cmd.PersistentFlags().GetBool("verbose")
```

### Argument Validation

```go
cmd.Args = func(cmd *cli.Command, args []string) error {
    if len(args) != 2 {
        return fmt.Errorf("requires exactly 2 arguments")
    }
    return nil
}
```

### Hooks

```go
cmd.PersistentPreRunE = func(cmd *cli.Command, args []string) error {
    fmt.Println("Before command execution")
    return nil
}

cmd.PostRunE = func(cmd *cli.Command, args []string) error {
    fmt.Println("After command execution")
    return nil
}
```

### Shell Completions

Generate completion scripts for various shells:

```go
// For Bash
root.GenBashCompletion(os.Stdout)

// For Zsh
root.GenZshCompletion(os.Stdout)

// For Fish
root.GenFishCompletion(os.Stdout)

// For PowerShell
root.GenPowerShellCompletion(os.Stdout)
```

## Documentation

The API is fully documented in the source code. See the [GoDoc](https://pkg.go.dev/github.com/neomen/cli) for detailed information.

## License

MIT License – see the [LICENSE](LICENSE) file for details.