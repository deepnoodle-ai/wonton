package cli

import (
	"fmt"
	"strings"
)

// parseResult contains the result of parsing command-line arguments.
type parseResult struct {
	// GlobalFlags are flag arguments that appear before the command.
	// Each entry is either "--name=value" or a pair of entries for "--name", "value".
	GlobalFlags []string

	// Command is the resolved command name (may be empty for root handler).
	Command string

	// Group is the group name if the command is within a group.
	Group string

	// CommandArgs are all arguments after the command name.
	CommandArgs []string
}

// parser handles command-line argument parsing with knowledge of the app structure.
type parser struct {
	app *App
}

// newParser creates a parser for the given app.
func newParser(app *App) *parser {
	return &parser{app: app}
}

// parse processes the argument list and returns a parseResult.
// It uses the app's command definitions to determine where global flags end
// and the command begins, avoiding heuristic-based guessing.
func (p *parser) parse(args []string) (*parseResult, error) {
	result := &parseResult{}

	// Handle empty args
	if len(args) == 0 {
		return result, nil
	}

	// Scan arguments looking for:
	// 1. The end-of-flags marker (--)
	// 2. A known command or group name
	// 3. Global flags and their values

	i := 0
	for i < len(args) {
		arg := args[i]

		// Handle end-of-flags marker
		if arg == "--" {
			// Everything after -- is treated as positional args (command detection still applies)
			i++
			if i < len(args) {
				// Try to resolve the next arg as a command/group
				p.resolveCommandPosition(result, args, i)
			}
			return result, nil
		}

		// Check if this arg is a known command or group
		if !looksLikeFlag(arg) {
			p.resolveCommandPosition(result, args, i)
			return result, nil
		}

		// This is a flag - parse it as a global flag
		if strings.HasPrefix(arg, "--") {
			name := strings.TrimPrefix(arg, "--")

			// Handle --name=value form
			if idx := strings.Index(name, "="); idx >= 0 {
				result.GlobalFlags = append(result.GlobalFlags, arg)
				i++
				continue
			}

			// Handle --help specially
			if name == "help" {
				result.GlobalFlags = append(result.GlobalFlags, arg)
				i++
				continue
			}

			// Look up the global flag
			gf := p.app.findGlobalFlag(name)
			if gf == nil {
				// Unknown global flag - could be a command-level flag appearing early.
				// Check if next arg looks like a command name.
				if i+1 < len(args) {
					next := args[i+1]
					if !looksLikeFlag(next) && p.isKnownCommand(next) {
						// Next arg is a command - this flag belongs to it
						// Include this flag in CommandArgs and resolve the command
						p.resolveCommandPosition(result, args, i+1)
						result.CommandArgs = append([]string{arg}, result.CommandArgs...)
						return result, nil
					}
					// Next arg is not a known command.
					// If it doesn't look like a flag, assume it's this flag's value.
					if !looksLikeFlag(next) {
						result.GlobalFlags = append(result.GlobalFlags, arg, next)
						i += 2
						continue
					}
				}
				// Flag without value or followed by another flag
				result.GlobalFlags = append(result.GlobalFlags, arg)
				i++
				continue
			}

			// Known global flag
			if _, isBool := gf.GetDefault().(bool); isBool {
				// Boolean flag - no value needed
				result.GlobalFlags = append(result.GlobalFlags, arg)
				i++
			} else {
				// Non-boolean flag - needs a value
				if i+1 >= len(args) {
					return nil, fmt.Errorf("flag --%s requires a value", name)
				}
				result.GlobalFlags = append(result.GlobalFlags, arg, args[i+1])
				i += 2
			}
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			// Short flag(s)
			shorts := arg[1:]

			// Handle -h specially
			if shorts == "h" {
				result.GlobalFlags = append(result.GlobalFlags, arg)
				i++
				continue
			}

			// For bundled short flags like -abc, only the last one can take a value
			lastShort := string(shorts[len(shorts)-1])
			gf := p.app.findGlobalFlagByShort(lastShort)

			if gf == nil {
				// Unknown short flag - similar logic to long flags
				if i+1 < len(args) {
					next := args[i+1]
					if !looksLikeFlag(next) && p.isKnownCommand(next) {
						p.resolveCommandPosition(result, args, i+1)
						result.CommandArgs = append([]string{arg}, result.CommandArgs...)
						return result, nil
					}
					if !looksLikeFlag(next) {
						result.GlobalFlags = append(result.GlobalFlags, arg, next)
						i += 2
						continue
					}
				}
				result.GlobalFlags = append(result.GlobalFlags, arg)
				i++
				continue
			}

			// Known global flag
			if _, isBool := gf.GetDefault().(bool); isBool {
				result.GlobalFlags = append(result.GlobalFlags, arg)
				i++
			} else {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("flag -%s requires a value", lastShort)
				}
				result.GlobalFlags = append(result.GlobalFlags, arg, args[i+1])
				i += 2
			}
		} else {
			// Bare "-" is treated as a positional argument
			result.CommandArgs = args[i:]
			return result, nil
		}
	}

	// Reached end of args - no command found, just global flags
	return result, nil
}

