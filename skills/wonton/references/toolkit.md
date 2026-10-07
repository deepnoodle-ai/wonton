# toolkit — configuration, resilience, git, and the small utilities

Covers `env`, `retry`, `schema`, `git`, `unidiff`, `humanize`, `strs`, `ptr`, `color`,
`runewidth`, `clipboard`, `terminal`, `termsession`, `gif`, `thumbnail`.

## env — configuration

```go
type Config struct {
	Host    string        `env:"HOST" default:"0.0.0.0"`
	Port    int           `env:"PORT" default:"8080"`
	Timeout time.Duration `env:"TIMEOUT" default:"30s"`
	DBName  string        `env:"DB_NAME,required"`
	DBPass  string        `env:"DB_PASS,notEmpty"`
	Hosts   []string      `env:"ALLOWED_HOSTS"`                     // comma-separated
	DataDir string        `env:"DATA_DIR,expand" default:"$HOME/data"`
	DB      struct {
		Host string `env:"HOST" default:"localhost"`
	} `envPrefix:"DB_"`
}

cfg, err := env.Parse[Config](
	env.WithPrefix("MYAPP"),                  // reads MYAPP_HOST, MYAPP_PORT, …
	env.WithEnvFile(".env", ".env.local"),
	env.WithJSONFile("config.json"),
)
```

`env.Must[Config](...)` panics instead of returning an error; `env.ParseInto(&cfg, opts...)` fills
an existing value. Other options: `WithStage`, `WithEnvironment`, `WithTagName`, `WithParser[T]`,
`WithRequiredIfNoDefault`, `WithUseFieldName`, `WithOnSet`, `WithRequireConfigFile`.

## retry — backoff

```go
data, err := retry.Do(ctx, func() (string, error) { return fetchData() },
	retry.WithMaxAttempts(5),
	retry.WithBackoff(time.Second, 30*time.Second),
	retry.WithJitter(0.1),
)

err = retry.DoSimple(ctx, func() error { return send() }, retry.WithMaxAttempts(3))
```

Stop early by returning `retry.MarkPermanent(err)`; test with `retry.IsPermanent(err)`. Shape the
delay with `WithConstantBackoff`, `WithLinearBackoff`, `WithFullJitter`, `WithBackoffMultiplier`,
or a custom `WithDelayFunc`. `WithRetryIf(fn)` filters which errors retry; `WithOnRetry(fn)`
observes each attempt. The context cancels the loop.

## schema — JSON Schema for LLM tools

```go
type SearchParams struct {
	Query  string   `json:"query" description:"Search query"`
	Limit  int      `json:"limit,omitempty" description:"Max results" minimum:"1" maximum:"100"`
	Tags   []string `json:"tags,omitempty" description:"Filter by tags"`
	Format string   `json:"format" enum:"json,csv,text" default:"json"`
}

s, err := schema.Generate(SearchParams{})
```

The result marshals straight into an Anthropic or OpenAI tool definition. `schema.Generate` also
takes `GenerateOptions` for strict-mode output.

## git — read-only repository access

```go
repo, err := git.Open(".")

branch, err := repo.CurrentBranch(ctx)
status, err := repo.Status(ctx)                  // .IsClean, .Branch, .Staged, .Unstaged
commits, err := repo.Log(ctx, git.LogOptions{Limit: 50, Author: "alice", Path: "src/"})
diff, err := repo.Diff(ctx, git.DiffOptions{From: "HEAD~3", To: "HEAD", IncludePatch: true})
```

`Commit` carries `.Hash`, `.ShortHash`, `.Subject`, `.Body`, `.Author.Name`, `.Timestamp`. `Diff`
carries `.Files`, `.TotalAdded`, `.TotalRemoved`; each `DiffFile` has `.Path`, `.Status`,
`.Additions`, `.Deletions`, `.Patch`. Also available: `Show`, `ShowFile`, `DiffFile`, `Blame`,
`Branches`, `LocalBranches`, `RemoteBranches`, `DefaultBranch`, `BranchExists`, `TrackedFiles`,
`UntrackedFiles`, `ModifiedFiles`, `IgnoredFiles`, `FileExists`, `Config`, `User`. Every call takes
a `context.Context`. Nothing here mutates the repo.

