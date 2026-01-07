package cli

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/color"
)

func TestDefaultHelpTheme(t *testing.T) {
	theme := DefaultHelpTheme()

	assert.Equal(t, color.NewRGB(80, 140, 255), theme.TitleStart)
	assert.Equal(t, color.NewRGB(80, 140, 255), theme.TitleEnd)
}

func TestGetHelpTheme(t *testing.T) {
	t.Run("returns custom theme when set", func(t *testing.T) {
		app := New("test")
		customTheme := DefaultHelpTheme()
		customTheme.TitleStart = color.NewRGB(255, 0, 0)
		app.HelpTheme(customTheme)

		theme := app.getHelpTheme()
		assert.Equal(t, color.NewRGB(255, 0, 0), theme.TitleStart)
	})

	t.Run("returns default theme when not set", func(t *testing.T) {
		app := New("test")
		theme := app.getHelpTheme()

		assert.Equal(t, color.NewRGB(80, 140, 255), theme.TitleStart)
	})
}

func TestHasSubcommands(t *testing.T) {
	t.Run("returns false for app with only root command", func(t *testing.T) {
		app := New("test")
		app.Run(func(ctx *Context) error {
			return nil
		})

		assert.False(t, app.hasSubcommands())
	})

	t.Run("returns true for app with subcommands", func(t *testing.T) {
		app := New("test")
		app.Command("build").Description("Build the project").Run(func(ctx *Context) error {
			return nil
		})

		assert.True(t, app.hasSubcommands())
	})

	t.Run("returns true for app with groups", func(t *testing.T) {
		app := New("test")
		group := app.Group("config").Description("Configuration commands")
		group.Command("get").Description("Get config").Run(func(ctx *Context) error {
			return nil
		})

		assert.True(t, app.hasSubcommands())
	})
}

func TestRenderAppHelp(t *testing.T) {
	t.Run("renders help for app without subcommands", func(t *testing.T) {
		app := New("myapp")
		app.Description("A test application")
		app.Version("1.0.0")
		app.Run(func(ctx *Context) error {
			return nil
		})

		view := app.renderAppHelp()
		assert.NotNil(t, view)
	})

	t.Run("renders help for app with subcommands", func(t *testing.T) {
		app := New("myapp")
		app.Description("A test application")
		app.Command("build").Description("Build the project").Run(func(ctx *Context) error {
			return nil
		})
		app.Command("test").Description("Run tests").Run(func(ctx *Context) error {
			return nil
		})

		view := app.renderAppHelp()
		assert.NotNil(t, view)
	})

	t.Run("renders help with global flags", func(t *testing.T) {
		app := New("myapp")
		app.AddGlobalFlag(String("config", "c").Help("Config file"))

		view := app.renderAppHelp()
		assert.NotNil(t, view)
	})

	t.Run("renders help with root command flags", func(t *testing.T) {
		app := New("myapp")
		cmd := app.Main()
		cmd.Flags(String("output", "o").Help("Output file"))

		view := app.renderAppHelp()
		assert.NotNil(t, view)
	})
}

func TestBuildRootUsageString(t *testing.T) {
	t.Run("builds usage for app with no args or flags", func(t *testing.T) {
		app := New("myapp")
		usage := app.buildRootUsageString()

		assert.Contains(t, usage, "myapp")
	})

	t.Run("includes flags when present", func(t *testing.T) {
		app := New("myapp")
		app.AddGlobalFlag(String("verbose", "v").Help("Verbose output"))
		usage := app.buildRootUsageString()

		assert.Contains(t, usage, "[flags]")
	})

	t.Run("includes required args", func(t *testing.T) {
		app := New("myapp")
		app.Args("input")
		usage := app.buildRootUsageString()

		assert.Contains(t, usage, "<input>")
	})

	t.Run("includes optional args", func(t *testing.T) {
		app := New("myapp")
		app.Args("output?")
		usage := app.buildRootUsageString()

		assert.Contains(t, usage, "[output]")
	})
}

