package tui

import (
	"testing"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestTreeNode(t *testing.T) {
	root := NewTreeNode("root")
	assert.Equal(t, "root", root.Label)
	assert.True(t, root.IsLeaf())

	child := NewTreeNode("child")
	root.AddChild(child)
	assert.False(t, root.IsLeaf())
	assert.Len(t, root.Children, 1)
}

func TestTreeNodeChaining(t *testing.T) {
	root := NewTreeNode("root").
		SetExpanded(true).
		SetData("test-data").
		AddChildren(
			NewTreeNode("child1"),
			NewTreeNode("child2"),
		)

	assert.True(t, root.Expanded)
	assert.Equal(t, "test-data", root.Data)
	assert.Len(t, root.Children, 2)
}

func TestTreeNodeExpandCollapse(t *testing.T) {
	root := NewTreeNode("root").AddChildren(
		NewTreeNode("child").AddChildren(
			NewTreeNode("grandchild"),
		),
	)

	root.ExpandAll()
	assert.True(t, root.Expanded)
	assert.True(t, root.Children[0].Expanded)

	root.CollapseAll()
	assert.False(t, root.Expanded)
	assert.False(t, root.Children[0].Expanded)
}

func TestTreeViewFlatten(t *testing.T) {
	root := NewTreeNode("root").SetExpanded(true).AddChildren(
		NewTreeNode("child1"),
		NewTreeNode("child2").SetExpanded(true).AddChildren(
			NewTreeNode("grandchild"),
		),
	)

	view := Tree(root)
	nodes := view.flatten()

	assert.Len(t, nodes, 4) // root + child1 + child2 + grandchild

	// Verify depths
	assert.Equal(t, 0, nodes[0].depth) // root
	assert.Equal(t, 1, nodes[1].depth) // child1
	assert.Equal(t, 1, nodes[2].depth) // child2
	assert.Equal(t, 2, nodes[3].depth) // grandchild
}

func TestTreeViewCollapsedChildren(t *testing.T) {
	root := NewTreeNode("root").SetExpanded(true).AddChildren(
		NewTreeNode("child").SetExpanded(false).AddChildren(
			NewTreeNode("hidden-grandchild"),
		),
	)

	view := Tree(root)
	nodes := view.flatten()

	assert.Len(t, nodes, 2) // only root and child visible
}

func TestTreeViewSize(t *testing.T) {
	root := NewTreeNode("root").SetExpanded(true).AddChildren(
		NewTreeNode("child1"),
		NewTreeNode("child2"),
	)

	view := Tree(root)
	w, h := view.size(100, 100)

	assert.True(t, w > 0)
	assert.Equal(t, 3, h) // 3 visible nodes
}

func TestTreeViewFixedHeight(t *testing.T) {
	root := NewTreeNode("root").SetExpanded(true).AddChildren(
		NewTreeNode("c1"),
		NewTreeNode("c2"),
		NewTreeNode("c3"),
		NewTreeNode("c4"),
	)

	view := Tree(root).Height(2)
	_, h := view.size(100, 100)

	assert.Equal(t, 2, h)
}

func TestTreeViewFindNode(t *testing.T) {
	root := NewTreeNode("root").SetExpanded(true).AddChildren(
		NewTreeNode("child1"),
		NewTreeNode("child2").AddChildren(
			NewTreeNode("target"),
		),
	)

	view := Tree(root)

	found := view.FindNode("target")
	assert.NotNil(t, found)
	assert.Equal(t, "target", found.Label)

	notFound := view.FindNode("nonexistent")
	assert.Nil(t, notFound)
}

func TestTreeViewOnSelect(t *testing.T) {
	var selectedLabel string

	root := NewTreeNode("root")
	Tree(root).OnSelect(func(node *TreeNode) {
		selectedLabel = node.Label
	})

	// The callback would be triggered through the interactive registry
	// during actual rendering and clicking
	assert.Equal(t, "", selectedLabel)
}

func TestTreeViewGetVisibleCount(t *testing.T) {
	root := NewTreeNode("root").SetExpanded(true).AddChildren(
		NewTreeNode("child1"),
		NewTreeNode("child2"),
	)

	view := Tree(root)
	assert.Equal(t, 3, view.GetVisibleCount())

	root.Expanded = false
	assert.Equal(t, 1, view.GetVisibleCount())
}

func TestTreeBranchChars(t *testing.T) {
	chars := DefaultTreeBranchChars()
	assert.Equal(t, "│", chars.Vertical)
	assert.Equal(t, "└", chars.Corner)
	assert.Equal(t, "├", chars.Tee)
	assert.Equal(t, "─", chars.Horizontal)

	ascii := ASCIITreeBranchChars()
	assert.Equal(t, "|", ascii.Vertical)
	assert.Equal(t, "`", ascii.Corner)
}