## unidiff — parse patches

```go
diff, err := unidiff.Parse(patchText)
for _, file := range diff.Files {
	for _, hunk := range file.Hunks {
		for _, line := range hunk.Lines {
			// line.Type is unidiff.LineAdded, LineRemoved, or LineContext
		}
	}
}
```

Pairs with `tui.DiffView(diff, lang, &scrollY)` for rendering.

## humanize, strs, ptr

```go
humanize.Bytes(1536)                         // "1.5 KiB"        (BytesSI for kB/MB)
humanize.Duration(90 * time.Second)          // "1m 30s"         (DurationShort for "1m")
humanize.Time(t)                             // "2 hours ago"    (RelativeTime(t, ref))
humanize.Number(1234567)                     // "1,234,567"
humanize.Percentage(3, 4)                    // "75.0%"
humanize.Ordinal(3)                          // "3rd"
humanize.PluralWord(5, "item", "items")      // "5 items"
humanize.Truncate("Hello, World!", 8)        // "Hello..."

strs.FirstNonEmpty(flagVal, envVal, "default")
strs.FirstNonBlankTrim(a, b)                 // ignores whitespace-only values
strs.Dedupe(tags)                            // order-preserving

ptr.To("value")                              // *string
ptr.Deref(p)                                 // zero value when nil
ptr.Or(p, "fallback")
ptr.IfNotZero(count)                          // nil when count == 0 — good for optional JSON
```

## color and runewidth

```go
fmt.Println(color.Red.Apply("Error"))                    // also ApplyBg, ApplyBold, ApplyDim
rgb := color.NewRGB(255, 128, 0)
fmt.Println(rgb.Apply("Orange"))                         // foreground; rgb.ApplyBg for background
stops := color.Gradient(color.NewRGB(255, 0, 0), color.NewRGB(0, 0, 255), 10)
if color.ShouldColorize(os.Stdout) { /* terminal supports color */ }
```

`runewidth` measures what a terminal actually draws: `StringWidth(s)`, `RuneWidth(r)`,
`Truncate(s, w, tail)`, `Fit`, `FitLeft`, `FitRight`, `WidthIndex(s)`, and `Graphemes(s)` (an
`iter.Seq2[string, int]` over grapheme clusters). Use it instead of `len()` or `utf8.RuneCount`
whenever you align columns — emoji, ZWJ sequences, and CJK are all wider than their rune count.

## terminal, clipboard, recordings

- `terminal` — raw mode, key/mouse decoding (`NewKeyDecoder`), OSC 8 hyperlinks
  (`terminal.Format(url, text)`), capability probing. `tui` sits on top of it; drop down only when
  you are not using `tui`.
- `clipboard` — `Available()`, `Read()`, `Write(s)`, `Clear()`, plus `*WithTimeout` and `*Context`
  variants. Always check `Available()` first.
- `termsession` — record to asciinema v2: `NewRecorder(path, w, h, opts)` for manual capture,
  `NewSession(SessionOptions{Command: ...})` to capture a child process, `NewPlayer` to replay,
  `LoadCastFile` to analyze.
- `gif` — `gif.New(w, h)` plus `AddFrame`/`AddFrameWithDelay` for programmatic animation, or
  `gif.RenderCast("session.cast", gif.DefaultCastOptions())` to turn a recording into a GIF.
- `thumbnail` — `thumbnail.Render(req)` produces preview images for files, including synthetic
  cards for text and code.

## Gotchas

- `env` struct tags are `env:"NAME,required"` / `,notEmpty` / `,expand`; nested structs need
  `envPrefix`.
- `retry.Do` is generic over the result type; `DoSimple` is the no-result form.
- `Apply` sets the foreground and takes only the text. Background is a separate `ApplyBg`.
- `git` never writes. Shell out for commits, pushes, or checkouts.
