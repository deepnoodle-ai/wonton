package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/deepnoodle-ai/wonton/terminal"
	"github.com/deepnoodle-ai/wonton/tui"
)

// Message represents a message in a conversation.
type Message struct {
	Role    string // "system", "user", "assistant"
	Content string
}

// Conversation manages multi-turn conversation state.
type Conversation struct {
	ctx      *Context
	messages []Message
	prompt   string
}

// ConversationFunc is the function signature for conversation handlers.
type ConversationFunc func(*Conversation) error

// NewConversation creates a new conversation context.
func NewConversation(ctx *Context) *Conversation {
	return &Conversation{
		ctx:      ctx,
		messages: make([]Message, 0),
		prompt:   "> ",
	}
}

// System adds a system message to the conversation.
func (c *Conversation) System(content string) *Conversation {
	c.messages = append(c.messages, Message{
		Role:    "system",
		Content: content,
	})
	return c
}

// AddMessage adds a message to the conversation history.
func (c *Conversation) AddMessage(role, content string) *Conversation {
	c.messages = append(c.messages, Message{
		Role:    role,
		Content: content,
	})
	return c
}

// History returns all messages in the conversation.
func (c *Conversation) History() []Message {
	return c.messages
}

// SetPrompt sets the input prompt string.
func (c *Conversation) SetPrompt(prompt string) *Conversation {
	c.prompt = prompt
	return c
}

// Input prompts for user input and returns it.
// In interactive mode, displays a rich prompt.
// In pipe mode, reads from stdin.
func (c *Conversation) Input(prompt string) (string, error) {
	if prompt != "" {
		c.prompt = prompt
	}

	if !c.ctx.Interactive() {
		// Non-interactive: read from stdin
		reader := bufio.NewReader(c.ctx.Stdin())
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		input := strings.TrimSuffix(line, "\n")
		c.messages = append(c.messages, Message{Role: "user", Content: input})
		return input, nil
	}

	// Interactive: use text input
	var value string
	done := false

	app := &conversationInputApp{
		conv:  c,
		value: &value,
		done:  &done,
	}

	err := tui.Run(app,
		tui.WithMouseTracking(true),
		tui.WithAlternateScreen(false),
	)
	if err != nil {
		return "", err
	}

	if !done {
		return "", io.EOF
	}

	c.messages = append(c.messages, Message{Role: "user", Content: value})
	return value, nil
}

// Reply outputs an assistant response, optionally streaming.
func (c *Conversation) Reply(prefix string, content string) {
	c.messages = append(c.messages, Message{Role: "assistant", Content: content})

	if c.ctx.Interactive() {
		// Display with formatting
		fmt.Fprintf(c.ctx.Stdout(), "%s%s\n", prefix, content)
	} else {
		// Raw output
		fmt.Fprintln(c.ctx.Stdout(), content)
	}
}

// ReplyStream outputs streaming response chunks.
func (c *Conversation) ReplyStream(prefix string, chunks <-chan string) error {
	if prefix != "" {
		fmt.Fprint(c.ctx.Stdout(), prefix)
	}

	var content strings.Builder
	for chunk := range chunks {
		content.WriteString(chunk)
		fmt.Fprint(c.ctx.Stdout(), chunk)
	}
	fmt.Fprintln(c.ctx.Stdout())

	c.messages = append(c.messages, Message{Role: "assistant", Content: content.String()})
	return nil
}

// Context returns the CLI context.
func (c *Conversation) Context() *Context {
	return c.ctx
}

// Interactive conversation input app
type conversationInputApp struct {
	conv  *Conversation
	value *string
	done  *bool
}

func (a *conversationInputApp) View() tui.View {
	return tui.VStack(
		tui.HStack(
			tui.Text("%s", a.conv.prompt),
			tui.Input(a.value).Width(60),
		),
	)
}

func (a *conversationInputApp) HandleEvent(event tui.Event) []tui.Cmd {
	switch e := event.(type) {
	case tui.KeyEvent:
		switch e.Key {
		case tui.KeyEnter:
			*a.done = true
			return []tui.Cmd{tui.Quit()}
		case tui.KeyCtrlC, tui.KeyCtrlD:
			return []tui.Cmd{tui.Quit()}
		}
	}
	return nil
}

// RunConversation runs an interactive conversation loop.
// This is a convenience function that handles the common pattern of:
// - Taking user input
// - Processing with a handler
// - Displaying response
func RunConversation(ctx *Context, handler func(conv *Conversation, input string) (string, error)) error {
	conv := NewConversation(ctx)

	for {
		input, err := conv.Input("")
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if strings.TrimSpace(input) == "" {
			continue
		}

		response, err := handler(conv, input)
		if err != nil {
			return err
		}

		conv.Reply("", response)
	}

	return nil
}

// ConversationView creates a rich TUI for conversations.
type ConversationView struct {
	messages   []Message
	input      string
	onSubmit   func(string)
	showSystem bool
}

// NewConversationView creates a new conversation view.
func NewConversationView(messages []Message, input *string, onSubmit func(string)) *ConversationView {
	return &ConversationView{
		messages: messages,
		onSubmit: onSubmit,
	}
}

// ShowSystemMessages enables display of system messages.
func (v *ConversationView) ShowSystemMessages(show bool) *ConversationView {
	v.showSystem = show
	return v
}

// View returns the Wonton view for the conversation.
func (v *ConversationView) View(input *string) tui.View {
	views := make([]tui.View, 0)

	// Render messages
	for _, msg := range v.messages {
		if msg.Role == "system" && !v.showSystem {
			continue
		}

		var style terminal.Style
		prefix := ""
		switch msg.Role {
		case "system":
			style = terminal.NewStyle().WithForeground(terminal.ColorYellow).WithItalic()
			prefix = "[System] "
		case "user":
			style = terminal.NewStyle().WithForeground(terminal.ColorGreen)
			prefix = "You: "
		case "assistant":
			style = terminal.NewStyle().WithForeground(terminal.ColorCyan)
			prefix = "Assistant: "
		}
		views = append(views, tui.Text("%s%s", prefix, msg.Content).Style(style))
	}

	views = append(views, tui.Spacer())
	views = append(views, tui.Divider())
	views = append(views, tui.HStack(
		tui.Text("> "),
		tui.Input(input).Placeholder("Type a message...").Width(60).OnSubmit(v.onSubmit),
	))

	return tui.VStack(views...).Padding(1)
}
