package tui

import (
	"testing"

	"github.com/deepnoodle-ai/gooey/require"
)

func TestTreeNode(t *testing.T) {
	root := NewTreeNode("root")
	require.Equal(t, "root", root.Label)
	require.True(t, root.IsLeaf())

	child := NewTreeNode("child")
	root.AddChild(child)
	require.False(t, root.IsLeaf())
	require.Len(t, root.Children, 1)
}

func TestTreeNodeChaining(t *testing.T) {
	root := NewTreeNode("root").
		SetExpanded(true).
		SetData("test-data").
		AddChildren(
			NewTreeNode("child1"),
			NewTreeNode("child2"),
		)

	require.True(t, root.Expanded)
	require.Equal(t, "test-data", root.Data)
	require.Len(t, root.Children, 2)
}

func TestTreeNodeExpandCollapse(t *testing.T) {
	root := NewTreeNode("root").AddChildren(
		NewTreeNode("child").AddChildren(
			NewTreeNode("grandchild"),
		),
	)

	root.ExpandAll()
	require.True(t, root.Expanded)
	require.True(t, root.Children[0].Expanded)

	root.CollapseAll()
	require.False(t, root.Expanded)
	require.False(t, root.Children[0].Expanded)
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

	require.Len(t, nodes, 4) // root + child1 + child2 + grandchild

	// Verify depths
	require.Equal(t, 0, nodes[0].depth) // root
	require.Equal(t, 1, nodes[1].depth) // child1
	require.Equal(t, 1, nodes[2].depth) // child2
	require.Equal(t, 2, nodes[3].depth) // grandchild
}

func TestTreeViewCollapsedChildren(t *testing.T) {
	root := NewTreeNode("root").SetExpanded(true).AddChildren(
		NewTreeNode("child").SetExpanded(false).AddChildren(
			NewTreeNode("hidden-grandchild"),
		),
	)

	view := Tree(root)
	nodes := view.flatten()

	require.Len(t, nodes, 2) // only root and child visible
}

func TestTreeViewSize(t *testing.T) {
	root := NewTreeNode("root").SetExpanded(true).AddChildren(
		NewTreeNode("child1"),
		NewTreeNode("child2"),
	)

	view := Tree(root)
	w, h := view.size(100, 100)

	require.True(t, w > 0)
	require.Equal(t, 3, h) // 3 visible nodes
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

	require.Equal(t, 2, h)
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
	require.NotNil(t, found)
	require.Equal(t, "target", found.Label)

	notFound := view.FindNode("nonexistent")
	require.Nil(t, notFound)
}

func TestTreeViewOnSelect(t *testing.T) {
	var selectedLabel string

	root := NewTreeNode("root")
	Tree(root).OnSelect(func(node *TreeNode) {
		selectedLabel = node.Label
	})

	// The callback would be triggered through the interactive registry
	// during actual rendering and clicking
	require.Equal(t, "", selectedLabel)
}

func TestTreeViewGetVisibleCount(t *testing.T) {
	root := NewTreeNode("root").SetExpanded(true).AddChildren(
		NewTreeNode("child1"),
		NewTreeNode("child2"),
	)

	view := Tree(root)
	require.Equal(t, 3, view.GetVisibleCount())

	root.Expanded = false
	require.Equal(t, 1, view.GetVisibleCount())
}

func TestTreeBranchChars(t *testing.T) {
	chars := DefaultTreeBranchChars()
	require.Equal(t, "│", chars.Vertical)
	require.Equal(t, "└", chars.Corner)
	require.Equal(t, "├", chars.Tee)
	require.Equal(t, "─", chars.Horizontal)

	ascii := ASCIITreeBranchChars()
	require.Equal(t, "|", ascii.Vertical)
	require.Equal(t, "`", ascii.Corner)
}