func TestRenderCommandHelp(t *testing.T) {
	t.Run("renders help for simple command", func(t *testing.T) {
		app := New("myapp")
		cmd := app.Command("build").Description("Build the project")
		cmd.Run(func(ctx *Context) error {
			return nil
		})

		view := cmd.renderCommandHelp()
		assert.NotNil(t, view)
	})

	t.Run("renders help for command with long description", func(t *testing.T) {
		app := New("myapp")
		cmd := app.Command("build").Description("Build the project")
		cmd.Long("This command builds the entire project from source.")
		cmd.Run(func(ctx *Context) error {
			return nil
		})

		view := cmd.renderCommandHelp()
		assert.NotNil(t, view)
	})

	t.Run("renders help for deprecated command", func(t *testing.T) {
		app := New("myapp")
		cmd := app.Command("old").Description("Deprecated command")
		cmd.Deprecated("Use 'new' instead")
		cmd.Run(func(ctx *Context) error {
			return nil
		})

		view := cmd.renderCommandHelp()
		assert.NotNil(t, view)
	})

	t.Run("renders help for command with aliases", func(t *testing.T) {
		app := New("myapp")
		cmd := app.Command("build").Description("Build the project")
		cmd.Aliases("b", "compile")
		cmd.Run(func(ctx *Context) error {
			return nil
		})

		view := cmd.renderCommandHelp()
		assert.NotNil(t, view)
	})

	t.Run("renders help for command with args", func(t *testing.T) {
		app := New("myapp")
		cmd := app.Command("deploy").Description("Deploy to environment")
		cmd.Args("env")
		cmd.Run(func(ctx *Context) error {
			return nil
		})

		view := cmd.renderCommandHelp()
		assert.NotNil(t, view)
	})

	t.Run("renders help for command with flags", func(t *testing.T) {
		app := New("myapp")
		cmd := app.Command("build").Description("Build the project")
		cmd.Flags(
			String("output", "o").Help("Output directory"),
			Bool("verbose", "v").Help("Verbose output"),
		)
		cmd.Run(func(ctx *Context) error {
			return nil
		})

		view := cmd.renderCommandHelp()
		assert.NotNil(t, view)
	})

	t.Run("renders help for group command", func(t *testing.T) {
		app := New("myapp")
		group := app.Group("config").Description("Configuration")
		cmd := group.Command("get").Description("Get config value")
		cmd.Run(func(ctx *Context) error {
			return nil
		})

		view := cmd.renderCommandHelp()
		assert.NotNil(t, view)
	})
}

func TestBuildUsageString(t *testing.T) {
	t.Run("builds usage for simple command", func(t *testing.T) {
		app := New("myapp")
		cmd := app.Command("build").Description("Build")
		cmd.Run(func(ctx *Context) error {
			return nil
		})

		usage := buildUsageString(cmd)
		assert.Contains(t, usage, "myapp build")
	})

	t.Run("includes flags when present", func(t *testing.T) {
		app := New("myapp")
		cmd := app.Command("build").Description("Build")
		cmd.Flags(String("output", "o").Help("Output"))
		cmd.Run(func(ctx *Context) error {
			return nil
		})

		usage := buildUsageString(cmd)
		assert.Contains(t, usage, "[flags]")
	})

	t.Run("includes args", func(t *testing.T) {
		app := New("myapp")
		cmd := app.Command("deploy").Description("Deploy")
		cmd.Args("env", "version?")
		cmd.Run(func(ctx *Context) error {
			return nil
		})

		usage := buildUsageString(cmd)
		assert.Contains(t, usage, "<env>")
		assert.Contains(t, usage, "[version]")
	})

	t.Run("includes group name", func(t *testing.T) {
		app := New("myapp")
		group := app.Group("config").Description("Configuration")
		cmd := group.Command("get").Description("Get")
		cmd.Run(func(ctx *Context) error {
			return nil
		})

		usage := buildUsageString(cmd)
		assert.Contains(t, usage, "config get")
	})
}

func TestRenderHeader(t *testing.T) {
	theme := DefaultHelpTheme()

	t.Run("renders header with name only", func(t *testing.T) {
		view := renderHeader("myapp", "", "", theme)
		assert.NotNil(t, view)
	})

	t.Run("renders header with description", func(t *testing.T) {
		view := renderHeader("myapp", "A test app", "", theme)
		assert.NotNil(t, view)
	})

	t.Run("renders header with version", func(t *testing.T) {
		view := renderHeader("myapp", "A test app", "1.0.0", theme)
		assert.NotNil(t, view)
	})
}

func TestRenderCommandHeader(t *testing.T) {
	theme := DefaultHelpTheme()

	t.Run("renders header with name only", func(t *testing.T) {
		view := renderCommandHeader("build", "", theme)
		assert.NotNil(t, view)
	})

	t.Run("renders header with description", func(t *testing.T) {
		view := renderCommandHeader("build", "Build the project", theme)
		assert.NotNil(t, view)
	})
}

func TestRenderSection(t *testing.T) {
	theme := DefaultHelpTheme()
	view := renderSection("COMMANDS", theme)
	assert.NotNil(t, view)
}

func TestRenderGradientText(t *testing.T) {
	start := color.NewRGB(255, 0, 0)
	end := color.NewRGB(0, 0, 255)

	t.Run("renders text with gradient", func(t *testing.T) {
		view := renderGradientText("hello", start, end)
		assert.NotNil(t, view)
	})

	t.Run("handles empty text", func(t *testing.T) {
		view := renderGradientText("", start, end)
		assert.NotNil(t, view)
	})
}

