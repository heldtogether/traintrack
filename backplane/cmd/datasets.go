package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/heldtogether/traintrack/cmd/trees"
	"github.com/heldtogether/traintrack/internal/auth"
	"github.com/heldtogether/traintrack/internal/datasets"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(datasetsCmd)
}

var datasetsCmd = &cobra.Command{
	Use:   "datasets",
	Short: "See the datasets history",
	Run: func(cmd *cobra.Command, args []string) {
		RunDatasetsLog()
	},
}

func RunDatasetsLog() {

	commits, err := FetchDatasets()
	if err != nil {
		fmt.Printf("couldn't fetch data: %s\n", err)
		os.Exit(1)
	}

	tree := trees.BuildTree(commits)
	lines := trees.RenderTree(tree, "", "")

	p := tea.NewProgram(datasetsModel{lines: lines}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func FetchDatasets() ([]*datasets.Dataset, error) {
	conf, err := LoadConfig(DefaultConfigPath)
	if err != nil {
		return nil, err
	}

	base, err := url.Parse(conf.URL)
	if err != nil {
		log.Fatalf("invalid base URL in config: %s", err)
	}
	base.Path = path.Join(base.Path, "datasets")
	url := base.String()

	token, err := auth.LoadToken(auth.DefaultTokenPath)
	if err != nil {
		return nil, err
	}
	bearer := "Bearer " + token.AccessToken

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", bearer)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%d - %s", resp.StatusCode, http.StatusText(resp.StatusCode))
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

type datasetsModel struct {
	lines    []string
	viewport viewport.Model
	ready    bool
}

func (m datasetsModel) Init() tea.Cmd {
	return nil
}

func (m datasetsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m datasetsModel) View() string {
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
