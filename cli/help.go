package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/deepnoodle-ai/wonton/color"
	"github.com/deepnoodle-ai/wonton/tui"
)

// HelpTheme defines the color scheme for styled help output.
//
// Customize the theme by creating a theme and passing it to App.HelpTheme:
//
//	theme := cli.DefaultHelpTheme()
//	theme.TitleStart = color.NewRGB(255, 100, 100)
//	theme.TitleEnd = color.NewRGB(100, 100, 255)
//	app.HelpTheme(theme)
type HelpTheme struct {
	// TitleGradient defines the start and end colors for the app name gradient.
	// Set both to the same value for a solid color.
	TitleStart color.RGB
	TitleEnd   color.RGB

	// SectionHeader is the style for section headers (USAGE, COMMANDS, etc.)
	SectionHeader tui.Style

	// Command is the style for command names
	Command tui.Style

	// Flag is the style for flag names
	Flag tui.Style

	// Hint is the style for hints, defaults, and metadata
	Hint tui.Style

	// Deprecated is the style for deprecation warnings
	Deprecated tui.Style
}

// DefaultHelpTheme returns the default help theme with blue titles.
//
// Get the default theme, customize it, then apply:
//
//	theme := cli.DefaultHelpTheme()
//	theme.Command = tui.NewStyle().WithForeground(tui.ColorYellow).WithBold()
//	app.HelpTheme(theme)
func DefaultHelpTheme() HelpTheme {
	return HelpTheme{
		TitleStart:    color.NewRGB(80, 140, 255),
		TitleEnd:      color.NewRGB(80, 140, 255),
		SectionHeader: tui.NewStyle().WithForeground(tui.ColorBrightWhite).WithBold(),
		Command:       tui.NewStyle().WithForeground(tui.ColorBrightGreen).WithBold(),
		Flag:          tui.NewStyle().WithForeground(tui.ColorBrightCyan),
		Hint:          tui.NewStyle().WithForeground(tui.ColorBrightBlack),
		Deprecated:    tui.NewStyle().WithForeground(tui.ColorYellow).WithItalic(),
	}
}

// getHelpTheme returns the app's custom theme or the default theme.
func (a *App) getHelpTheme() HelpTheme {
	if a.helpTheme != nil {
		return *a.helpTheme
	}
	return DefaultHelpTheme()
}

// renderAppHelp renders the main application help
func (a *App) renderAppHelp() tui.View {
	theme := a.getHelpTheme()

	return tui.Stack(
		renderHeader(a.name, a.description, a.version, theme),
		tui.Stack(
			renderSection("USAGE", theme),
			tui.Text("  %s <command> [flags] [args]", a.name),
		),
		tui.If(len(a.commands) > 0, tui.Stack(
			renderSection("COMMANDS", theme),
			renderCommands(a.commands, theme),
		)),
		tui.If(len(a.groups) > 0, tui.Stack(
			renderSection("COMMAND GROUPS", theme),
			renderGroups(a.groups, theme),
		)),
		tui.If(len(a.globalFlags) > 0, tui.Stack(
			renderSection("GLOBAL FLAGS", theme),
			renderFlags(a.globalFlags, theme),
		)),
		renderFooter(a.name, theme),
	).Gap(1)
}

// renderCommandHelp renders help for a specific command
func (c *Command) renderCommandHelp() tui.View {
	theme := c.app.getHelpTheme()

	cmdName := c.name
	if c.group != nil {
		cmdName = c.group.name + " " + c.name
	}

	return tui.Stack(
		renderCommandHeader(cmdName, c.description, theme),
		tui.If(c.deprecated != "", tui.Group(
			tui.Text("  DEPRECATED: ").Style(theme.Deprecated),
			tui.Text("%s", c.deprecated).Style(theme.Deprecated),
		)),
		tui.If(c.longDesc != "", tui.Text("  %s", c.longDesc).Style(theme.Hint)),
		tui.Stack(
			renderSection("USAGE", theme),
			tui.Text("%s", buildUsageString(c)),
		),
		tui.If(len(c.aliases) > 0, tui.Stack(
			renderSection("ALIASES", theme),
			tui.Text("  %s", strings.Join(c.aliases, ", ")).Style(theme.Command),
		)),
		tui.If(len(c.args) > 0, tui.Stack(
			renderSection("ARGUMENTS", theme),
			renderArgs(c.args, theme),
		)),
		tui.If(len(c.flags) > 0, tui.Stack(
			renderSection("FLAGS", theme),
			renderFlags(c.flags, theme),
		)),
		tui.If(c.app != nil && len(c.app.globalFlags) > 0, tui.Stack(
			renderSection("GLOBAL FLAGS", theme),
			renderFlags(c.app.globalFlags, theme),
		)),
	).Gap(1)
}

// buildUsageString constructs the usage string for a command
func buildUsageString(c *Command) string {
	usage := "  " + c.app.name
	if c.group != nil {
		usage += " " + c.group.name
	}
	usage += " " + c.name
	if len(c.flags) > 0 || len(c.app.globalFlags) > 0 {
		usage += " [flags]"
	}
	for _, arg := range c.args {
		if arg.Required {
			usage += " <" + arg.Name + ">"
		} else {
			usage += " [" + arg.Name + "]"
		}
	}
	return usage
}

