# tui — declarative terminal UI

Views are values. Your app returns a fresh view tree every frame; the runtime diffs and draws it.
Full docs: [`tui/README.md`][readme].

[readme]: https://github.com/deepnoodle-ai/wonton/blob/main/tui/README.md

## Three ways to render

| Mode | Interface | Entry point | Use for |
| --- | --- | --- | --- |
| Fullscreen | `View() View` | `tui.Run(app, opts...)` | dashboards, browsers, full TUIs |
| Inline | `LiveView() View` | `tui.NewInlineApp(...)`, `.Run(app)` | REPLs, chat, build logs |
| One-shot | none | `tui.Print(view)` / `tui.Sprint(view)` | styled output, no event loop |

Optional interfaces on any app: `EventHandler` (`HandleEvent(Event) []Cmd`), `Initializable`
(`Init() error`), `Destroyable` (`Destroy()`).

## Runtime and events

```go
tui.Run(app,
	tui.WithFPS(30),               // emits TickEvent; 0 = no ticks
	tui.WithMouseTracking(true),
	tui.WithAlternateScreen(true), // default true
	tui.WithBracketedPaste(true),
)
```

Events all satisfy `Event` (`Timestamp() time.Time`): `KeyEvent{Rune, Key, Shift}`, `MouseEvent`,
`TickEvent{Time, Frame}`, `TimerEvent{Tag}`, `ResizeEvent{Width, Height}`, `ErrorEvent{Err}`,
`QuitEvent`, `BatchEvent`. Custom events just need a `Timestamp()` method:

```go
type loadedEvent struct {
	Time time.Time
	Data []byte
}

func (e loadedEvent) Timestamp() time.Time { return e.Time }
```

Keys: `KeyArrowUp/Down/Left/Right`, `KeyEnter`, `KeyTab`, `KeyBackspace`, `KeyEscape`, `KeyDelete`,
`KeyInsert`, `KeyHome`, `KeyEnd`, `KeyPageUp`, `KeyPageDown`, `KeyF1`–`KeyF12`, `KeyCtrlA`–
`KeyCtrlZ`, `KeyUnknown`. For a printable character `Key == KeyUnknown` and `Rune != 0`.

**Key routing:** the focused widget sees keys first and consumes what it handles, so plain letters
never reach `HandleEvent` while an input has focus. `KeyCtrlC` is always delivered — bind quit to
it in any app with inputs.

## Async work

```go
type Cmd func() Event
```

Return `[]Cmd` from `HandleEvent`; the runtime runs each in its own goroutine and feeds the
returned event back through `HandleEvent`.

```go
func (a *app) load() tui.Cmd {
	url := a.url                                    // capture before the goroutine starts
	return func() tui.Event {
		body, err := get(url)
		return loadedEvent{Time: time.Now(), Data: body, Err: err}
	}
}
```

Helpers: `tui.Quit()`, `tui.After(d, tag)` (delivers `TimerEvent{Tag: tag}`), `tui.Batch(cmds...)`
(returns `[]Cmd`), `tui.Sequence(cmds...)` (returns one `Cmd`), `tui.None()`.

A `Cmd` goroutine may **read** fields set before it was returned, but must never **write** app
state. Send results back as events, or stream them with `runner.SendEvent(event)` on an InlineApp.

`Init()` returns only `error`. To start work at launch, set a flag and issue the command from the
first `TickEvent` (requires `WithFPS`), or start a goroutine that calls `SendEvent`.

## View catalog

Layout: `Stack` (vertical), `Group` (horizontal), `ZStack` (layered), `Spacer`, `Empty`,
`HeaderBar(text)`, `StatusBar(text)`, `Divider()`. Containers inherit flex from their children.

Sizing wrappers: `Width(w, v)`, `Height(h, v)`, `MaxWidth`, `MinWidth`, `MaxHeight`, `MinHeight`,
`Padding(n, v)`, `PaddingHV(h, v, inner)`, `Bordered(v)`, `Scroll(v, &scrollY)`.

Text: `Text(format, args...)`, `Markdown(content, &scrollY)`, `Code(src, lang)`,
`DiffView(diff, lang, &scrollY)`.

