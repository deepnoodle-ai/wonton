# InlineApp Test Example

This example comprehensively tests the `InlineApp` implementation to verify:

- ✅ Live view that dynamically changes size (expands and shrinks)
- ✅ Scrollback history with various content types
- ✅ No extra newlines between live view and scrollback
- ✅ Different view compositions (Stack, Group, styled text)
- ✅ Async command execution with custom events
- ✅ Proper cursor positioning during updates

## Running the Example

```bash
go run ./examples/inline_test
```

## Test Phases

The example progresses through 5 phases that test different aspects:

### Phase 1: Minimal View
- **Size**: Small (5 lines)
- **Test**: Basic rendering
- **Action**: Press `s` to start

### Phase 2: Expanded View
- **Size**: Medium (8 lines)
- **Test**: Auto-incrementing items with scrollback printing
- **Behavior**: Automatically adds items and prints to scrollback
- **Action**: Wait for completion, then press `n`

### Phase 3: Large View with Sections
- **Size**: Large (15-18 lines, dynamic)
- **Test**: Complex layouts with conditional sections
- **Features**:
  - Progress section with checkmarks
  - Toggleable status section (press `t`)
  - Batch printing of messages to scrollback
- **Action**: Press `n` after batch print completes

### Phase 4: Minimal View (Shrink Test)
- **Size**: Small (5 lines)
- **Test**: View shrinking - verifies no residual lines
- **Action**: Press `n`

### Phase 5: Styled Content
- **Size**: Medium (8 lines)
- **Test**: Final test with styled text and colors
- **Action**: Press `r` to restart or `q` to quit

## Keyboard Controls

| Key      | Action                           |
| -------- | -------------------------------- |
| `s`      | Start test (Phase 1)             |
| `n`      | Next phase                       |
| `t`      | Toggle status section (Phase 3)  |
| `r`      | Reset/restart test               |
| `h`      | Print help message to scrollback |
| `c`      | Clear scrollback buffer          |
| `q`      | Quit application                 |
| `Ctrl+C` | Quit application                 |

## What to Verify

When running this test, pay attention to:

1. **No extra blank lines** between scrollback content and the live view
2. **Smooth transitions** when the live view changes size
3. **Clean rendering** - no flickering or residual characters
4. **Proper scrollback** - messages stay in history when live view updates
5. **Dividers align** correctly at the configured width (70 columns)

## Expected Output

You should see:
- Initial program description
- Live view at the bottom (bordered with dividers)
- Scrollback messages appear above the live view
- Live view updates in place as you progress through phases
- Final message after quitting confirming clean exit
