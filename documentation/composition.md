# What Components Exist

You have a solid foundation of components:

Interactive: Button, CheckboxGroup, RadioGroup, TabCompleter, Table, Modal
Progress: Spinner, ProgressBar, MultiProgress
Layout: Grid, Frame, Box, Screen, Layout
Animation: AnimatedLayout, AnimatedText, AnimatedStatusBar
Input: Multiple input field implementations

Component Composition: Limited, Not Arbitrary

Short answer: No, components cannot nest arbitrarily like in web design. Here's why:

What Works Today

1. Grid Layout - The only proper container that supports composition:


    - Accepts widgets via AddWidget() and AddWidgetSpan()
    - Creates SubFrame regions with automatic bounds clipping
    - Grids can technically nest inside other grids
    - See examples/grid_layout_demo/main.go for working examples

2. Manual Wrapper Pattern - You can compose by embedding:
   type MyComposite struct {
   button *Button
   checkbox *CheckboxGroup
   }

func (c \*MyComposite) Draw(frame RenderFrame) {
buttonFrame := frame.SubFrame(image.Rect(0, 0, 20, 3))
c.button.Draw(buttonFrame)
// ... manually position children
}

What's Missing

1. No Component Tree - Unlike React's component tree or HTML DOM:


    - Screen contains a flat list of widgets
    - No parent-child hierarchy or traversal
    - No event bubbling up the tree

2. Absolute Positioning Everywhere - Components use fixed X, Y coordinates:


    - Button, Table, CheckboxGroup all have absolute positions
    - Makes composition difficult - can't easily move a component into a container
    - No relative positioning model

3. No Layout Managers - Only Grid provides automatic positioning:


    - No FlexBox equivalent
    - No VBox/HBox for simple stacking
    - No constraint-based sizing

4. No Standard Container - Missing a general-purpose container widget:


    - No Box/Panel that accepts arbitrary children
    - No standardized children array pattern
    - Grid is purpose-built for grid layouts, not general composition

5. Widget Interface Too Simple:
   type Widget interface {
   Draw(frame RenderFrame)
   HandleKey(event KeyEvent) bool
   }


    - No SetBounds() for parent to position child
    - No initialization/cleanup lifecycle
    - No way to notify of position/size changes

Architectural Limitations

The core issue is absolute coordinates vs relative positioning:

// Current pattern - absolute positioning
button := &Button{
X: 10, Y: 5, // Fixed screen coordinates
Label: "Click Me",
}

// What's needed for composition - bounds-based
button := NewButton(ButtonProps{
Label: "Click Me",
})
container.AddChild(button) // Container assigns bounds

Components like Button, Table, and CheckboxGroup expose public X, Y fields and expect absolute screen coordinates. This works fine for flat layouts but breaks down when you want a component
to position its children.

Comparison to Web Design

| Feature                | React/HTML            | Gooey                              |
| ---------------------- | --------------------- | ---------------------------------- |
| Arbitrary nesting      | ✅ Full tree          | ❌ Flat list (Grid only exception) |
| Auto layout            | ✅ Flexbox, Grid      | ⚠️ Grid only                       |
| Relative positioning   | ✅ Standard           | ❌ Absolute coords                 |
| Props/children pattern | ✅ Standard           | ❌ Manual wiring                   |
| Style inheritance      | ✅ CSS cascade        | ❌ Each component owns style       |
| Event propagation      | ✅ Bubbling/capturing | ❌ Flat iteration                  |

What You Can Do Today

For practical composition:

1. Use Grid for structured layouts - it's the most capable container
2. Create wrapper widgets that implement the Widget interface and manually delegate to children
3. Use SubFrame to create bounded regions for child components
4. Manual coordinate calculation for positioning children

Example from examples/checkbox_demo/main.go:74:
type CheckboxDemo struct {
checkbox \*CheckboxGroup
}

func (d \*CheckboxDemo) Draw(frame RenderFrame) {
d.checkbox.Draw(frame) // Delegate to child
}

To Enable Web-Like Composition

The library would need:

1. Container widget with children array and layout manager
2. Bounds-based positioning - replace X, Y fields with SetBounds(rect)
3. Layout managers - VBox, HBox, FlexLayout interfaces
4. Component tree - hierarchical structure for rendering/events
5. Enhanced Widget interface with lifecycle methods

The good news: The SubFrame mechanism proves this is architecturally feasible.
Grid demonstrates that bounded, compositional rendering works in Gooey - it
just needs to be generalized beyond a single layout type.
