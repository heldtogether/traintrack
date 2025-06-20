package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/heldtogether/traintrack/internal/datasets"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(logCmd)
}

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "See the datasets history",
	Run: func(cmd *cobra.Command, args []string) {
		RunLog()
	},
}

func RunLog() {

	commits, err := FetchData()
	if err != nil {
		fmt.Printf("couldn't fetch data: %s", err)
		os.Exit(1)
	}

	tree := BuildTree(commits)
	lines := RenderTree(tree, "", "")

	p := tea.NewProgram(model{lines: lines}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

type treeNode struct {
	Commit   *datasets.Dataset
	Children []*treeNode
}

func stringPtr(s string) *string {
	return &s
}

func FetchData() ([]*datasets.Dataset, error) {
	resp, err := http.Get("http://localhost:8080/datasets")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("couldn't fetch data")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data []*datasets.Dataset
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return data, err
}

func BuildTree(commits []*datasets.Dataset) []*treeNode {
	nodeMap := map[string]*treeNode{}
	var roots []*treeNode

	for _, c := range commits {
		node := &treeNode{Commit: c}
		nodeMap[c.ID] = node
	}

	for _, node := range nodeMap {
		if node.Commit.Parent != nil {
			if parent, ok := nodeMap[*node.Commit.Parent]; ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}

	var sortChildren func(n *treeNode)
	sortChildren = func(n *treeNode) {
		sort.SliceStable(n.Children, func(i, j int) bool {
			return n.Children[i].Commit.ID < n.Children[j].Commit.ID
		})
		for _, child := range n.Children {
			sortChildren(child)
		}
	}
	for _, root := range roots {
		sortChildren(root)
	}

	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Commit.ID < roots[j].Commit.ID
	})

	return roots
}

func RenderTree(commits []*treeNode, prefix string, suffix string) []string {
	tree := []string{}
	for i, c := range commits {
		pre := strings.Repeat("| ", len(commits)-1-i)
		if c.Commit.Parent == nil {
			pre = ""
			if i != 0 {
				tree = append(tree, "\n---\n")
			}
		}
		tree = append(tree, fmt.Sprintf("%s%s* %.8s - %s %s: %s", prefix, pre, c.Commit.ID, c.Commit.Name, c.Commit.Version, c.Commit.Description))
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

type model struct {
	lines    []string
	viewport viewport.Model
	ready    bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		}

		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-2)
			m.viewport.SetContent(strings.Join(m.lines, "\n"))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) View() string {
	//return m.viewport.View() + "\n\n[↑ up] [↓ down] [q to quit]"
	// scrollPercent := int(float64(m.viewport.ScrollPercent()) * 100)
	total := m.viewport.TotalLineCount()
	bottom := m.viewport.YOffset + m.viewport.Height
	if bottom > total {
		bottom = total
	}
	scrollPercent := int(float64(bottom) / float64(total) * 100)

	statusBar := fmt.Sprintf("[↑ up] [↓ down] [q to quit]%s%3d%%", strings.Repeat(" ", max(1, m.viewport.Width-32)), scrollPercent)
	return m.viewport.View() + "\n\n" + statusBar
}