// resolveCommandPosition resolves the command/group at args[pos] and sets up
// the result accordingly. This handles group subcommand resolution and ensures
// unknown tokens are preserved as positional args.
func (p *parser) resolveCommandPosition(result *parseResult, args []string, pos int) {
	if pos >= len(args) {
		return
	}

	arg := args[pos]
	cmd, group := p.resolveCommand(arg)

	if cmd != "" && group == "" {
		// Direct command (not a group)
		result.Command = cmd
		result.CommandArgs = args[pos+1:]
		return
	}

	if group != "" {
		// It's a group - check for subcommand
		result.Group = group
		if cmd != "" {
			// group:command syntax already resolved the subcommand
			result.Command = cmd
			result.CommandArgs = args[pos+1:]
			return
		}

		// Look for subcommand in remaining args
		if pos+1 < len(args) {
			nextArg := args[pos+1]
			if subCmd := p.resolveGroupSubcommand(group, nextArg); subCmd != "" {
				result.Command = subCmd
				result.CommandArgs = args[pos+2:]
				return
			}
		}
		// Group without subcommand - remaining args go to group handler
		result.CommandArgs = args[pos+1:]
		return
	}

	// Not a known command or group
	if !p.hasCommands() {
		// No commands defined - treat all as positional
		result.CommandArgs = args[pos:]
		return
	}

	// Unknown token in command position - let ExecuteContext handle it
	// Preserve the token as the command name for error reporting
	result.Command = arg
	result.CommandArgs = args[pos+1:]
}

// resolveCommand checks if the argument matches a known command or group.
// Returns (commandName, groupName). For a direct command, groupName is empty.
// For a group, commandName is empty. For group:command syntax, both are set.
func (p *parser) resolveCommand(arg string) (command, group string) {
	// Built-in commands are always recognized
	if arg == "help" || arg == "version" {
		return arg, ""
	}

	// Check direct commands
	if _, ok := p.app.commands[arg]; ok {
		return arg, ""
	}

	// Check command aliases
	for name, cmd := range p.app.commands {
		for _, alias := range cmd.aliases {
			if alias == arg {
				return name, ""
			}
		}
	}

	// Check for group:command pattern
	if parts := strings.SplitN(arg, ":", 2); len(parts) == 2 {
		if g, ok := p.app.groups[parts[0]]; ok {
			if _, ok := g.commands[parts[1]]; ok {
				return parts[1], parts[0]
			}
			// Check aliases within the group
			for cmdName, cmd := range g.commands {
				for _, alias := range cmd.aliases {
					if alias == parts[1] {
						return cmdName, parts[0]
					}
				}
			}
		}
	}

	// Check if it's a group name
	if _, ok := p.app.groups[arg]; ok {
		return "", arg
	}

	return "", ""
}

// resolveGroupSubcommand checks if subArg is a valid subcommand of the given group.
func (p *parser) resolveGroupSubcommand(groupName, subArg string) string {
	g, ok := p.app.groups[groupName]
	if !ok {
		return ""
	}

	// Direct subcommand match
	if _, ok := g.commands[subArg]; ok {
		return subArg
	}

	// Check aliases
	for name, cmd := range g.commands {
		for _, alias := range cmd.aliases {
			if alias == subArg {
				return name
			}
		}
	}

	return ""
}

// isKnownCommand returns true if arg is a known command, group, or alias.
func (p *parser) isKnownCommand(arg string) bool {
	cmd, group := p.resolveCommand(arg)
	return cmd != "" || group != ""
}

// hasCommands returns true if the app has any defined commands or groups.
func (p *parser) hasCommands() bool {
	// Check for non-root commands
	for name := range p.app.commands {
		if name != "" {
			return true
		}
	}
	return len(p.app.groups) > 0
}

// looksLikeFlag returns true if the string looks like a flag rather than a value.
// This allows values like "-1" (negative numbers) while still treating "-v" as a flag.
func looksLikeFlag(s string) bool {
	if !strings.HasPrefix(s, "-") {
		return false
	}
	if len(s) == 1 {
		return false // just "-"
	}
	// Check if it could be a negative number
	if len(s) >= 2 {
		second := s[1]
		// -N or -.N where N is a digit
		if second >= '0' && second <= '9' {
			return false // looks like negative number
		}
		if second == '.' && len(s) > 2 {
			return false // looks like negative decimal
		}
	}
	return true
}
