package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
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

type choice struct {
	selected bool
	label    string
	path     string
}

type model struct {
	choices      []choice
	selected     map[int]choice
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
		fileNames[i] = choice{
			selected: false,
			label:    strings.TrimPrefix(file, prefix),
			path:     fmt.Sprintf("%s/%s", homeDir, file),
		}
	}

	return &model{
		choices: fileNames,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m *model) getVisibleItems() (unmanaged, managed []choice) {
	for _, item := range m.choices {
		if item.selected {
			managed = append(managed, item)
		} else {
			unmanaged = append(unmanaged, item)
		}
	}
	return unmanaged, managed
}

func (m model) dryRunSymLink() {
	// Create a temporary directory
	// Create a symlink in the temporary directory
	// Check if the symlink is valid
	// Return the result

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory")
		os.Exit(1)
	}
	stacheHomeDir := fmt.Sprintf("%s/%s", userHomeDir, _stacheDir)
	for _, choice := range m.choices {
		if choice.selected {
			fmt.Println("Symlinking to stache directory")
			fmt.Printf("Linking %s to %s\n", choice.path, stacheHomeDir)
			fmt.Printf("Final location: %s/%s\n", stacheHomeDir, choice.label)
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "x":
			m.dryRunSymLink()

		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			m.activeColumn = (m.activeColumn + 1) % 2
			m.cursor = 0
			m.leftScroll = 0
			m.rightScroll = 0

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.activeColumn == 0 && m.cursor < m.leftScroll {
					m.leftScroll = m.cursor
				} else if m.activeColumn == 1 && m.cursor < m.rightScroll {
					m.rightScroll = m.cursor
				}
			}

		case "down", "j":
			unmanaged, managed := m.getVisibleItems()
			currentLen := len(unmanaged)
			if m.activeColumn == 1 {
				currentLen = len(managed)
			}

			if m.cursor < currentLen-1 {
				m.cursor++
				contentHeight := m.height - 4
				if m.activeColumn == 0 && m.cursor >= m.leftScroll+contentHeight {
					m.leftScroll = m.cursor - contentHeight + 1
				} else if m.activeColumn == 1 && m.cursor >= m.rightScroll+contentHeight {
					m.rightScroll = m.cursor - contentHeight + 1
				}
			}

		case "enter", " ":
			unmanaged, managed := m.getVisibleItems()
			if m.activeColumn == 0 && m.cursor < len(unmanaged) {
				currentIndex := -1
				count := 0
				for i, choice := range m.choices {
					if !choice.selected {
						if count == m.cursor {
							currentIndex = i
							break
						}
						count++
					}
				}
				if currentIndex != -1 {
					m.choices[currentIndex].selected = true
				}
			} else if m.activeColumn == 1 && m.cursor < len(managed) {
				// Find the actual index in the original choices slice
				currentIndex := -1
				count := 0
				for i, choice := range m.choices {
					if choice.selected {
						if count == m.cursor {
							currentIndex = i
							break
						}
						count++
					}
				}
				if currentIndex != -1 {
					m.choices[currentIndex].selected = false
				}
			}
			// Reset cursor if it would be out of bounds
			unmanaged, managed = m.getVisibleItems()
			if m.activeColumn == 0 && m.cursor >= len(unmanaged) {
				m.cursor = len(unmanaged) - 1
				if m.cursor < 0 {
					m.cursor = 0
				}
			} else if m.activeColumn == 1 && m.cursor >= len(managed) {
				m.cursor = len(managed) - 1
				if m.cursor < 0 {
					m.cursor = 0
				}
			}
		}

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
	}

	return m, nil
}

func (m model) View() string {
	w, h, err := term.GetSize(0)
	if err != nil {
		fmt.Println("Error getting terminal size")
		os.Exit(1)
	}
	contentHeight := h - 10

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

	unmanaged, managed := m.getVisibleItems()

	// Build unmanaged files list
	unmanagedStrings := make([]string, len(unmanaged))
	for i, choice := range unmanaged {
		cursor := " "
		if m.activeColumn == 0 && m.cursor == i {
			cursor = ">"
		}
		normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		renderedLabel := normalStyle.Render(choice.label)
		unmanagedStrings[i] = fmt.Sprintf("%s %s", cursor, renderedLabel)
	}

	// Build managed files list
	managedStrings := make([]string, len(managed))
	for i, choice := range managed {
		cursor := " "
		if m.activeColumn == 1 && m.cursor == i {
			cursor = ">"
		}
		selectedStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(PastelGreen))
		renderedLabel := selectedStyle.Render(choice.label)
		managedStrings[i] = fmt.Sprintf("%s %s", cursor, renderedLabel)
	}

	// Apply scrolling
	if len(unmanagedStrings) > contentHeight {
		end := m.leftScroll + contentHeight
		if end > len(unmanagedStrings) {
			end = len(unmanagedStrings)
		}
		unmanagedStrings = unmanagedStrings[m.leftScroll:end]
	}

	if len(managedStrings) > contentHeight {
		end := m.rightScroll + contentHeight
		if end > len(managedStrings) {
			end = len(managedStrings)
		}
		managedStrings = managedStrings[m.rightScroll:end]
	}

	leftContent += strings.Join(unmanagedStrings, "\n")
	rightContent += strings.Join(managedStrings, "\n")

	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftColumn.Render(leftContent),
		rightColumn.Render(rightContent),
	)

	return columns + "\n\nPress q to quit, Tab to switch columns, Space to select, x to save\n"
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