// renderHeader creates the styled app header with gradient title
func renderHeader(name, description, version string, theme HelpTheme) tui.View {
	titleLine := renderGradientText(name, theme.TitleStart, theme.TitleEnd)
	if description != "" {
		titleLine = tui.Group(
			titleLine,
			tui.Text(" - "),
			tui.Text("%s", description),
		)
	}

	return tui.Stack(
		titleLine,
		tui.If(version != "", tui.Text("  v%s", version).Style(theme.Hint)),
	).Gap(0)
}

// renderCommandHeader creates the styled command header
func renderCommandHeader(name, description string, theme HelpTheme) tui.View {
	if description != "" {
		return tui.Group(
			tui.Text("%s", name).Style(theme.Command),
			tui.Text(" - "),
			tui.Text("%s", description),
		)
	}
	return tui.Text("%s", name).Style(theme.Command)
}

// renderSection creates a styled section header
func renderSection(title string, theme HelpTheme) tui.View {
	return tui.Text("%s", title).Style(theme.SectionHeader)
}

// renderGradientText creates text with a horizontal color gradient
func renderGradientText(text string, start, end color.RGB) tui.View {
	runes := []rune(text)
	if len(runes) == 0 {
		return tui.Empty()
	}

	colors := color.Gradient(start, end, len(runes))
	views := make([]tui.View, len(runes))

	for i, r := range runes {
		style := tui.NewStyle().WithFgRGB(colors[i]).WithBold()
		views[i] = tui.Text("%s", string(r)).Style(style)
	}

	return tui.Group(views...)
}

// renderCommands renders the command list as a Stack
func renderCommands(commands map[string]*Command, theme HelpTheme) tui.View {
	names := sortedKeys(commands)
	views := make([]tui.View, 0, len(names))

	for _, name := range names {
		cmd := commands[name]
		if cmd.hidden {
			continue
		}
		views = append(views, tui.Group(
			tui.Text("  %-16s", name).Style(theme.Command),
			tui.Text("%s", cmd.description),
		))
	}

	return tui.Stack(views...).Gap(0)
}

// renderGroups renders the command groups list as a Stack
func renderGroups(groups map[string]*Group, theme HelpTheme) tui.View {
	names := sortedGroupKeys(groups)
	views := make([]tui.View, 0, len(names)*2)

	for _, name := range names {
		group := groups[name]
		views = append(views, tui.Group(
			tui.Text("  %-16s", name).Style(theme.Command),
			tui.Text("%s", group.description),
		))

		// Subcommands
		subNames := sortedKeys(group.commands)
		for _, subName := range subNames {
			subCmd := group.commands[subName]
			if subCmd.hidden {
				continue
			}
			views = append(views, tui.Group(
				tui.Text("    %-14s", subName).Style(theme.Flag),
				tui.Text("%s", subCmd.description).Style(theme.Hint),
			))
		}
	}

	return tui.Stack(views...).Gap(0)
}

// renderFlags renders the flags list as a Stack
func renderFlags(flags []Flag, theme HelpTheme) tui.View {
	views := make([]tui.View, 0, len(flags))

	for _, f := range flags {
		if f.IsHidden() {
			continue
		}
		views = append(views, renderFlag(f, theme))
	}

	return tui.Stack(views...).Gap(0)
}

// renderFlag renders a single flag
func renderFlag(f Flag, theme HelpTheme) tui.View {
	// Build flag prefix
	prefix := "  "
	if f.GetShort() != "" {
		prefix += "-" + f.GetShort() + ", "
	} else {
		prefix += "    "
	}
	prefix += fmt.Sprintf("--%-14s", f.GetName())

	// Build metadata
	meta := buildFlagMeta(f)

	return tui.Group(
		tui.Text("%s", prefix).Style(theme.Flag),
		tui.Text("%s", f.GetHelp()),
		tui.If(meta != "", tui.Text(" (%s)", meta).Style(theme.Hint)),
	)
}

// buildFlagMeta builds the metadata string for a flag
func buildFlagMeta(f Flag) string {
	var parts []string
	if def := f.GetDefault(); def != nil && def != "" && def != false && def != 0 {
		parts = append(parts, fmt.Sprintf("default: %v", def))
	}
	if f.IsRequired() {
		parts = append(parts, "required")
	}
	if enum := f.GetEnum(); len(enum) > 0 {
		parts = append(parts, strings.Join(enum, "|"))
	}
	return strings.Join(parts, ", ")
}

// renderArgs renders the arguments list as a Stack
func renderArgs(args []*Arg, theme HelpTheme) tui.View {
	views := make([]tui.View, len(args))

	for i, arg := range args {
		views[i] = tui.Group(
			tui.Text("  %-16s", arg.Name).Style(theme.Command),
			tui.If(arg.Description != "", tui.Text("%s", arg.Description)),
			tui.If(!arg.Required, tui.Text(" (optional)").Style(theme.Hint)),
		)
	}

	return tui.Stack(views...).Gap(0)
}

// renderFooter creates the help footer with hint
func renderFooter(appName string, theme HelpTheme) tui.View {
	return tui.Group(
		tui.Text("Run '"),
		tui.Text("%s <command> --help", appName).Style(theme.Flag),
		tui.Text("' for more information on a command."),
	)
}

// sortedKeys returns sorted command names
func sortedKeys(commands map[string]*Command) []string {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// sortedGroupKeys returns sorted group names
func sortedGroupKeys(groups map[string]*Group) []string {
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