func TestRenderCommands(t *testing.T) {
	theme := DefaultHelpTheme()

	t.Run("renders command list", func(t *testing.T) {
		commands := map[string]*Command{
			"build": {name: "build", description: "Build the project"},
			"test":  {name: "test", description: "Run tests"},
		}

		view := renderCommands(commands, theme)
		assert.NotNil(t, view)
	})

	t.Run("skips hidden commands", func(t *testing.T) {
		commands := map[string]*Command{
			"build":  {name: "build", description: "Build the project"},
			"secret": {name: "secret", description: "Secret command", hidden: true},
		}

		view := renderCommands(commands, theme)
		assert.NotNil(t, view)
	})
}

func TestRenderGroups(t *testing.T) {
	theme := DefaultHelpTheme()

	groups := map[string]*Group{
		"config": {
			name:        "config",
			description: "Configuration commands",
			commands: map[string]*Command{
				"get": {name: "get", description: "Get value"},
				"set": {name: "set", description: "Set value"},
			},
		},
	}

	view := renderGroups(groups, theme)
	assert.NotNil(t, view)
}

func TestRenderFlags(t *testing.T) {
	theme := DefaultHelpTheme()

	flags := []Flag{
		String("output", "o").Help("Output file"),
		Bool("verbose", "v").Help("Verbose output"),
		Int("count", "c").Default(10).Help("Item count"),
	}

	view := renderFlags(flags, theme)
	assert.NotNil(t, view)
}

func TestRenderFlag(t *testing.T) {
	theme := DefaultHelpTheme()

	t.Run("renders flag with short name", func(t *testing.T) {
		flag := String("output", "o").Help("Output file")
		view := renderFlag(flag, theme, 10)
		assert.NotNil(t, view)
	})

	t.Run("renders flag without short name", func(t *testing.T) {
		flag := String("output", "").Help("Output file")
		view := renderFlag(flag, theme, 10)
		assert.NotNil(t, view)
	})
}

func TestBuildFlagMeta(t *testing.T) {
	t.Run("includes default value", func(t *testing.T) {
		flag := String("output", "o").Default("out.txt")
		meta := buildFlagMeta(flag)
		assert.Contains(t, meta, "default:")
		assert.Contains(t, meta, "out.txt")
	})

	t.Run("includes required", func(t *testing.T) {
		flag := String("api-key", "").Required()
		meta := buildFlagMeta(flag)
		assert.Contains(t, meta, "required")
	})

	t.Run("includes enum values", func(t *testing.T) {
		flag := String("level", "l").Help("Log level").Enum("debug", "info", "warn")
		meta := buildFlagMeta(flag)
		assert.Contains(t, meta, "debug")
		assert.Contains(t, meta, "info")
		assert.Contains(t, meta, "warn")
	})

	t.Run("returns empty for flag with no metadata", func(t *testing.T) {
		flag := String("name", "")
		meta := buildFlagMeta(flag)
		assert.Equal(t, "", meta)
	})

	t.Run("skips false bool default", func(t *testing.T) {
		flag := Bool("verbose", "")
		meta := buildFlagMeta(flag)
		assert.Equal(t, "", meta)
	})

	t.Run("skips zero int default", func(t *testing.T) {
		flag := Int("count", "")
		meta := buildFlagMeta(flag)
		assert.Equal(t, "", meta)
	})

	t.Run("skips empty string default", func(t *testing.T) {
		flag := String("name", "")
		meta := buildFlagMeta(flag)
		assert.Equal(t, "", meta)
	})
}

func TestRenderArgs(t *testing.T) {
	theme := DefaultHelpTheme()

	args := []*Arg{
		{Name: "input", Description: "Input file", Required: true},
		{Name: "output", Description: "Output file", Required: false},
	}

	view := renderArgs(args, theme)
	assert.NotNil(t, view)
}

func TestRenderFooter(t *testing.T) {
	theme := DefaultHelpTheme()
	view := renderFooter("myapp", theme)
	assert.NotNil(t, view)
}

func TestSortedKeys(t *testing.T) {
	commands := map[string]*Command{
		"zebra":  {name: "zebra"},
		"alpha":  {name: "alpha"},
		"middle": {name: "middle"},
	}

	keys := sortedKeys(commands)
	assert.Equal(t, 3, len(keys))
	assert.Equal(t, "alpha", keys[0])
	assert.Equal(t, "middle", keys[1])
	assert.Equal(t, "zebra", keys[2])
}

func TestSortedGroupKeys(t *testing.T) {
	groups := map[string]*Group{
		"zebra":  {name: "zebra"},
		"alpha":  {name: "alpha"},
		"middle": {name: "middle"},
	}

	keys := sortedGroupKeys(groups)
	assert.Equal(t, 3, len(keys))
	assert.Equal(t, "alpha", keys[0])
	assert.Equal(t, "middle", keys[1])
	assert.Equal(t, "zebra", keys[2])
}
