package tui

import (
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/deepnoodle-ai/wonton/internal/terminalstate"
)

type transitionInput struct {
	pause, resume, discard func() error
}

func (o transitionInput) Read([]byte) (int, error) { return 0, io.EOF }
func (o transitionInput) Pause() error             { return o.pause() }
func (o transitionInput) Resume() error            { return o.resume() }
func (o transitionInput) DiscardResidual() error   { return o.discard() }
func (transitionInput) Stop()                      {}
func (transitionInput) Close() error               { return nil }

func TestHandoffTransitionFailuresAndStop(t *testing.T) {
	sentinel := errors.New("fixture failure")
	for _, phase := range []string{"none", "snapshot", "pause", "release", "callback", "discard", "restore", "repaint", "resume", "stop-child", "stop-repaint", "panic"} {
		t.Run(phase, func(t *testing.T) {
			var calls []string
			stopped := false
			var fatal error
			m := &managedInput{done: make(chan struct{}), wake: make(chan struct{})}
			m.owner = transitionInput{
				pause: func() error {
					calls = append(calls, "pause")
					if phase == "pause" {
						return sentinel
					}
					close(m.cycle.ack)
					return nil
				},
				discard: func() error {
					calls = append(calls, "discard")
					if phase == "discard" {
						return sentinel
					}
					return nil
				},
				resume: func() error {
					calls = append(calls, "resume")
					if phase == "resume" {
						return sentinel
					}
					return nil
				},
			}
			snapshot := func() (terminalstate.Transition, error) {
				calls = append(calls, "snapshot")
				if phase == "snapshot" {
					return terminalstate.Transition{}, sentinel
				}
				return terminalstate.Transition{
					Release: func() error {
						calls = append(calls, "release")
						if phase == "release" {
							return sentinel
						}
						return nil
					},
					Restore: func(plain bool) error {
						if plain {
							calls = append(calls, "plain")
						} else {
							calls = append(calls, "restore")
						}
						if phase == "restore" && !plain {
							return sentinel
						}
						return nil
					},
				}, nil
			}
			var operationErr, restoreErr error
			var panicValue any
			func() {
				defer func() { panicValue = recover() }()
				operationErr, restoreErr = runTerminalHandoff(func() error {
					calls = append(calls, "callback")
					if phase == "stop-child" {
						stopped = true
					}
					if phase == "panic" {
						panic("fixture panic")
					}
					if phase == "callback" {
						return sentinel
					}
					return nil
				}, m, snapshot, func() bool { return stopped }, func() error {
					calls = append(calls, "repaint")
					if phase == "stop-repaint" {
						stopped = true
					}
					if phase == "repaint" {
						return sentinel
					}
					return nil
				}, func(err error) { fatal = err; stopped = true })
			}()
			expected := map[string][]string{
				"none":         {"snapshot", "pause", "release", "callback", "discard", "restore", "repaint", "resume"},
				"snapshot":     {"snapshot"},
				"pause":        {"snapshot", "pause", "plain"},
				"release":      {"snapshot", "pause", "release", "discard", "restore", "repaint", "resume"},
				"callback":     {"snapshot", "pause", "release", "callback", "discard", "restore", "repaint", "resume"},
				"discard":      {"snapshot", "pause", "release", "callback", "discard", "plain", "plain"},
				"restore":      {"snapshot", "pause", "release", "callback", "discard", "restore", "plain"},
				"repaint":      {"snapshot", "pause", "release", "callback", "discard", "restore", "repaint", "plain", "plain"},
				"resume":       {"snapshot", "pause", "release", "callback", "discard", "restore", "repaint", "resume", "plain"},
				"stop-child":   {"snapshot", "pause", "release", "callback", "discard", "plain"},
				"stop-repaint": {"snapshot", "pause", "release", "callback", "discard", "restore", "repaint", "plain"},
				"panic":        {"snapshot", "pause", "release", "callback", "discard", "restore", "repaint", "resume"},
			}[phase]
			if !reflect.DeepEqual(calls, expected) {
				t.Fatalf("phases=%v want=%v", calls, expected)
			}
			wantOperation := phase == "snapshot" || phase == "pause" || phase == "release" || phase == "callback"
			wantFatal := phase == "pause" || phase == "discard" || phase == "restore" || phase == "repaint" || phase == "resume"
			if (operationErr != nil) != wantOperation || (restoreErr != nil) != wantFatal || (fatal != nil) != wantFatal {
				t.Fatalf("operation=%v restore=%v fatal=%v", operationErr, restoreErr, fatal)
			}
			if phase == "panic" && panicValue != "fixture panic" {
				t.Fatalf("panic=%v", panicValue)
			}
		})
	}
}
