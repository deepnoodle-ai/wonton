package termsession

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// LoadCastFile loads a .cast file from disk and returns its contents.
//
// The file is automatically decompressed if it's gzip-compressed.
// This is a convenience wrapper around LoadCast that opens the file for you.
//
// Returns:
//   - RecordingHeader: The file metadata (dimensions, title, etc.)
//   - []RecordingEvent: All events in chronological order
//   - error: Any error encountered while loading
//
// Example:
//
//	header, events, err := termsession.LoadCastFile("recording.cast")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Recording is %.2f seconds long\n", events[len(events)-1].Time)
func LoadCastFile(filename string) (*RecordingHeader, []RecordingEvent, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	return LoadCast(file)
}

// LoadCast loads a .cast recording from an io.Reader.
//
// This function automatically detects and handles gzip compression by checking
// the magic bytes. It parses the asciinema v2 format: first line is a JSON header,
// subsequent lines are JSON arrays representing events.
//
// Malformed events are silently skipped to handle recordings with corruption.
//
// Use this when you have a recording in memory or from a network stream.
// For loading from a file path, use LoadCastFile instead.
func LoadCast(r io.Reader) (*RecordingHeader, []RecordingEvent, error) {
	// We need to peek at the first bytes to detect gzip
	// Use a seeker if available, otherwise buffer
	var reader io.Reader

	if seeker, ok := r.(io.ReadSeeker); ok {
		magic := make([]byte, 2)
		if _, err := io.ReadFull(r, magic); err != nil {
			return nil, nil, fmt.Errorf("failed to read file header: %w", err)
		}
		if _, err := seeker.Seek(0, 0); err != nil {
			return nil, nil, fmt.Errorf("failed to seek: %w", err)
		}

		if magic[0] == 0x1f && magic[1] == 0x8b {
			gzipReader, err := gzip.NewReader(r)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create gzip reader: %w", err)
			}
			defer gzipReader.Close()
			reader = gzipReader
		} else {
			reader = r
		}
	} else {
		// Buffer the reader so we can peek
		br := bufio.NewReader(r)
		magic, err := br.Peek(2)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read file header: %w", err)
		}

		if magic[0] == 0x1f && magic[1] == 0x8b {
			gzipReader, err := gzip.NewReader(br)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create gzip reader: %w", err)
			}
			defer gzipReader.Close()
			reader = gzipReader
		} else {
			reader = br
		}
	}

	scanner := bufio.NewScanner(reader)
	// Increase buffer size for long lines
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	// Read header (first line)
	if !scanner.Scan() {
		return nil, nil, fmt.Errorf("empty file")
	}

	var header RecordingHeader
	if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
		return nil, nil, fmt.Errorf("failed to parse header: %w", err)
	}

	// Read events (remaining lines)
	var events []RecordingEvent
	for scanner.Scan() {
		var raw []interface{}
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			continue // Skip malformed lines
		}

		if len(raw) < 3 {
			continue // Skip incomplete events
		}

		// Parse [time, type, data] array
		timeVal, ok := raw[0].(float64)
		if !ok {
			continue
		}
		typeVal, ok := raw[1].(string)
		if !ok {
			continue
		}
		dataVal, ok := raw[2].(string)
		if !ok {
			continue
		}

		events = append(events, RecordingEvent{
			Time: timeVal,
			Type: typeVal,
			Data: dataVal,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading recording: %w", err)
	}

	return &header, events, nil
}

// Duration returns the total duration of a recording in seconds.
//
// The duration is determined by the timestamp of the last event.
// Returns 0 if there are no events.
func Duration(events []RecordingEvent) float64 {
	if len(events) == 0 {
		return 0
	}
	return events[len(events)-1].Time
}

// OutputEvents filters and returns only output events (type "o").
//
// Input events (type "i") are filtered out. This is useful because most
// playback scenarios only need to render output, and input events are
// primarily kept for analysis purposes.
//
// The returned slice contains references to the original events (not copies).
func OutputEvents(events []RecordingEvent) []RecordingEvent {
	var output []RecordingEvent
	for _, e := range events {
		if e.Type == "o" {
			output = append(output, e)
		}
	}
	return output
}
