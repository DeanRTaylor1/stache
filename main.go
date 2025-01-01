package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const _stacheDir = ".stache"

const (
	// Pastel Color Palette
	PastelRed     = "211" // Soft pink-red
	PastelGreen   = "121" // Mint green
	PastelYellow  = "229" // Soft yellow
	PastelBlue    = "153" // Sky blue
	PastelMagenta = "225" // Light magenta
	PastelCyan    = "159" // Light cyan
	PastelPink    = "218" // Rose pink
	PastelOrange  = "223" // Peach
	PastelPurple  = "183" // Lavender

	// Base colors
	TextGray      = "245" // Soft gray for regular text
	SelectionGray = "240" // Slightly darker gray for selections
	Background    = "255" // Off-white for backgrounds
)

type model struct {
	availableTable table.Model
	managedTable   table.Model
	activeTable    int // 0 for available, 1 for managed
	width          int
	height         int
}

func getManagedStyles() table.Styles {
	managedStyle := table.DefaultStyles()

	managedStyle.Header = managedStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(PastelGreen)).
		BorderBottom(true).
		Bold(false)

	managedStyle.Selected = managedStyle.Selected.
		Foreground(lipgloss.Color(PastelYellow)).
		Background(lipgloss.Color("57")).
		Bold(false)

	return managedStyle
}

func getAvailableStyles() table.Styles {
	availableStyle := table.DefaultStyles()

	availableStyle.Header = availableStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(PastelRed)).
		BorderBottom(true).
		Bold(false)

	availableStyle.Selected = availableStyle.Selected.
		Foreground(lipgloss.Color(PastelYellow)).
		Background(lipgloss.Color("57")).
		Bold(false)

	return availableStyle
}

func newModel(dotfiles []string) *model {
	columns := []table.Column{
		{Title: "File", Width: 30},
	}

	availableRows := make([]table.Row, 0)
	managedRows := make([]table.Row, 0)

	homeDir, _ := os.UserHomeDir()
	for _, file := range dotfiles {
		prefix := fmt.Sprintf("%s/", homeDir)
		label := strings.TrimPrefix(file, prefix)
		availableRows = append(availableRows, table.Row{label})
	}

	managedStyle := getManagedStyles()
	availableStyle := getAvailableStyles()

	availableTable := table.New(
		table.WithColumns(columns),
		table.WithRows(availableRows),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	availableTable.SetStyles(availableStyle)

	managedTable := table.New(
		table.WithColumns(columns),
		table.WithRows(managedRows),
		table.WithFocused(false),
		table.WithHeight(10),
	)
	managedTable.SetStyles(managedStyle)

	return &model{
		availableTable: availableTable,
		managedTable:   managedTable,
		activeTable:    0,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func moveItemBetweenTables(fromTable, toTable *table.Model) {
	if len(fromTable.Rows()) == 0 {
		return
	}

	cursor := fromTable.Cursor()
	selectedRow := fromTable.SelectedRow()

	newToRows := append(toTable.Rows(), selectedRow)
	toTable.SetRows(newToRows)

	fromRows := fromTable.Rows()
	newFromRows := append(fromRows[:cursor], fromRows[cursor+1:]...)
	fromTable.SetRows(newFromRows)

	if len(newFromRows) == 0 {
		fromTable.SetCursor(0)
	} else if cursor >= len(newFromRows) {
		fromTable.SetCursor(len(newFromRows) - 1)
	} else {
		fromTable.SetCursor(cursor)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		tableHeight := m.height - 8
		m.availableTable.SetHeight(tableHeight)
		m.managedTable.SetHeight(tableHeight)

		columnWidth := (m.width / 2) - 6
		m.availableTable.SetColumns([]table.Column{
			{Title: "Available Files", Width: columnWidth},
		})
		m.managedTable.SetColumns([]table.Column{
			{Title: "Managed by Stache", Width: columnWidth},
		})

	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.activeTable = (m.activeTable + 1) % 2
			if m.activeTable == 0 {
				m.availableTable.Focus()
				m.managedTable.Blur()
			} else {
				m.availableTable.Blur()
				m.managedTable.Focus()
			}
			return m, nil

		case "enter", " ":
			if m.activeTable == 0 {
				moveItemBetweenTables(&m.availableTable, &m.managedTable)
			} else {
				moveItemBetweenTables(&m.managedTable, &m.availableTable)
			}

		case "q", "ctrl+c":
			return m, tea.Quit

		case "x":
			m.dryRunSymLink()
		default:
			if m.activeTable == 0 {
				m.availableTable, cmd = m.availableTable.Update(msg)
			} else {
				m.managedTable, cmd = m.managedTable.Update(msg)
			}

		}

	}

	return m, cmd
}

func (m model) dryRunSymLink() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory")
		os.Exit(1)
	}
	stacheHomeDir := fmt.Sprintf("%s/%s", userHomeDir, _stacheDir)

	for _, row := range m.managedTable.Rows() {
		filename := row[0]
		fmt.Printf("Would symlink %s/%s to %s/%s\n", userHomeDir, filename, stacheHomeDir, filename)
	}
}

func (m model) View() string {
	tableStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		Padding(0, 1)

	if m.activeTable == 0 {
		tableStyle = tableStyle.BorderForeground(lipgloss.Color("86"))
	} else {
		tableStyle = tableStyle.BorderForeground(lipgloss.Color("240"))
	}
	leftTable := tableStyle.Render(m.availableTable.View())

	if m.activeTable == 1 {
		tableStyle = tableStyle.BorderForeground(lipgloss.Color("86"))
	} else {
		tableStyle = tableStyle.BorderForeground(lipgloss.Color("240"))
	}
	rightTable := tableStyle.Render(m.managedTable.View())

	tables := lipgloss.JoinHorizontal(lipgloss.Top, leftTable, rightTable)
	help := "\nPress tab to switch tables, space/enter to move items, x to save, q to quit"

	return tables + help
}

func loadDotFiles() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory")
		os.Exit(1)
	}

	files, err := os.ReadDir(homeDir)
	if err != nil {
		fmt.Println("Error reading directory")
		os.Exit(1)
	}

	var dotFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), ".") {
			dotFiles = append(dotFiles, file.Name())
		}
	}
	return dotFiles
}

func main() {
	files := loadDotFiles()
	p := tea.NewProgram(newModel(files), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}
