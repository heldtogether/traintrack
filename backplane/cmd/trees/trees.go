package trees

import (
	"fmt"
	"sort"
	"strings"
)

type Treeable interface {
	GetID() string
	GetName() string
	GetDescription() string
	GetVersion() string
	GetParent() *string
}

type treeNode[T Treeable] struct {
	Commit   T
	Children []*treeNode[T]
}

func stringPtr(s string) *string {
	return &s
}

func BuildTree[T Treeable](commits []T) []*treeNode[T] {
	nodeMap := map[string]*treeNode[T]{}
	var roots []*treeNode[T]

	for _, c := range commits {
		node := &treeNode[T]{Commit: c}
		nodeMap[c.GetID()] = node
	}

	for _, node := range nodeMap {
		if node.Commit.GetParent() != nil {
			if parent, ok := nodeMap[*node.Commit.GetParent()]; ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}

	var sortChildren func(n *treeNode[T])
	sortChildren = func(n *treeNode[T]) {
		sort.SliceStable(n.Children, func(i, j int) bool {
			return n.Children[i].Commit.GetID() < n.Children[j].Commit.GetID()
		})
		for _, child := range n.Children {
			sortChildren(child)
		}
	}
	for _, root := range roots {
		sortChildren(root)
	}

	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Commit.GetID() < roots[j].Commit.GetID()
	})

	return roots
}

func RenderTree[T Treeable](commits []*treeNode[T], prefix string, suffix string) []string {
	tree := []string{}
	for i, c := range commits {
		pre := strings.Repeat("| ", len(commits)-1-i)
		if c.Commit.GetParent() == nil {
			pre = ""
			if i != 0 {
				tree = append(tree, "\n---\n")
			}
		}
		tree = append(
			tree,
			fmt.Sprintf(
				"%s%s* %.8s - %s %s: %s",
				prefix, pre,
				c.Commit.GetID(),
				c.Commit.GetName(),
				c.Commit.GetVersion(),
				c.Commit.GetDescription(),
			),
		)
		if len(c.Children) > 0 {
			if len(c.Children) > 1 {
				tree = append(tree, fmt.Sprintf("%s|%s", pre, strings.Repeat("\\ ", len(c.Children)-1)))
			}
			tree = append(tree, RenderTree(c.Children, fmt.Sprintf("%s%s", prefix, pre), suffix)...)
		}
	}
	for i, line := range tree {
		tree[i] = strings.Trim(line, " -:")
	}
	return tree
}
