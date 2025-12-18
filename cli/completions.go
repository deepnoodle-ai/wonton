package cli

import (
	"fmt"
	"io"
	"strings"
)

// GenerateBashCompletion writes a bash completion script to the writer.
//
// The script enables tab completion for commands, subcommands, and flags.
// To install:
//
//	myapp completion bash > /usr/local/etc/bash_completion.d/myapp
//	# or
//	source <(myapp completion bash)
func (a *App) GenerateBashCompletion(w io.Writer) error {
	tmpl := `# bash completion for %[1]s

_%[1]s_completions() {
    local cur prev commands
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Top-level commands
    commands="%[2]s"

    case "${prev}" in
        %[1]s)
            COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
            return 0
            ;;
%[3]s
    esac

    # Default to file completion
    COMPREPLY=( $(compgen -f -- ${cur}) )
}

complete -F _%[1]s_completions %[1]s
`

	commands := a.getCommandNames()
	groupCases := a.generateBashGroupCases()

	_, err := fmt.Fprintf(w, tmpl, a.name, strings.Join(commands, " "), groupCases)
	return err
}

// GenerateZshCompletion writes a zsh completion script to the writer.
//
// The script enables tab completion for commands, subcommands, and flags.
// To install:
//
//	myapp completion zsh > "${fpath[1]}/_myapp"
//	# or add to ~/.zshrc:
//	eval "$(myapp completion zsh)"
func (a *App) GenerateZshCompletion(w io.Writer) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("#compdef %s\n\n", a.name))

	// Commands function
	sb.WriteString(fmt.Sprintf("__%s_commands() {\n", a.name))
	sb.WriteString("    local commands\n")
	sb.WriteString("    commands=(\n")
	sb.WriteString(a.generateZshCommandDesc())
	sb.WriteString("\n    )\n")
	sb.WriteString(fmt.Sprintf("    _describe -t commands '%s commands' commands\n", a.name))
	sb.WriteString("}\n\n")

	// Group functions
	for name, group := range a.groups {
		sb.WriteString(fmt.Sprintf("__%s_group_%s() {\n", a.name, name))
		sb.WriteString("    local commands\n")
		sb.WriteString("    commands=(\n")
		for cmdName, cmd := range group.commands {
			if !cmd.hidden {
				sb.WriteString(fmt.Sprintf("        '%s:%s'\n", cmdName, escapeZsh(cmd.description)))
			}
		}
		sb.WriteString("    )\n")
		sb.WriteString(fmt.Sprintf("    _describe -t commands '%s %s commands' commands\n", a.name, name))
		sb.WriteString("}\n\n")
	}

	// Main completion function
	sb.WriteString(fmt.Sprintf("_%s() {\n", a.name))
	sb.WriteString("    local curcontext=\"$curcontext\" state line\n")
	sb.WriteString("    typeset -A opt_args\n\n")
	sb.WriteString("    _arguments -C \\\n")
	sb.WriteString(fmt.Sprintf("        '1: :__%s_commands' \\\n", a.name))
	sb.WriteString("        '*::arg:->args'\n\n")
	sb.WriteString("    case $state in\n")
	sb.WriteString("        args)\n")
	sb.WriteString("            case $line[1] in\n")
	for name := range a.groups {
		sb.WriteString(fmt.Sprintf("                %s)\n", name))
		sb.WriteString(fmt.Sprintf("                    __%s_group_%s\n", a.name, name))
		sb.WriteString("                    ;;\n")
	}
	sb.WriteString("            esac\n")
	sb.WriteString("            ;;\n")
	sb.WriteString("    esac\n")
	sb.WriteString("}\n\n")

	// Register completion - works with both eval and file sourcing
	// Check if compdef is available (requires compinit to be loaded)
	sb.WriteString("if (( $+functions[compdef] )); then\n")
	sb.WriteString(fmt.Sprintf("    compdef _%s %s\n", a.name, a.name))
	sb.WriteString("fi\n")

	_, err := io.WriteString(w, sb.String())
	return err
}

