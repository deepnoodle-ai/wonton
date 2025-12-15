package cli

import (
	"fmt"
	"io"
	"strings"
)

// GenerateBashCompletion writes bash completion script to w.
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

// GenerateZshCompletion writes zsh completion script to w.
func (a *App) GenerateZshCompletion(w io.Writer) error {
	tmpl := `#compdef %[1]s

__%[1]s_commands() {
    local commands
    commands=(
%[2]s
    )
    _describe -t commands '%[1]s commands' commands
}

__%[1]s_group_%[3]s() {
    local commands
    commands=(
%[4]s
    )
    _describe -t commands '%[1]s %[3]s commands' commands
}

_%[1]s() {
    local curcontext="$curcontext" state line
    typeset -A opt_args

    _arguments -C \
        '1: :__%[1]s_commands' \
        '*::arg:->args'

    case $state in
        args)
            case $line[1] in
%[5]s
            esac
            ;;
    esac
}

_%[1]s "$@"
`

	cmdDesc := a.generateZshCommandDesc()
	groupName := ""
	groupCmds := ""
	groupCases := ""

	// Generate group completions
	for name, group := range a.groups {
		groupName = name
		var cmds []string
		for cmdName, cmd := range group.commands {
			if !cmd.hidden {
				cmds = append(cmds, fmt.Sprintf("        '%s:%s'", cmdName, escapeZsh(cmd.description)))
			}
		}
		groupCmds = strings.Join(cmds, "\n")
		groupCases += fmt.Sprintf("                %s)\n                    __%s_group_%s\n                    ;;\n",
			name, a.name, name)
	}

	_, err := fmt.Fprintf(w, tmpl, a.name, cmdDesc, groupName, groupCmds, groupCases)
	return err
}

// GenerateFishCompletion writes fish completion script to w.
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

// AddCompletionCommand adds the completion command to the app.
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
