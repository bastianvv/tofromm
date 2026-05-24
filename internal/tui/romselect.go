package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bastianvv/tofromm/internal/client"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type romItem struct {
	rom      client.Rom
	selected bool
}

func (r romItem) Title() string {
	if r.selected {
		return selectedStyle.Render("✓ " + r.rom.FsName)
	}
	return " " + r.rom.FsName
}

func (r romItem) Description() string {
	return r.rom.PlatformFsSlug
}

func (r romItem) FilterValue() string {
	return r.rom.FsName
}

type Model struct {
	list      list.Model
	selected  map[int]bool
	roms      []client.Rom
	done      bool
	cancelled bool
}

func New(roms []client.Rom) Model {

	items := make([]list.Item, len(roms))
	for i, rom := range roms {
		items[i] = romItem{rom: rom}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select ROMs to sync"
	l.SetFilteringEnabled(true)

	return Model{
		list:     l,
		selected: make(map[int]bool),
		roms:     roms,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-2)
		return m, nil

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.cancelled = true
			return m, tea.Quit

		case " ":
			item, ok := m.list.SelectedItem().(romItem)
			if !ok {
				break
			}
			m.selected[item.rom.ID] = !m.selected[item.rom.ID]
			item.selected = m.selected[item.rom.ID]
			for i, li := range m.list.Items() {
				if ri, ok := li.(romItem); ok && ri.rom.ID == item.rom.ID {
					cmd := m.list.SetItem(i, item)
					return m, cmd
				}
			}

		case "enter":
			if m.selectionCount() > 0 {
				m.done = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	status := statusStyle.Render(fmt.Sprintf(
		"%d selected  •  space: select  •  /: filter  •  enter: confirm  •  q: quit",
		m.selectionCount(),
	))
	return m.list.View() + "\n" + status
}

func (m Model) selectionCount() int {
	count := 0
	for _, v := range m.selected {
		if v {
			count++
		}
	}
	return count
}

func (m Model) SelectedRoms() []client.Rom {
	var result []client.Rom
	for _, rom := range m.roms {
		if m.selected[rom.ID] {
			result = append(result, rom)
		}
	}
	return result
}

func Run(roms []client.Rom) ([]client.Rom, error) {
	p := tea.NewProgram(New(roms), tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	result := final.(Model)
	if result.cancelled {
		return nil, nil
	}
	return result.SelectedRoms(), nil
}
