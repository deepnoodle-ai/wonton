# cli — commands, flags, context, middleware

Fluent builder for command-line apps. Integrates with `tui` for interactive modes and with `env`
for configuration. Full docs: [`cli/README.md`][readme].

[readme]: https://github.com/deepnoodle-ai/wonton/blob/main/cli/README.md

## Shape of an app

```go
app := cli.New("myapp").
	Description("Short one-liner shown in help").
	Long("Extended prose shown after usage").
	Version("1.0.0").
	AddCompletionCommand()            // adds `myapp completion bash|zsh|fish`

app.GlobalFlags(cli.Bool("verbose", "v").Help("Verbose output"))
app.Use(cli.Recover())                // middleware for every command

err := app.Execute()                  // or ExecuteArgs([]string), ExecuteContext(ctx, args)
```

Three layouts:

- **Subcommands**: `app.Command("build")`, grouped with `app.Group("users").Command("create")`.
- **Single command**: `app.Main().Args("urls...").Flags(...).Run(handler)`.
- **Both**: a root handler plus subcommands; help shows dual usage lines automatically.

Commands and groups appear in help in registration order, not alphabetically.

## Commands

```go
app.Command("deploy").
	Description("Deploy the app").
	Alias("d").
	Args("env", "target?").           // see the arg-spec table below
	Flags(
		cli.String("region", "r").Default("us-east-1").Enum("us-east-1", "eu-west-1"),
		cli.Duration("timeout").Default(30*time.Second).Help("Deadline"),
		cli.Strings("tag", "t").Help("Repeatable: -t a -t b"),
	).
	Validate(func(ctx *cli.Context) error { return nil }).   // runs before the handler
	Use(cli.ConfirmBefore("Really deploy?")).
	Run(func(ctx *cli.Context) error { return nil })
```

Other command builders: `.Long()`, `.UsageSummary()`, `.Hidden()`, `.Deprecated(msg)`,
`.OmitGlobalFlag(names...)`, `.AddArg(&cli.Arg{...})`.

### Positional arg specs

`Args` declares slots; the parser enforces them. There is no separate arg-count validator.

| Spec | Meaning |
| --- | --- |
| `"name"` | required |
| `"name?"` | optional |
| `"name..."` | variadic, one or more (must be last) |
| `"name?..."` | variadic, zero or more (must be last) |

Extra positionals are rejected unless the last slot is variadic. An invalid layout — a duplicate
name, a variadic slot that is not last, a required slot after an optional one — **panics at
registration**, so a bad declaration fails the moment the program starts. Use `.Validate(fn)` for
anything beyond arity.

## Flags

Builders: `cli.String`, `cli.Bool`, `cli.Int`, `cli.Int64`, `cli.Float64`, `cli.Duration`,
`cli.Strings`, `cli.Ints`. The short name is optional: `cli.Int("times", "t")` or
`cli.Int("times")`. Every builder supports `.Default(v)` (typed — `Duration` takes a
`time.Duration`), `.Help(t)`, `.Env("VAR")`, `.Required()`, `.Hidden()`. String flags add
`.Enum(values...)` and `.ValidateWith(fn)`.

Struct-based flags keep large flag sets readable:

```go
type runFlags struct {
	Model  string  `flag:"model,m" default:"claude-sonnet" help:"Model" env:"MYAPP_MODEL"`
	Temp   float64 `flag:"temperature,t" default:"0.7" help:"Sampling temperature"`
	Format string  `flag:"format,f" enum:"text,json" default:"text" help:"Output format"`
}

cmd := app.Command("run").Args("prompt")
cli.ParseFlags[runFlags](cmd)                     // registers the flags
cmd.Run(func(ctx *cli.Context) error {
	f, err := cli.BindFlags[runFlags](ctx)        // reads them back into the struct
	if err != nil {
		return err
	}
	ctx.Printf("model=%s temp=%.2f\n", f.Model, f.Temp)
	return nil
})
```

Global flags may appear before or after the command name. `--` stops flag parsing, but command
names are still resolved after it. Negative numbers (`--count -5`) are read as values, not flags.
When a flag value could be mistaken for a command name, use `--flag=value`.

## Context

Reading input: `ctx.Args()`, `ctx.Arg(i)`, `ctx.NArg()`, `ctx.String`, `ctx.Strings`, `ctx.Int`,
`ctx.Ints`, `ctx.Int64`, `ctx.Float64`, `ctx.Bool`, `ctx.Duration`, `ctx.IsSet(name)`, and
`ctx.Context()` for the `context.Context`.

Writing output: `ctx.Print/Printf/Println` (stdout), `ctx.Error/Errorf/Errorln` (stderr), and the
semantic helpers `ctx.Success`, `ctx.Fail`, `ctx.Warn`, `ctx.Info`. Raw streams are
`ctx.Stdin()`, `ctx.Stdout()`, `ctx.Stderr()` — use those instead of `os.Stdout` so tests can
capture output.

Prompts (they need a TTY): `ctx.Select(title, options...)`, `ctx.SelectString(...)`,
`ctx.Input(prompt)`, `ctx.Confirm(message)`.

## Progressive interactivity

The same command can behave differently when piped:

```go
app.Command("process").
	Args("files...").
	Interactive(func(ctx *cli.Context) error { return runTUI(ctx.Args()) }).
	NonInteractive(func(ctx *cli.Context) error { return runBatch(ctx.Args()) })
```

`Interactive`/`NonInteractive` are selected by TTY detection. A single `Run` handler can branch on
`ctx.Interactive()` instead. `app.ForceInteractive(true)` pins the choice in tests.

## Middleware

```go
func timing() cli.Middleware {
	return func(next cli.Handler) cli.Handler {
		return func(ctx *cli.Context) error {
			start := time.Now()
			err := next(ctx)
			ctx.Info("took %s", time.Since(start))
			return err
		}
	}
}
```

Built-ins: `cli.Recover()`, `cli.RequireFlags(names...)`, `cli.ConfirmBefore(msg)` (interactive
only), `cli.Before(fn)`, `cli.After(fn)`. App middleware wraps command middleware:
`A → B → C → D → handler`.

## Errors and exit codes

```go
return cli.Error("config not found").Hint("run `myapp init` first")
return cli.Errorf("upload to %s failed", bucket).Wrap(err).Code("ERR_UPLOAD")
return cli.Exit(2)                        // specific exit status
```

At the top level, `cli.IsHelpRequested(err)` reports the `--help` path (exit 0) and
`cli.GetExitCode(err)` yields the status. Both use `errors.As`, so wrapped errors work.
`app.PrintError(err)` prints a colorized `Error: ...` line.

## Gotchas

- Returning a bare `error` from a handler is fine, but `cli.Error(...)` gives users a hint line.
- `ctx.Println` writes to the app's configured writer; `fmt.Println` bypasses it and breaks tests.
- Prompt helpers fail without a TTY — guard them with `ctx.Interactive()`.
- Prefer `app.Main()` over a root `Run` when the tool has no subcommands: help output is clearer.

See [testing.md](testing.md) for `app.Test(t, cli.TestArgs(...))` and I/O capture.
