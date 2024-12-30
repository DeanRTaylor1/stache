package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

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

type choice struct {
	selected bool
	label    string
	path     string
}

type model struct {
	choices      []choice
	selected     map[int]struct{}
	cursor       int
	activeColumn int // 0 for left, 1 for right
	leftScroll   int // track scroll position for left column
	rightScroll  int // track scroll position for right column
	width        int // terminal width
	height       int // terminal height
}

func newModel(dotfiles []string) *model {
	fileNames := make([]choice, len(dotfiles))
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory")
	}
	for i, file := range dotfiles {
		prefix := fmt.Sprintf("%s/", homeDir)
		selected := false
		fileNames[i] = choice{
			selected: selected,
			label:    strings.TrimPrefix(file, prefix),
			path:     fmt.Sprintf("%s/%s", homeDir, file),
		}

	}

	return &model{
		choices:  fileNames,
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			// Switch active column
			m.activeColumn = (m.activeColumn + 1) % 2
			m.cursor = 0 // Reset cursor for new column

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				// Update scroll if needed
				if m.activeColumn == 0 && m.cursor < m.leftScroll {
					m.leftScroll = m.cursor
				} else if m.activeColumn == 1 && m.cursor < m.rightScroll {
					m.rightScroll = m.cursor
				}
			}

		case "down", "j":
			maxLen := len(m.choices) - len(m.selected)
			if m.activeColumn == 1 {
				maxLen = len(m.selected)
			}
			if m.cursor < maxLen-1 {
				m.cursor++
				// Update scroll if cursor would go off screen
				contentHeight := m.height - 4
				if m.activeColumn == 0 && m.cursor >= m.leftScroll+contentHeight {
					m.leftScroll = m.cursor - contentHeight + 1
				} else if m.activeColumn == 1 && m.cursor >= m.rightScroll+contentHeight {
					m.rightScroll = m.cursor - contentHeight + 1
				}
			}

		case "enter", " ":
			if m.activeColumn == 0 {
				// Only allow selection in the left column
				_, ok := m.selected[m.cursor]
				if ok {
					delete(m.selected, m.cursor)
				} else {
					m.selected[m.cursor] = struct{}{}
				}
				m.choices[m.cursor].selected = !m.choices[m.cursor].selected
			} else {
				// In right column, space/enter removes from selection
				if m.cursor < len(m.selected) {
					// Find the actual index in choices that corresponds to this managed file
					for i, choice := range m.choices {
						if choice.selected && m.cursor == 0 {
							delete(m.selected, i)
							m.choices[i].selected = false
							break
						}
						if choice.selected {
							m.cursor--
						}
					}
				}
			}

		case "h":
			// Optional: scroll left column up one page
			if m.leftScroll > 0 {
				m.leftScroll--
			}

		case "l":
			// Optional: scroll right column up one page
			if m.rightScroll > 0 {
				m.rightScroll--
			}
		}

	// Handle window resize
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	}

	return m, nil
}

func (m model) View() string {
	// Get terminal dimensions
	w, h, err := term.GetSize(0)
	if err != nil {
		fmt.Println("Error getting terminal size")
		os.Exit(1)
	}
	contentHeight := h - 10 // subtract space for headers and footer

	// Create columns that fill height
	leftColumn := lipgloss.NewStyle().
		Width(w/2 - 2).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	if m.activeColumn == 0 {
		leftColumn = leftColumn.BorderForeground(lipgloss.Color(PastelGreen))
	}

	rightColumn := lipgloss.NewStyle().
		Width(w/2 - 2).
		Height(contentHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	if m.activeColumn == 1 {
		rightColumn = rightColumn.BorderForeground(lipgloss.Color(PastelGreen))
	}

	leftContent := "Unmanaged Files:\n\n"
	rightContent := "Managed Files:\n\n"

	// Build unmanaged files list with scrolling
	unmanaged := []string{}
	for i, choice := range m.choices {
		if _, ok := m.selected[i]; !ok {
			cursor := " "
			if m.activeColumn == 0 && m.cursor == i {
				cursor = ">"
			}
			normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
			renderedLabel := normalStyle.Render(choice.label)
			unmanaged = append(unmanaged, fmt.Sprintf("%s %s", cursor, renderedLabel))
		}
	}

	// Apply scrolling to left column
	visibleLeft := unmanaged
	if len(unmanaged) > contentHeight {
		end := m.leftScroll + contentHeight
		if end > len(unmanaged) {
			end = len(unmanaged)
		}
		visibleLeft = unmanaged[m.leftScroll:end]
	}
	leftContent += strings.Join(visibleLeft, "\n")

	// Build managed files list with scrolling
	managed := []string{}
	for i, choice := range m.choices {
		if _, ok := m.selected[i]; ok {
			cursor := " "
			if m.activeColumn == 1 && m.cursor == i {
				cursor = ">"
			}
			selectedStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(PastelGreen))
			renderedLabel := selectedStyle.Render(choice.label)
			managed = append(managed, fmt.Sprintf("%s %s", cursor, renderedLabel))
		}
	}

	// Apply scrolling to right column
	visibleRight := managed
	if len(managed) > contentHeight {
		end := m.rightScroll + contentHeight
		if end > len(managed) {
			end = len(managed)
		}
		visibleRight = managed[m.rightScroll:end]
	}
	rightContent += strings.Join(visibleRight, "\n")

	// Join columns side by side
	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn.Render(leftContent),
		rightColumn.Render(rightContent),
	)

	return columns + "\n\nPress q to quit, Tab to switch columns, Space to select\n"
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
	p := tea.NewProgram(newModel(files))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Something went wrong!")
		os.Exit(1)
	}
	println("Hello, World!")
}
