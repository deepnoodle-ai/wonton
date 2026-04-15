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

// hasSubcommands returns true if the app has any subcommands or groups.
// It excludes the root command (name == "") from the count.
func (a *App) hasSubcommands() bool {
	for name := range a.commands {
		if name != "" {
			return true
		}
	}
	return len(a.groups) > 0
}

// renderAppHelp renders the main application help
func (a *App) renderAppHelp() tui.View {
	theme := a.getHelpTheme()
	hasSubcmds := a.hasSubcommands()

	// Build usage view based on whether we have subcommands and a root command with args
	usageView := a.buildUsageView(hasSubcmds, theme)

	// Get root command flags (always include for root-only apps, or when root command has flags)
	var rootFlags []Flag
	if rootCmd := a.commands[""]; rootCmd != nil {
		rootFlags = rootCmd.flags
	}

	views := []tui.View{
		renderHeader(a.name, a.description, a.version, theme),
	}
	if a.longDesc != "" {
		views = append(views, tui.Text("%s", a.longDesc))
	}
	views = append(views, tui.Stack(
		renderSection("USAGE", theme),
		usageView,
	))
	if len(a.examples) > 0 {
		views = append(views, tui.Stack(
			renderSection("EXAMPLES", theme),
			renderExamples(a.examples, theme),
		))
	}
	if len(a.commands) > 0 && hasSubcmds {
		views = append(views, tui.Stack(
			renderSection("COMMANDS", theme),
			renderOrderedCommands(a.commands, a.commandOrder, theme),
		))
	}

	// Flat-routed groups appear as their own named sections
	for _, name := range a.groupOrder {
		group := a.groups[name]
		if group == nil || !group.flatRouting {
			continue
		}
		views = append(views, tui.Stack(
			renderSection(strings.ToUpper(name), theme),
			renderOrderedCommands(group.commands, group.commandOrder, theme),
		))
	}

	// Non-flat-routed groups appear in COMMAND GROUPS
	if a.hasNonFlatGroups() {
		views = append(views, tui.Stack(
			renderSection("COMMAND GROUPS", theme),
			renderFilteredGroups(a.groups, a.groupOrder, false, theme),
		))
	}

	if len(rootFlags) > 0 {
		views = append(views, tui.Stack(
			renderSection("FLAGS", theme),
			renderFlags(rootFlags, theme),
		))
	}
	if len(a.globalFlags) > 0 {
		views = append(views, tui.Stack(
			renderSection("GLOBAL FLAGS", theme),
			renderFlags(a.globalFlags, theme),
		))
	}
	if hasSubcmds {
		views = append(views, renderFooter(a.name, theme))
	}

	return tui.Stack(views...).Gap(1)
}

// hasNonFlatGroups returns true if any group does not use flat routing.
func (a *App) hasNonFlatGroups() bool {
	for _, g := range a.groups {
		if !g.flatRouting {
			return true
		}
	}
	return false
}

// buildUsageView builds the usage section view, showing dual usage lines when
// the app has both a root command with args and subcommands.
func (a *App) buildUsageView(hasSubcmds bool, theme HelpTheme) tui.View {
	if !hasSubcmds {
		return tui.Text("%s", a.buildRootUsageString())
	}

	rootCmd := a.commands[""]
	hasRootWithArgs := rootCmd != nil && (len(rootCmd.args) > 0 || rootCmd.handler != nil)

	if !hasRootWithArgs {
		return tui.Text("  %s <command> [flags] [args]", a.name)
	}

	// Show both usage lines: root command and subcommand
	rootUsage := a.buildRootUsageString()
	rootSummary := ""
	if rootCmd != nil && rootCmd.usageSummary != "" {
		rootSummary = rootCmd.usageSummary
	}
	subcmdUsage := fmt.Sprintf("  %s <command> [flags]", a.name)

	views := []tui.View{}
	if rootSummary != "" {
		views = append(views, tui.Group(
			tui.Text("%s", rootUsage),
			tui.Text("  %s", rootSummary).Style(theme.Hint),
		))
	} else {
		views = append(views, tui.Text("%s", rootUsage))
	}
	views = append(views, tui.Group(
		tui.Text("%s", subcmdUsage),
		tui.Text("  Run a subcommand").Style(theme.Hint),
	))

	return tui.Stack(views...).Gap(0)
}