// GenerateFishCompletion writes a fish completion script to the writer.
//
// The script enables tab completion for commands, subcommands, and flags.
// To install:
//
//	myapp completion fish > ~/.config/fish/completions/myapp.fish
func (a *App) GenerateFishCompletion(w io.Writer) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# fish completion for %s\n\n", a.name))

	// Disable file completion by default
	sb.WriteString(fmt.Sprintf("complete -c %s -f\n\n", a.name))

	// Add commands
	for name, cmd := range a.commands {
		if cmd.hidden {
			continue
		}
		sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_use_subcommand' -a '%s' -d '%s'\n",
			a.name, name, escapeFish(cmd.description)))

		// Add flags for each command
		for _, flag := range cmd.flags {
			if flag.IsHidden() {
				continue
			}
			if flag.GetShort() != "" {
				sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_seen_subcommand_from %s' -s '%s' -l '%s' -d '%s'\n",
					a.name, name, flag.GetShort(), flag.GetName(), escapeFish(flag.GetHelp())))
			} else {
				sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_seen_subcommand_from %s' -l '%s' -d '%s'\n",
					a.name, name, flag.GetName(), escapeFish(flag.GetHelp())))
			}
		}
	}

	// Add groups
	for groupName, group := range a.groups {
		sb.WriteString(fmt.Sprintf("\n# Group: %s\n", groupName))
		sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_use_subcommand' -a '%s' -d '%s'\n",
			a.name, groupName, escapeFish(group.description)))

		for cmdName, cmd := range group.commands {
			if cmd.hidden {
				continue
			}
			sb.WriteString(fmt.Sprintf("complete -c %s -n '__fish_seen_subcommand_from %s' -a '%s' -d '%s'\n",
				a.name, groupName, cmdName, escapeFish(cmd.description)))
		}
	}

	_, err := io.WriteString(w, sb.String())
	return err
}

// CompletionCommand returns a command that generates shell completions.
//
// This command can be added to your app to provide built-in completion generation:
//
//	app.commands["completion"] = cli.CompletionCommand()
//
// Or use AddCompletionCommand for convenience.
func CompletionCommand() *Command {
	cmd := &Command{
		name:        "completion",
		description: "Generate shell completion scripts",
		flags:       make([]Flag, 0),
		args:        make([]*Arg, 0),
	}
	cmd.AddArg(&Arg{
		Name:        "shell",
		Description: "Shell type (bash, zsh, fish)",
		Required:    true,
	})
	cmd.handler = func(ctx *Context) error {
		shell := ctx.Arg(0)
		switch shell {
		case "bash":
			return ctx.App().GenerateBashCompletion(ctx.Stdout())
		case "zsh":
			return ctx.App().GenerateZshCompletion(ctx.Stdout())
		case "fish":
			return ctx.App().GenerateFishCompletion(ctx.Stdout())
		default:
			return Errorf("unsupported shell: %s (use bash, zsh, or fish)", shell)
		}
	}
	return cmd
}

// AddCompletionCommand adds the built-in completion command to the app.
//
// Users can then generate completions with:
//
//	myapp completion bash
//	myapp completion zsh
//	myapp completion fish
//
// Example:
//
//	app := cli.New("myapp")
//	app.AddCompletionCommand()
//	// Users can now run: myapp completion bash
func (a *App) AddCompletionCommand() *App {
	cmd := CompletionCommand()
	cmd.app = a
	a.commands["completion"] = cmd
	return a
}

// Helper functions

func (a *App) getCommandNames() []string {
	var names []string
	for name, cmd := range a.commands {
		if !cmd.hidden {
			names = append(names, name)
		}
	}
	for name := range a.groups {
		names = append(names, name)
	}
	return names
}

func (a *App) generateBashGroupCases() string {
	var sb strings.Builder
	for name, group := range a.groups {
		var cmds []string
		for cmdName, cmd := range group.commands {
			if !cmd.hidden {
				cmds = append(cmds, cmdName)
			}
		}
		sb.WriteString(fmt.Sprintf("        %s)\n", name))
		sb.WriteString(fmt.Sprintf("            COMPREPLY=( $(compgen -W \"%s\" -- ${cur}) )\n", strings.Join(cmds, " ")))
		sb.WriteString("            return 0\n")
		sb.WriteString("            ;;\n")
	}
	return sb.String()
}

func (a *App) generateZshCommandDesc() string {
	var lines []string
	for name, cmd := range a.commands {
		if !cmd.hidden {
			lines = append(lines, fmt.Sprintf("        '%s:%s'", name, escapeZsh(cmd.description)))
		}
	}
	for name, group := range a.groups {
		lines = append(lines, fmt.Sprintf("        '%s:%s'", name, escapeZsh(group.description)))
	}
	return strings.Join(lines, "\n")
}

func escapeZsh(s string) string {
	s = strings.ReplaceAll(s, "'", "'\\''")
	s = strings.ReplaceAll(s, ":", "\\:")
	return s
}

func escapeFish(s string) string {
	s = strings.ReplaceAll(s, "'", "\\'")
	return s
}
