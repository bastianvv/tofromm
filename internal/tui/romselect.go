package tui

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/bastianvv/tofromm/internal/client"
	"github.com/bastianvv/tofromm/internal/emulator"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type stage int

const (
	stagePlatform stage = iota
	stageLoading
	stageRoms
)

type platformItem struct {
	platform client.Platform
}

func (p platformItem) Title() string {
	return p.platform.Name
}

func (p platformItem) Description() string {
	return p.platform.FsSlug
}

func (p platformItem) FilterValue() string {
	return p.platform.Name
}

type romsLoadedMsg struct {
	roms      []client.Rom
	syncedIDs map[int]bool
	err       error
}

type Model struct {
	stage      stage
	c          *client.Client
	emuConfigs map[string]emulator.Config

	platList  list.Model
	platforms []client.Platform

	list      list.Model
	roms      []client.Rom
	syncedIDs map[int]bool
	selected  map[int]bool

	done      bool
	cancelled bool
	err       error
}

type romItem struct {
	rom      client.Rom
	selected bool
}

func (r romItem) Title() string {
	if r.selected {
		return selectedStyle.Render("✓ " + r.rom.FsName)
	}
	return "  " + r.rom.FsName
}

func (r romItem) Description() string {
	return r.rom.PlatformFsSlug
}

func (r romItem) FilterValue() string {
	return r.rom.FsName
}

func New(c *client.Client, emuConfigs map[string]emulator.Config, platforms []client.Platform) Model {

	items := make([]list.Item, len(platforms))
	for i, p := range platforms {
		items[i] = platformItem{platform: p}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select a platform"
	l.SetFilteringEnabled(true)

	return Model{
		stage:      stagePlatform,
		c:          c,
		emuConfigs: emuConfigs,
		platforms:  platforms,
		platList:   l,
		selected:   make(map[int]bool),
		syncedIDs:  make(map[int]bool),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.platList.SetSize(msg.Width, msg.Height-2)
		if m.stage == stageRoms {
			m.list.SetSize(msg.Width, msg.Height-2)
		}
		return m, nil

	case romsLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.cancelled = true
			return m, tea.Quit
		}
		items := make([]list.Item, len(msg.roms))
		for i, rom := range msg.roms {
			items[i] = romItem{rom: rom, selected: msg.syncedIDs[rom.ID]}
			if msg.syncedIDs[rom.ID] {
				m.selected[rom.ID] = true
			}
		}
		l := list.New(items, list.NewDefaultDelegate(), m.platList.Width(), m.platList.Height())
		l.Title = "Select ROMs to sync"
		l.SetFilteringEnabled(true)
		m.list = l
		m.roms = msg.roms
		m.syncedIDs = msg.syncedIDs
		m.stage = stageRoms
		return m, nil

	case tea.KeyMsg:
		switch m.stage {
		case stagePlatform:
			return m.updatePlatform(msg)
		case stageRoms:
			return m.updateRoms(msg)
		}
	}

	var cmd tea.Cmd
	switch m.stage {
	case stagePlatform:
		m.platList, cmd = m.platList.Update(msg)
	case stageRoms:
		m.list, cmd = m.list.Update(msg)
	}
	return m, cmd
}

func (m Model) updatePlatform(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.platList.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.platList, cmd = m.platList.Update(msg)
		return m, cmd
	}
	switch msg.String() {
	case "ctrl+c", "q":
		m.cancelled = true
		return m, tea.Quit
	case "enter", " ":
		item, ok := m.platList.SelectedItem().(platformItem)
		if !ok {
			break
		}
		m.stage = stageLoading
		return m, m.fetchRoms(item.platform)
	}
	var cmd tea.Cmd
	m.platList, cmd = m.platList.Update(msg)
	return m, cmd
}

func (m Model) updateRoms(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.list.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
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
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) fetchRoms(platform client.Platform) tea.Cmd {
	return func() tea.Msg {
		roms, err := m.c.GetRomsByPlatform(platform.ID)
		if err != nil {
			return romsLoadedMsg{err: err}
		}
		romNameIndex := make(map[string]int, len(roms))
		for _, rom := range roms {
			romNameIndex[rom.FsNameNoExt] = rom.ID
		}
		syncedIDs := make(map[int]bool)
		for kind, cfg := range m.emuConfigs {
			for _, slug := range cfg.Platforms {
				if slug != platform.FsSlug {
					continue
				}
				emu, e := emulator.New(kind, cfg)
				if e != nil {
					continue
				}
				for _, s := range emulator.ScanSaves(emu, []string{platform.FsSlug}, romNameIndex) {
					syncedIDs[s.RomID] = true
				}
				break
			}
		}
		sort.SliceStable(roms, func(i, j int) bool {
			return syncedIDs[roms[i].ID] && !syncedIDs[roms[j].ID]
		})
		return romsLoadedMsg{roms: roms, syncedIDs: syncedIDs}
	}

}

func (m Model) View() string {
	switch m.stage {
	case stagePlatform:
		return m.platList.View()
	case stageLoading:
		return "\n Loading ROMs..."
	case stageRoms:
		status := statusStyle.Render(fmt.Sprintf(
			"%d selected  •  space: select  •  /: filter  •  enter: confirm  •  q: quit",
			m.selectionCount(),
		))
		return m.list.View() + "\n" + status
	}

	return ""
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

func Run(c *client.Client, emuConfigs map[string]emulator.Config, platforms []client.Platform) ([]client.Rom, error) {
	p := tea.NewProgram(New(c, emuConfigs, platforms), tea.WithAltScreen())
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
