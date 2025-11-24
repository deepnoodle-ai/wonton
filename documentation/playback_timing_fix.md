# Playback Timing Fix - Eliminating "Spurts"

## Problem Description

Playback of recorded sessions exhibited choppy "spurt" behavior where multiple events would play simultaneously instead of smoothly, even though the recording captured timing correctly.

## Root Cause Analysis

### Recording Behavior
The recording system captures events at the time of each `Print()` call (not at `Flush()` time). This is intentional and correct - it preserves the logical timing of when the application generates output.

However, this creates recordings where **multiple events occur within microseconds of each other**:

```
Event timestamps from demo.cast:
[2.561579, "o", "Enter text (line 1/3): "]
[2.561926917, "o", "> "]                      ← Only 0.35ms later!

[22.954217292, "o", "\u001b[16;1H"]
[22.954293417, "o", "Recording complete!"]   ← Only 0.076ms later!
[22.954358125, "o", "\u001b[18;1H"]          ← Only 0.065ms later!
[22.954363083, "o", "Play it back with:"]    ← Only 0.005ms later!
[22.954400208, "o", "  go run..."]           ← Only 0.037ms later!
```

### Playback Behavior (Original)
The original playback loop would:
1. Calculate target time for next event
2. Sleep until that time
3. Play event
4. Repeat

**Problems:**
- `time.Sleep()` for sub-millisecond durations (0.3ms, 0.076ms, etc.) is highly inaccurate
- Loop overhead (lock acquisition, time calculations) adds jitter
- Terminal I/O has its own latency
- Result: Events that should appear microseconds apart instead appear in irregular "spurts"

## Solution

Two complementary fixes were implemented:

### 1. Minimum Sleep Threshold (playback.go:174-183)

Skip sleeping for durations shorter than 10ms:

```go
if elapsed < targetTime {
    sleepDuration := time.Duration((targetTime - elapsed) * float64(time.Second))
    // Only sleep if duration is meaningful (>= 10ms)
    const minSleepDuration = 10 * time.Millisecond
    if sleepDuration >= minSleepDuration {
        select {
        case <-p.stopChan:
            return nil
        case <-time.After(sleepDuration):
        }
    }
}
```

**Why it helps:** Eliminates inaccurate micro-sleeps that cause jitter.

### 2. Event Batching (playback.go:187-217)

Batch output events that occur within 10ms of each other and write them all at once:

```go
const batchThreshold = 0.01 // 10ms in seconds
var batchedOutput []string
batchIndex := currentIndex

for batchIndex < len(p.events) {
    batchEvent := p.events[batchIndex]
    if batchEvent.Type == "o" {
        timeDiff := (batchEvent.Time - event.Time) / speed
        if batchIndex == currentIndex || timeDiff < batchThreshold {
            batchedOutput = append(batchedOutput, batchEvent.Data)
            batchIndex++
        } else {
            break
        }
    } else {
        // Skip input events (not replayed)
        if batchIndex > currentIndex {
            break
        }
        batchIndex++
    }
}

// Write all batched output at once
if len(batchedOutput) > 0 {
    terminal.mu.Lock()
    if terminal.out != nil {
        for _, data := range batchedOutput {
            terminal.out.Write([]byte(data))
        }
        if f, ok := terminal.out.(*os.File); ok {
            f.Sync()
        }
    }
    terminal.mu.Unlock()
}
```

**Why it helps:**
- Events that are naturally grouped (rapid `Print()` calls) are played as a unit
- Reduces number of terminal writes, improving efficiency
- Eliminates timing jitter between closely-spaced events
- Maintains smooth playback flow

## Example Impact

**Before:** Recording with 5 rapid `Println()` calls at timestamps:
```
[22.954217, "o", "line 1"]
[22.954293, "o", "line 2"]  ← 0.076ms later
[22.954358, "o", "line 3"]  ← 0.065ms later
[22.954363, "o", "line 4"]  ← 0.005ms later
[22.954400, "o", "line 5"]  ← 0.037ms later
```

Would play back choppily, with irregular delays due to sleep inaccuracy and loop overhead.

**After:** All 5 lines are batched and written at once at timestamp 22.954, then playback continues smoothly to next event cluster.

## Trade-offs

- **Timing accuracy:** Events within 10ms are coalesced, losing sub-10ms timing precision
- **Acceptable because:** Original recordings already have this limitation (microsecond events are impractical to replay accurately)
- **Benefit:** Smooth, natural playback that matches user perception of the original session

## Testing

1. Build: `go build ./...`
2. Record a session: `go run examples/recording_demo/main.go`
3. Play it back: `go run examples/playback_demo/main.go demo.cast`
4. Verify smooth playback without spurts

## Future Improvements

Consider making the batch threshold configurable:
- Shorter threshold (5ms) for more precise playback
- Longer threshold (20ms) for smoother playback on slower terminals
