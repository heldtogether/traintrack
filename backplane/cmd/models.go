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
	"github.com/heldtogether/traintrack/internal/models"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(modelsCmd)
}

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "See the models history",
	Run: func(cmd *cobra.Command, args []string) {
		RunModelsLog()
	},
}

func RunModelsLog() {

	commits, err := FetchModels()
	if err != nil {
		fmt.Printf("couldn't fetch data: %s\n", err)
		os.Exit(1)
	}

	tree := trees.BuildTree(commits)
	lines := trees.RenderTree(tree, "", "")

	p := tea.NewProgram(modelsModel{lines: lines}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func FetchModels() ([]*models.Model, error) {
	conf, err := LoadConfig(DefaultConfigPath)
	if err != nil {
		return nil, err
	}

	base, err := url.Parse(conf.URL)
	if err != nil {
		log.Fatalf("invalid base URL in config: %s", err)
	}
	base.Path = path.Join(base.Path, "models")
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

	var data []*models.Model
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return data, err
}

type modelsModel struct {
	lines    []string
	viewport viewport.Model
	ready    bool
}

func (m modelsModel) Init() tea.Cmd {
	return nil
}

func (m modelsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m modelsModel) View() string {
	total := m.viewport.TotalLineCount()
	bottom := m.viewport.YOffset + m.viewport.Height
	if bottom > total {
		bottom = total
	}
	scrollPercent := int(float64(bottom) / float64(total) * 100)

	statusBar := fmt.Sprintf("[↑ up] [↓ down] [q to quit]%s%3d%%", strings.Repeat(" ", max(1, m.viewport.Width-32)), scrollPercent)
	return m.viewport.View() + "\n\n" + statusBar
}
