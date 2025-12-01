package tui

import (
	"image"

	"github.com/mattn/go-runewidth"
)

// TreeNode represents a node in a tree view.
type TreeNode struct {
	// Label is the display text for this node.
	Label string

	// Children contains the child nodes.
	Children []*TreeNode

	// Expanded indicates whether this node's children are visible.
	Expanded bool

	// Data holds arbitrary user data associated with this node.
	Data any
}

// NewTreeNode creates a new tree node with the given label.
func NewTreeNode(label string) *TreeNode {
	return &TreeNode{Label: label}
}

// AddChild adds a child node and returns it for chaining.
func (n *TreeNode) AddChild(child *TreeNode) *TreeNode {
	n.Children = append(n.Children, child)
	return child
}

// AddChildren adds multiple child nodes.
func (n *TreeNode) AddChildren(children ...*TreeNode) *TreeNode {
	n.Children = append(n.Children, children...)
	return n
}

// SetData sets the user data and returns the node for chaining.
func (n *TreeNode) SetData(data any) *TreeNode {
	n.Data = data
	return n
}

// SetExpanded sets the expanded state.
func (n *TreeNode) SetExpanded(expanded bool) *TreeNode {
	n.Expanded = expanded
	return n
}

// IsLeaf returns true if this node has no children.
func (n *TreeNode) IsLeaf() bool {
	return len(n.Children) == 0
}

// ExpandAll expands this node and all descendants.
func (n *TreeNode) ExpandAll() {
	n.Expanded = true
	for _, child := range n.Children {
		child.ExpandAll()
	}
}

// CollapseAll collapses this node and all descendants.
func (n *TreeNode) CollapseAll() {
	n.Expanded = false
	for _, child := range n.Children {
		child.CollapseAll()
	}
}

// treeView displays a tree of nodes with expand/collapse support.
type treeView struct {
	root         *TreeNode
	selected     *TreeNode
	scrollY      *int
	height       int
	onSelect     func(*TreeNode)
	style        Style
	selectedSty  Style
	expandedChar string
	collapsedChr string
	leafChar     string
	branchChars  TreeBranchChars
}

// TreeBranchChars defines the characters used for drawing tree branches.
type TreeBranchChars struct {
	Vertical   string // │
	Corner     string // └
	Tee        string // ├
	Horizontal string // ─
}

// DefaultTreeBranchChars returns the default unicode tree branch characters.
func DefaultTreeBranchChars() TreeBranchChars {
	return TreeBranchChars{
		Vertical:   "│",
		Corner:     "└",
		Tee:        "├",
		Horizontal: "─",
	}
}

// ASCIITreeBranchChars returns ASCII tree branch characters.
func ASCIITreeBranchChars() TreeBranchChars {
	return TreeBranchChars{
		Vertical:   "|",
		Corner:     "`",
		Tee:        "|",
		Horizontal: "-",
	}
}

// Tree creates a tree view with the given root node.
//
// Example:
//
//	root := tui.NewTreeNode("Root").SetExpanded(true).AddChildren(
//	    tui.NewTreeNode("Child 1"),
//	    tui.NewTreeNode("Child 2").AddChildren(
//	        tui.NewTreeNode("Grandchild"),
//	    ),
//	)
//	Tree(root).OnSelect(func(node *tui.TreeNode) {
//	    fmt.Println("Selected:", node.Label)
//	})
func Tree(root *TreeNode) *treeView {
	return &treeView{
		root:         root,
		expandedChar: "▼",
		collapsedChr: "▶",
		leafChar:     " ",
		branchChars:  DefaultTreeBranchChars(),
		style:        NewStyle(),
		selectedSty:  NewStyle().WithReverse(),
	}
}

// Selected sets the currently selected node.
func (t *treeView) Selected(node *TreeNode) *treeView {
	t.selected = node
	return t
}

// OnSelect sets a callback when a node is selected/clicked.
func (t *treeView) OnSelect(fn func(*TreeNode)) *treeView {
	t.onSelect = fn
	return t
}

// ScrollY sets the scroll position pointer.
func (t *treeView) ScrollY(scrollY *int) *treeView {
	t.scrollY = scrollY
	return t
}

// Height sets a fixed height for the tree.
func (t *treeView) Height(h int) *treeView {
	t.height = h
	return t
}

// Style sets the default style for nodes.
func (t *treeView) Style(s Style) *treeView {
	t.style = s
	return t
}

// SelectedStyle sets the style for the selected node.
func (t *treeView) SelectedStyle(s Style) *treeView {
	t.selectedSty = s
	return t
}

// ExpandChar sets the character shown for expanded nodes.
func (t *treeView) ExpandChar(c string) *treeView {
	t.expandedChar = c
	return t
}

// CollapseChar sets the character shown for collapsed nodes.
func (t *treeView) CollapseChar(c string) *treeView {
	t.collapsedChr = c
	return t
}

// LeafChar sets the character shown for leaf nodes.
func (t *treeView) LeafChar(c string) *treeView {
	t.leafChar = c
	return t
}

// BranchChars sets the characters used for drawing tree branches.
func (t *treeView) BranchChars(chars TreeBranchChars) *treeView {
	t.branchChars = chars
	return t
}

// flattenedNode represents a node in the flattened view with its depth and visibility info.
type flattenedNode struct {
	node     *TreeNode
	depth    int
	isLast   bool   // is this the last sibling at its level
	ancestors []bool // for each ancestor level, whether we need to draw a vertical line
}

