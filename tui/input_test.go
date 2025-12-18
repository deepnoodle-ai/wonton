package tui

import (
	"strings"
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
	"github.com/deepnoodle-ai/wonton/terminal"
)

func TestKeyConstants_Exported(t *testing.T) {
	// Verify key constants are properly exported from terminal package
	assert.Equal(t, terminal.KeyUnknown, KeyUnknown)
	assert.Equal(t, terminal.KeyEnter, KeyEnter)
	assert.Equal(t, terminal.KeyTab, KeyTab)
	assert.Equal(t, terminal.KeyBackspace, KeyBackspace)
	assert.Equal(t, terminal.KeyEscape, KeyEscape)
	assert.Equal(t, terminal.KeyArrowUp, KeyArrowUp)
	assert.Equal(t, terminal.KeyArrowDown, KeyArrowDown)
	assert.Equal(t, terminal.KeyArrowLeft, KeyArrowLeft)
	assert.Equal(t, terminal.KeyArrowRight, KeyArrowRight)
	assert.Equal(t, terminal.KeyHome, KeyHome)
	assert.Equal(t, terminal.KeyEnd, KeyEnd)
	assert.Equal(t, terminal.KeyPageUp, KeyPageUp)
	assert.Equal(t, terminal.KeyPageDown, KeyPageDown)
	assert.Equal(t, terminal.KeyDelete, KeyDelete)
	assert.Equal(t, terminal.KeyInsert, KeyInsert)
}

func TestFunctionKeyConstants_Exported(t *testing.T) {
	// Verify function key constants
	assert.Equal(t, terminal.KeyF1, KeyF1)
	assert.Equal(t, terminal.KeyF2, KeyF2)
	assert.Equal(t, terminal.KeyF3, KeyF3)
	assert.Equal(t, terminal.KeyF4, KeyF4)
	assert.Equal(t, terminal.KeyF5, KeyF5)
	assert.Equal(t, terminal.KeyF6, KeyF6)
	assert.Equal(t, terminal.KeyF7, KeyF7)
	assert.Equal(t, terminal.KeyF8, KeyF8)
	assert.Equal(t, terminal.KeyF9, KeyF9)
	assert.Equal(t, terminal.KeyF10, KeyF10)
	assert.Equal(t, terminal.KeyF11, KeyF11)
	assert.Equal(t, terminal.KeyF12, KeyF12)
}

func TestCtrlKeyConstants_Exported(t *testing.T) {
	// Verify ctrl key constants
	assert.Equal(t, terminal.KeyCtrlA, KeyCtrlA)
	assert.Equal(t, terminal.KeyCtrlB, KeyCtrlB)
	assert.Equal(t, terminal.KeyCtrlC, KeyCtrlC)
	assert.Equal(t, terminal.KeyCtrlD, KeyCtrlD)
	assert.Equal(t, terminal.KeyCtrlE, KeyCtrlE)
	assert.Equal(t, terminal.KeyCtrlF, KeyCtrlF)
	assert.Equal(t, terminal.KeyCtrlG, KeyCtrlG)
	assert.Equal(t, terminal.KeyCtrlH, KeyCtrlH)
	assert.Equal(t, terminal.KeyCtrlI, KeyCtrlI)
	assert.Equal(t, terminal.KeyCtrlJ, KeyCtrlJ)
	assert.Equal(t, terminal.KeyCtrlK, KeyCtrlK)
	assert.Equal(t, terminal.KeyCtrlL, KeyCtrlL)
	assert.Equal(t, terminal.KeyCtrlM, KeyCtrlM)
	assert.Equal(t, terminal.KeyCtrlN, KeyCtrlN)
	assert.Equal(t, terminal.KeyCtrlO, KeyCtrlO)
	assert.Equal(t, terminal.KeyCtrlP, KeyCtrlP)
	assert.Equal(t, terminal.KeyCtrlQ, KeyCtrlQ)
	assert.Equal(t, terminal.KeyCtrlR, KeyCtrlR)
	assert.Equal(t, terminal.KeyCtrlS, KeyCtrlS)
	assert.Equal(t, terminal.KeyCtrlT, KeyCtrlT)
	assert.Equal(t, terminal.KeyCtrlU, KeyCtrlU)
	assert.Equal(t, terminal.KeyCtrlV, KeyCtrlV)
	assert.Equal(t, terminal.KeyCtrlW, KeyCtrlW)
	assert.Equal(t, terminal.KeyCtrlX, KeyCtrlX)
	assert.Equal(t, terminal.KeyCtrlY, KeyCtrlY)
	assert.Equal(t, terminal.KeyCtrlZ, KeyCtrlZ)
}

func TestPasteConstants_Exported(t *testing.T) {
	// Verify paste constants
	assert.Equal(t, terminal.PasteAccept, PasteAccept)
	assert.Equal(t, terminal.PasteReject, PasteReject)
	assert.Equal(t, terminal.PasteModified, PasteModified)
}

func TestPasteDisplayConstants_Exported(t *testing.T) {
	// Verify paste display mode constants
	assert.Equal(t, terminal.PasteDisplayNormal, PasteDisplayNormal)
	assert.Equal(t, terminal.PasteDisplayPlaceholder, PasteDisplayPlaceholder)
	assert.Equal(t, terminal.PasteDisplayHidden, PasteDisplayHidden)
}

func TestKeyEventType_IsAlias(t *testing.T) {
	// Verify KeyEvent is an alias for terminal.KeyEvent
	var ke KeyEvent
	var tke terminal.KeyEvent

	// Both should be the same type
	ke = tke
	tke = ke

	// Verify we can access fields
	ke.Key = KeyEnter
	ke.Rune = 'a'
	ke.Shift = true
	ke.Alt = true
	ke.Ctrl = true

	assert.Equal(t, KeyEnter, ke.Key)
	assert.Equal(t, 'a', ke.Rune)
	assert.True(t, ke.Shift)
	assert.True(t, ke.Alt)
	assert.True(t, ke.Ctrl)
}

func TestKeyType_IsAlias(t *testing.T) {
	// Verify Key type is an alias
	var k Key = KeyEnter
	assert.Equal(t, KeyEnter, k)
}

func TestNewKeyDecoder_Exported(t *testing.T) {
	// Verify NewKeyDecoder is exported
	assert.NotNil(t, NewKeyDecoder)

	// Create a decoder to verify it works (requires io.Reader)
	decoder := NewKeyDecoder(strings.NewReader("test"))
	assert.NotNil(t, decoder)
}

func TestPasteTypes_AreAliases(t *testing.T) {
	// Verify paste-related types are properly aliased
	var _ PasteHandlerDecision
	var _ PasteInfo
	var _ PasteHandler
	var _ PasteDisplayMode
}
