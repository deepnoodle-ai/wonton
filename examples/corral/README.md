# corral

A terminal workspace for running shells and coding agents side by side, in
the style of [herdr](https://herdr.dev). The main area is split into columns
of live terminals, and each column into rows, as deep as you like. A sidebar
lists your spaces and every pane in them.

Every pane is a tmux session on a private tmux server (`tmux -L corral`).
tmux does the terminal emulation, so anything that runs in a terminal runs in
a pane, and the sessions outlive corral: quit, run it again, and every pane is
back where you left it, still running.

```bash
go run ./examples/corral            # needs tmux on PATH
go run ./examples/corral -L work    # a separate set of sessions
```

## Using it

**Mouse**

| Do this                                   | To                                         |
| ----------------------------------------- | ------------------------------------------ |
| Click a pane, or a pane in the sidebar    | Focus it (switching spaces if needed)      |
| Click a space in the sidebar              | Show it                                    |
| Drag the edge between two panes           | Resize them                                |
| Drag the sidebar's right edge             | Resize the sidebar                         |
| Right-click a pane                        | Split, zoom, rename, copy, paste, close    |
| Right-click an edge between panes         | Even out, swap sides, turn columns to rows |
| Right-click a space or the top bar        | Space and workspace actions, quit          |
| Scroll a pane                             | Page through its history                   |
| Click the top bar's buttons               | Split right, split down, add a column      |
| Drag across text in a pane                | Select it and copy it to the clipboard     |

Programs that read the mouse themselves (vim with `mouse=a`, htop) get clicks,
drags, and the wheel directly. To select text in one of those, hold Shift (or
Option on macOS): most terminals then select on their own.

**Keys** go to the focused pane. Ctrl+B is the prefix for corral's own:

| Ctrl+B, then | Does                                   |
| ------------ | -------------------------------------- |
| `\|` `%`     | Split right                            |
| `-` `"`      | Split down                             |
| `c`          | New column                             |
| arrows, hjkl | Focus the neighboring pane             |
| `o`          | Focus the next pane                    |
| `z`          | Zoom the pane to fill the space        |
| `,`          | Rename the pane                        |
| `m`          | Open the pane's menu                   |
| `=`          | Equalize pane sizes                    |
| `s`          | New space in the pane's directory      |
| `n` `p`      | Next and previous space                |
| `x`          | Close the pane                         |
| `d` `q`      | Quit, leaving every session running    |
| `?`          | List these                             |
| Ctrl+B       | Send Ctrl+B to the pane                |

"Quit and end all sessions" in the top bar's menu stops the tmux server.

## How it is built

| Piece                        | Provides                                                  |
| ---------------------------- | --------------------------------------------------------- |
| `tmux.Terminal`              | One live pane: screen snapshots, keys, paste, mouse       |
| `tui.SplitState`, `Split`    | The nested column and row layout and its draggable edges  |
| `tui.MenuState`, `Menu`      | The right-click menus and the Ctrl+B ? key list           |
| `tui.HitMap`                 | What the pointer is over: a pane, a sidebar row, a button |
| `tui.Cells`, `GridSelection` | Drawing each pane's screen, its cursor and selection      |

The layout — spaces, the split tree, pane names — is saved as JSON in a tmux
user option (`@corral-layout`) on the server, so it lives and dies with the
sessions it describes. On start, corral drops panes whose sessions have ended
and adds any sessions the layout does not know about.

| File       | Contents                                             |
| ---------- | ---------------------------------------------------- |
| `main.go`  | Flags and startup                                    |
| `model.go` | Spaces and panes, and saving and restoring the layout |
| `app.go`   | Events and the actions behind every key and menu     |
| `input.go` | Key and mouse routing                                |
| `menus.go` | The context menus                                    |
| `view.go`  | The sidebar, bars, panes, and rename prompt          |