Inputs: `Input(&s)`, `InputField(&s)` (adds `.History`, `.OnComplete`, `.OnKey`, `.Multiline`),
`TextArea(&s)`. Give focusable views an `.ID("name")`.

Interactive: `Button(label, fn)`, `Clickable(label, fn)`, `PromptChoice(&selected, &text)`.

Data: `Table(columns, &selected)`, `Tree(root)`, `SelectList(items, &sel)`,
`FilterableList(items, &sel)`, `FilterableListStrings(labels, &sel)`, `CheckboxList`, `RadioList`,
`Progress(current, total)`, `Loading(frame)`.

Composition: `ForEach(items, fn)`, `HForEach(items, fn)`, `If(cond, v)`, `IfElse(cond, a, b)`,
`Switch(value, cases...)`, `Canvas(draw)`, `CanvasContext(draw)`.

List-shaped views share one key map when focused: arrows, PageUp/PageDown, Home/End, and vi
`j`/`k`/`g`/`G`. `FilterableList` routes letters to its filter, so vi keys are off there.

## Styling

```go
tui.Text("%s", name).Bold().Dim().Italic().Underline().Wrap().Center()
tui.Text("hi").Fg(tui.ColorGreen).Bg(tui.ColorBlack)   // 16 ANSI colors
tui.Text("hi").FgRGB(60, 160, 255)                     // true color
tui.Text("saved").Success()                            // also Error, Warning, Info, Muted, Hint
```

Animations attach with `.Animate(anim)`: `Rainbow(speed)` (`.Reverse()`, `.Length(n)`),
`Wave(speed, colors...)`, `Pulse(color, speed)`, `Slide(speed, base, hi)`, `Sparkle`,
`Typewriter`, `Glitch`. They animate from the render context — no frame argument. `Loading(frame)`
is the exception and needs a counter you update from `TickEvent`.

Animated progress bars need an RGB style, and the animation call comes last because it changes the
returned type:

```go
base := tui.NewStyle().WithFgRGB(tui.NewRGB(60, 160, 255))
var bar tui.View = tui.Progress(done, total).Width(40).Label("Sync: ").
	Style(base).Shimmer(tui.NewRGB(255, 255, 255), 3)
```

## InlineApp

`Print(view)` and `Printf(format, args...)` push content into scrollback above the live region;
`LiveView()` redraws in place. `Printf` emits plain text only — use `Print(view)` for anything
styled. `Print` renders statically, so animations belong in `LiveView()`.

```go
runner := tui.NewInlineApp(tui.WithInlineFPS(30))
err := runner.Run(app)      // app implements LiveView() tui.View
```

Options: `WithInlineWidth`, `WithInlineFPS`, `WithInlineMouseTracking`, `WithInlineBracketedPaste`,
`WithInlineKittyKeyboard`, `WithInlinePasteTabWidth`, `WithInlineBackslashEnter`.
`tui.RunInline(app, opts...)` is the one-liner form. InlineApp needs stdin to be a TTY.

## Gotchas

- `Text` is `Sprintf`-shaped: `tui.Text("%s", s)`, never `tui.Text(s)`.
- No mutexes. `View()`/`LiveView()` and `HandleEvent()` are called sequentially on one goroutine;
  locking around `Print()` deadlocks because `Print()` re-renders `LiveView()` synchronously.
- Settle state *before* calling `Print()` — a `LiveView()` height change afterwards leaves ghost
  lines at the bottom of the live region.
- `Markdown` and `Scroll` need a `*int` for scroll position; pass `nil` to `Markdown` for no
  scrolling.
- Declare conditionally-animated views as `tui.View`; `.Animate()` returns a different concrete
  type than `*TextView`.
- Chain order matters where a modifier widens the type. `Stack(...).Padding(n)` returns `View`,
  so `.Flex()`, `.Gap()` and `.Align()` must come before it.
- Keep the live region small. Large scrollable content belongs in a fullscreen `tui.Run` app.

Snapshot testing of views is covered in [testing.md](testing.md).