// flatten converts the tree to a flat list of visible nodes.
func (t *treeView) flatten() []flattenedNode {
	if t.root == nil {
		return nil
	}

	var result []flattenedNode
	t.flattenNode(t.root, 0, true, nil, &result)
	return result
}

func (t *treeView) flattenNode(node *TreeNode, depth int, isLast bool, ancestors []bool, result *[]flattenedNode) {
	*result = append(*result, flattenedNode{
		node:     node,
		depth:    depth,
		isLast:   isLast,
		ancestors: append([]bool{}, ancestors...), // copy ancestors
	})

	if !node.Expanded || len(node.Children) == 0 {
		return
	}

	// Update ancestors for children
	newAncestors := append(ancestors, !isLast)

	for i, child := range node.Children {
		childIsLast := i == len(node.Children)-1
		t.flattenNode(child, depth+1, childIsLast, newAncestors, result)
	}
}

func (t *treeView) size(maxWidth, maxHeight int) (int, int) {
	nodes := t.flatten()

	// Calculate width needed
	maxW := 0
	for _, fn := range nodes {
		// indent + expand char + space + label
		indent := fn.depth * 2
		expandWidth := runewidth.StringWidth(t.expandedChar)
		labelWidth := runewidth.StringWidth(fn.node.Label)
		w := indent + expandWidth + 1 + labelWidth
		if w > maxW {
			maxW = w
		}
	}

	w := maxW
	if maxWidth > 0 && w > maxWidth {
		w = maxWidth
	}

	h := t.height
	if h == 0 {
		h = len(nodes)
	}
	if maxHeight > 0 && h > maxHeight {
		h = maxHeight
	}

	return w, h
}

func (t *treeView) render(frame RenderFrame, bounds image.Rectangle) {
	if bounds.Empty() || t.root == nil {
		return
	}

	nodes := t.flatten()
	width := bounds.Dx()
	height := bounds.Dy()
	subFrame := frame.SubFrame(bounds)

	// Get scroll position
	scrollY := 0
	if t.scrollY != nil {
		scrollY = *t.scrollY
	}

	// Clamp scroll
	maxScroll := len(nodes) - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollY > maxScroll {
		scrollY = maxScroll
	}
	if scrollY < 0 {
		scrollY = 0
	}

	// Update scroll pointer
	if t.scrollY != nil && *t.scrollY != scrollY {
		*t.scrollY = scrollY
	}

	branchStyle := NewStyle().WithForeground(ColorBrightBlack)

	// Render visible nodes
	for y := 0; y < height && scrollY+y < len(nodes); y++ {
		fn := nodes[scrollY+y]
		node := fn.node

		x := 0

		// Draw tree branches
		for level := 0; level < fn.depth; level++ {
			if level < len(fn.ancestors) {
				if fn.ancestors[level] {
					// Draw vertical line
					subFrame.PrintStyled(x, y, t.branchChars.Vertical+" ", branchStyle)
				} else {
					// Draw space
					subFrame.PrintStyled(x, y, "  ", branchStyle)
				}
			}
			x += 2
		}

		// Draw connector for non-root nodes
		if fn.depth > 0 {
			connector := t.branchChars.Tee
			if fn.isLast {
				connector = t.branchChars.Corner
			}
			subFrame.PrintStyled(x-2, y, connector+t.branchChars.Horizontal, branchStyle)
		}

		// Draw expand/collapse indicator
		indicator := t.leafChar
		if !node.IsLeaf() {
			if node.Expanded {
				indicator = t.expandedChar
			} else {
				indicator = t.collapsedChr
			}
		}

		// Determine style based on selection
		style := t.style
		if node == t.selected {
			style = t.selectedSty
		}

		subFrame.PrintStyled(x, y, indicator+" ", style)
		x += runewidth.StringWidth(indicator) + 1

		// Draw label
		label := node.Label
		labelWidth := runewidth.StringWidth(label)
		if x+labelWidth > width {
			label = truncateToWidth(label, width-x)
		}
		subFrame.PrintStyled(x, y, label, style)

		// Register clickable region
		clickBounds := image.Rect(
			bounds.Min.X,
			bounds.Min.Y+y,
			bounds.Max.X,
			bounds.Min.Y+y+1,
		)
		nodeCopy := node // capture for closure
		interactiveRegistry.RegisterButton(clickBounds, func() {
			// Toggle expand/collapse for non-leaf nodes
			if !nodeCopy.IsLeaf() {
				nodeCopy.Expanded = !nodeCopy.Expanded
			}
			t.selected = nodeCopy
			if t.onSelect != nil {
				t.onSelect(nodeCopy)
			}
		})
	}
}

// GetVisibleCount returns the number of currently visible nodes.
func (t *treeView) GetVisibleCount() int {
	return len(t.flatten())
}

// FindNode finds a node by its label (depth-first search).
func (t *treeView) FindNode(label string) *TreeNode {
	if t.root == nil {
		return nil
	}
	return findNodeByLabel(t.root, label)
}

func findNodeByLabel(node *TreeNode, label string) *TreeNode {
	if node.Label == label {
		return node
	}
	for _, child := range node.Children {
		if found := findNodeByLabel(child, label); found != nil {
			return found
		}
	}
	return nil
}