// buildRootUsageString builds the usage string for the root command.
func (a *App) buildRootUsageString() string {
	usage := "  " + a.name
	rootCmd := a.commands[""]
	hasFlags := len(a.globalFlags) > 0 || (rootCmd != nil && len(rootCmd.flags) > 0)
	if hasFlags {
		usage += " [flags]"
	}
	// Check for root command args
	if rootCmd != nil {
		for _, arg := range rootCmd.args {
			if arg.Required {
				usage += " <" + arg.Name + ">"
			} else {
				usage += " [" + arg.Name + "]"
			}
		}
	} else {
		for _, arg := range a.args {
			if arg.Required {
				usage += " <" + arg.Name + ">"
			} else {
				usage += " [" + arg.Name + "]"
			}
		}
	}
	return usage
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

// buildUsageString constructs the usage string for a command.
// For commands in flat-routed groups, it returns both the flat form
// (e.g., "app resize") and the grouped form (e.g., "app transform resize").
func buildUsageString(c *Command) string {
	suffix := ""
	if len(c.flags) > 0 || len(c.app.globalFlags) > 0 {
		suffix += " [flags]"
	}
	for _, arg := range c.args {
		if arg.Required {
			suffix += " <" + arg.Name + ">"
		} else {
			suffix += " [" + arg.Name + "]"
		}
	}

	if c.group != nil && c.group.flatRouting {
		flat := "  " + c.app.name + " " + c.name + suffix
		grouped := "  " + c.app.name + " " + c.group.name + " " + c.name + suffix
		return flat + "\n" + grouped
	}

	usage := "  " + c.app.name
	if c.group != nil {
		usage += " " + c.group.name
	}
	usage += " " + c.name + suffix
	return usage
}

// renderHeader creates the styled app header with gradient title
func renderHeader(name, description, version string, theme HelpTheme) tui.View {
	titleLine := renderGradientText(name, theme.TitleStart, theme.TitleEnd)
	if description != "" {
		parts := []tui.View{
			titleLine,
			tui.Text(" - "),
			tui.Text("%s", description),
		}
		if version != "" {
			parts = append(parts, tui.Text(" (v%s)", version).Style(theme.Hint))
		}
		titleLine = tui.Group(parts...)
	} else if version != "" {
		titleLine = tui.Group(titleLine, tui.Text(" (v%s)", version).Style(theme.Hint))
	}

	return titleLine
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

// renderCommands renders the command list as a Stack (alphabetical order, for maps without order tracking).
func renderCommands(commands map[string]*Command, theme HelpTheme) tui.View {
	names := sortedKeys(commands)
	return renderCommandList(commands, names, theme)
}

// renderOrderedCommands renders the command list in insertion order.
func renderOrderedCommands(commands map[string]*Command, order []string, theme HelpTheme) tui.View {
	if len(order) == 0 {
		return renderCommands(commands, theme)
	}
	return renderCommandList(commands, order, theme)
}

// renderCommandList renders commands in the given name order.
func renderCommandList(commands map[string]*Command, names []string, theme HelpTheme) tui.View {
	width := maxVisibleCommandNameLen(commands, names)
	views := make([]tui.View, 0, len(names))
	for _, name := range names {
		cmd := commands[name]
		if cmd == nil || cmd.hidden || name == "" {
			continue
		}
		views = append(views, tui.Group(
			tui.Text("  %-*s  ", width, name).Style(theme.Command),
			tui.Text("%s", cmd.description),
		))
	}
	return tui.Stack(views...).Gap(0)
}

// maxVisibleCommandNameLen returns the longest non-hidden command name length,
// with a sensible minimum to avoid overly narrow columns.
func maxVisibleCommandNameLen(commands map[string]*Command, names []string) int {
	max := 14
	for _, name := range names {
		cmd := commands[name]
		if cmd == nil || cmd.hidden || name == "" {
			continue
		}
		if len(name) > max {
			max = len(name)
		}
	}
	return max
}

// renderGroups renders the command groups list as a Stack (alphabetical order).
func renderGroups(groups map[string]*Group, theme HelpTheme) tui.View {
	names := sortedGroupKeys(groups)
	return renderGroupList(groups, names, theme)
}

// renderOrderedGroups renders the command groups in insertion order.
func renderOrderedGroups(groups map[string]*Group, order []string, theme HelpTheme) tui.View {
	if len(order) == 0 {
		return renderGroups(groups, theme)
	}
	return renderGroupList(groups, order, theme)
}

// renderFilteredGroups renders groups filtered by flat routing mode.
// If flat is true, only flat-routed groups are shown; if false, only non-flat groups.
func renderFilteredGroups(groups map[string]*Group, order []string, flat bool, theme HelpTheme) tui.View {
	if len(order) == 0 {
		order = sortedGroupKeys(groups)
	}
	groupWidth := maxVisibleGroupNameLen(groups, order, flat)
	groupBlocks := make([]tui.View, 0, len(order))
	anyExpanded := false
	for _, name := range order {
		group := groups[name]
		if group == nil || group.flatRouting != flat {
			continue
		}
		block := []tui.View{
			tui.Group(
				tui.Text("  %-*s  ", groupWidth, name).Style(theme.Command),
				tui.Text("%s", group.description),
			),
		}
		if group.isExpanded() {
			anyExpanded = true
			subOrder := group.commandOrder
			if len(subOrder) == 0 {
				subOrder = sortedKeys(group.commands)
			}
			subWidth := maxVisibleCommandNameLen(group.commands, subOrder)
			for _, subName := range subOrder {
				subCmd := group.commands[subName]
				if subCmd == nil || subCmd.hidden {
					continue
				}
				block = append(block, tui.Group(
					tui.Text("    %-*s  ", subWidth, subName).Style(theme.Flag),
					tui.Text("%s", subCmd.description).Style(theme.Hint),
				))
			}
		}
		groupBlocks = append(groupBlocks, tui.Stack(block...).Gap(0))
	}
	gap := 0
	if anyExpanded {
		gap = 1
	}
	return tui.Stack(groupBlocks...).Gap(gap)
}

// maxVisibleGroupNameLen returns the longest group name length among groups
// matching the requested flat-routing mode.
func maxVisibleGroupNameLen(groups map[string]*Group, order []string, flat bool) int {
	max := 14
	for _, name := range order {
		g := groups[name]
		if g == nil || g.flatRouting != flat {
			continue
		}
		if len(name) > max {
			max = len(name)
		}
	}
	return max
}

// renderGroupList renders groups in the given name order.
func renderGroupList(groups map[string]*Group, names []string, theme HelpTheme) tui.View {
	groupWidth := 14
	for _, name := range names {
		if g := groups[name]; g != nil && len(name) > groupWidth {
			groupWidth = len(name)
		}
	}

	groupBlocks := make([]tui.View, 0, len(names))
	anyExpanded := false
	for _, name := range names {
		group := groups[name]
		if group == nil {
			continue
		}
		block := []tui.View{
			tui.Group(
				tui.Text("  %-*s  ", groupWidth, name).Style(theme.Command),
				tui.Text("%s", group.description),
			),
		}

		if group.isExpanded() {
			anyExpanded = true
			// Subcommands in insertion order
			subOrder := group.commandOrder
			if len(subOrder) == 0 {
				subOrder = sortedKeys(group.commands)
			}
			subWidth := maxVisibleCommandNameLen(group.commands, subOrder)
			for _, subName := range subOrder {
				subCmd := group.commands[subName]
				if subCmd == nil || subCmd.hidden {
					continue
				}
				block = append(block, tui.Group(
					tui.Text("    %-*s  ", subWidth, subName).Style(theme.Flag),
					tui.Text("%s", subCmd.description).Style(theme.Hint),
				))
			}
		}
		groupBlocks = append(groupBlocks, tui.Stack(block...).Gap(0))
	}

	gap := 0
	if anyExpanded {
		gap = 1
	}
	return tui.Stack(groupBlocks...).Gap(gap)
}

// renderExamples renders the examples section as a Stack.
func renderExamples(examples []Example, theme HelpTheme) tui.View {
	views := make([]tui.View, len(examples))
	for i, ex := range examples {
		views[i] = tui.Stack(
			tui.Text("  %s", ex.Description).Style(theme.Hint),
			tui.Text("  $ %s", ex.Command).Style(theme.Command),
		).Gap(0)
	}
	return tui.Stack(views...).Gap(1)
}

// renderFlags renders the flags list as a Stack
func renderFlags(flags []Flag, theme HelpTheme) tui.View {
	views := make([]tui.View, 0, len(flags))

	// Calculate the maximum flag name length for alignment
	maxNameLen := 0
	for _, f := range flags {
		if f.IsHidden() {
			continue
		}
		if len(f.GetName()) > maxNameLen {
			maxNameLen = len(f.GetName())
		}
	}

	for _, f := range flags {
		if f.IsHidden() {
			continue
		}
		views = append(views, renderFlag(f, theme, maxNameLen))
	}

	return tui.Stack(views...).Gap(0)
}

// renderFlag renders a single flag with dynamic width alignment
func renderFlag(f Flag, theme HelpTheme, maxNameLen int) tui.View {
	// Build flag prefix
	prefix := "  "
	if f.GetShort() != "" {
		prefix += "-" + f.GetShort() + ", "
	} else {
		prefix += "    "
	}
	prefix += fmt.Sprintf("--%-*s", maxNameLen, f.GetName())

	// Build metadata
	meta := buildFlagMeta(f)

	return tui.Group(
		tui.Text("%s", prefix).Style(theme.Flag),
		tui.Text(" "),
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
