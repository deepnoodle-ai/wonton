package fetch

import (
	"encoding/json"
	"fmt"
)

// TypedAction represents an action to be performed on a page during fetching.
//
// Actions are used with browser automation to perform operations like taking
// screenshots or waiting for page elements before capturing content.
// Each action type implements this interface.
type TypedAction interface {
	// GetType returns the action type identifier (e.g., "screenshot", "wait").
	GetType() string
}

// BaseAction contains common fields for all action types.
//
// This type is embedded in specific action types to provide the Type field.
type BaseAction struct {
	// Type is the action type identifier.
	Type string `json:"type"`
}

// GetType returns the action type identifier.
func (a BaseAction) GetType() string {
	return a.Type
}

// ScreenshotAction captures a screenshot of the page.
//
// This action requires browser automation support and will fail with
// HTTPFetcher. The screenshot is returned as a base64-encoded image
// in the Response.Screenshot field.
type ScreenshotAction struct {
	BaseAction
	// FullPage specifies whether to capture the entire page (true) or
	// just the visible viewport (false).
	FullPage bool `json:"full_page,omitempty"`
}

// WaitAction pauses execution for a specified duration or until an element appears.
//
// This action requires browser automation support. It can wait either for
// a specific element to appear on the page or for a fixed duration.
// At least one of Selector or Milliseconds should be specified.
type WaitAction struct {
	BaseAction
	// Selector is a CSS selector to wait for. When specified, the action
	// waits until an element matching this selector appears on the page.
	Selector string `json:"selector,omitempty"`

	// Milliseconds is the duration to wait in milliseconds. When specified
	// without Selector, waits for this fixed duration.
	Milliseconds int `json:"milliseconds,omitempty"`
}

// Action is a wrapper for polymorphic action types, supporting JSON marshaling.
//
// This type handles the serialization and deserialization of different action
// types, allowing actions to be specified in JSON with their specific fields.
type Action struct {
	// Action is the underlying typed action (ScreenshotAction, WaitAction, etc.).
	Action TypedAction
}

// UnmarshalJSON implements custom unmarshaling for polymorphic actions.
//
// This method deserializes JSON into the appropriate action type based on
// the "type" field in the JSON. Returns an error for unknown action types.
func (a *Action) UnmarshalJSON(data []byte) error {
	// Unmarshal just the type field
	var typeOnly struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typeOnly); err != nil {
		return err
	}
	switch typeOnly.Type {
	case "screenshot":
		var action ScreenshotAction
		action.Type = typeOnly.Type
		if err := json.Unmarshal(data, &action); err != nil {
			return err
		}
		a.Action = &action
	case "wait":
		var action WaitAction
		action.Type = typeOnly.Type
		if err := json.Unmarshal(data, &action); err != nil {
			return err
		}
		a.Action = &action
	default:
		return fmt.Errorf("unknown action type: %s", typeOnly.Type)
	}
	return nil
}

// MarshalJSON implements custom marshaling for polymorphic actions.
//
// This method serializes the underlying action to JSON, preserving all
// action-specific fields along with the type field.
func (a *Action) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Action)
}

// ScreenshotActionOptions contains options for creating a screenshot action.
type ScreenshotActionOptions struct {
	// FullPage specifies whether to capture the full page or just the viewport.
	FullPage bool `json:"full_page,omitempty"`
}

// NewScreenshotAction creates a new screenshot action with the given options.
//
// Example:
//
//	action := fetch.NewScreenshotAction(fetch.ScreenshotActionOptions{
//		FullPage: true,
//	})
func NewScreenshotAction(options ScreenshotActionOptions) Action {
	return Action{
		Action: &ScreenshotAction{
			BaseAction: BaseAction{Type: "screenshot"},
			FullPage:   options.FullPage,
		},
	}
}

// WaitActionOptions contains options for creating a wait action.
type WaitActionOptions struct {
	// Selector is a CSS selector to wait for. The action will wait until
	// an element matching this selector appears on the page.
	Selector string `json:"selector,omitempty"`

	// Milliseconds is the duration to wait in milliseconds.
	Milliseconds int `json:"milliseconds,omitempty"`
}

// NewWaitAction creates a new wait action with the given options.
//
// Specify Selector to wait for an element, Milliseconds to wait a fixed
// duration, or both to wait for an element with a timeout.
//
// Example:
//
//	// Wait for an element to appear
//	action := fetch.NewWaitAction(fetch.WaitActionOptions{
//		Selector: ".content",
//	})
//
//	// Wait for a fixed duration
//	action := fetch.NewWaitAction(fetch.WaitActionOptions{
//		Milliseconds: 1000,
//	})
func NewWaitAction(options WaitActionOptions) Action {
	return Action{
		Action: &WaitAction{
			BaseAction:   BaseAction{Type: "wait"},
			Selector:     options.Selector,
			Milliseconds: options.Milliseconds,
		},
	}
}
