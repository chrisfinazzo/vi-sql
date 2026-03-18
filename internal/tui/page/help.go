package page

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kopecmaciej/tview"
	"github.com/kopecmaciej/vi-sql/internal/config"
	"github.com/kopecmaciej/vi-sql/internal/manager"
	"github.com/kopecmaciej/vi-sql/internal/tui/core"
)

const (
	HelpPageId = "Help"
)

// sectionOrder defines the preferred display order for key sections.
// Sections absent from this list are appended at the end.
var sectionOrder = []string{
	"Navigation", "Global", "Help", "Connection",
	"Main", "Schema", "InputBar", "Content",
	"Peeker", "QueryBar", "Index", "IndexAddForm", "Structure", "History",
}

type Help struct {
	*core.BaseElement
	*core.Flex

	style       *config.HelpStyle
	leftFlex    *core.Flex
	sectionList *core.List
	keysTable   *core.Table
	searchInput *core.InputField

	allSections      []config.OrderedKeys
	filteredSections []config.OrderedKeys
	searchMode       bool
}

func NewHelp() *Help {
	h := &Help{
		BaseElement: core.NewBaseElement(),
		Flex:        core.NewFlex(),
		leftFlex:    core.NewFlex(),
		sectionList: core.NewList(),
		keysTable:   core.NewTable(),
		searchInput: core.NewInputField(),
	}

	h.SetIdentifier(HelpPageId)
	h.SetAfterInitFunc(h.init)

	return h
}

func (h *Help) init() error {
	h.setLayout()
	h.setStyle()
	h.setKeybindings()
	h.handleEvents()
	return nil
}

func (h *Help) handleEvents() {
	go h.HandleEvents(HelpPageId, func(event manager.EventMsg) {
		switch event.Message.Type {
		case manager.StyleChanged:
			h.setStyle()
			go h.App.QueueUpdateDraw(func() {
				h.Render()
			})
		}
	})
}

func (h *Help) setLayout() {
	h.Flex.SetBorder(true)
	h.Flex.SetTitle(" Help ")
	h.Flex.SetTitleAlign(tview.AlignLeft)

	h.leftFlex.SetDirection(tview.FlexRow)

	h.sectionList.SetTitle(" Sections ")
	h.sectionList.SetBorder(true)
	h.sectionList.ShowSecondaryText(false)
	h.sectionList.SetBorderPadding(0, 0, 1, 0)

	h.keysTable.SetTitle(" Keys ")
	h.keysTable.SetBorder(true)
	h.keysTable.SetBorderPadding(0, 0, 1, 1)
	h.keysTable.SetSelectable(false, false)
	h.keysTable.SetScrollBarEnabled(true)
	h.keysTable.SetEvaluateAllRows(true)

	h.searchInput.SetLabel(" / ")
	h.searchInput.SetBorder(true)

	h.leftFlex.AddItem(h.sectionList, 0, 1, true)

	h.Flex.AddItem(h.leftFlex, 28, 0, true)
	h.Flex.AddItem(h.keysTable, 0, 1, false)
}

func (h *Help) setStyle() {
	h.style = &h.App.GetStyles().Help
	h.SetStyle(h.App.GetStyles())
	h.leftFlex.SetStyle(h.App.GetStyles())
	h.sectionList.SetStyle(h.App.GetStyles())
	h.keysTable.SetStyle(h.App.GetStyles())
	h.searchInput.SetStyle(h.App.GetStyles())

	globalBg := h.App.GetStyles().Global.BackgroundColor.Color()
	focusColor := h.App.GetStyles().Global.FocusColor.Color()
	textColor := h.App.GetStyles().Global.TextColor.Color()

	h.sectionList.SetMainTextColor(textColor)
	h.sectionList.SetSelectedStyle(tcell.StyleDefault.
		Foreground(globalBg).
		Background(focusColor))

	h.keysTable.SetScrollBarStyle(
		tcell.StyleDefault.Foreground(h.style.ScrollBarThumbColor.Color()),
		tcell.StyleDefault.Foreground(h.style.ScrollBarTrackColor.Color()),
	)
}

func (h *Help) setKeybindings() {
	k := h.App.GetKeys()

	h.sectionList.SetChangedFunc(func(index int, _, _ string, _ rune) {
		h.renderKeysForSection(index)
	})

	h.sectionList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case k.Contains(k.Help.Close, event.Name()):
			h.App.Pages.RemovePage(HelpPageId)
			return nil
		case k.Contains(k.Navigation.FocusRight, event.Name()):
			h.App.SetFocusInternal(h.keysTable)
			return nil
		case k.Contains(k.Help.Search, event.Name()):
			h.enterSearchMode()
			return nil
		case k.Contains(k.Navigation.MoveDown, event.Name()):
			curr := h.sectionList.GetCurrentItem()
			h.sectionList.SetCurrentItem(curr + 1)
			return nil
		case k.Contains(k.Navigation.MoveUp, event.Name()):
			if curr := h.sectionList.GetCurrentItem(); curr > 0 {
				h.sectionList.SetCurrentItem(curr - 1)
			}
			return nil
		}
		return event
	})

	h.keysTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case k.Contains(k.Help.Close, event.Name()):
			h.App.Pages.RemovePage(HelpPageId)
			return nil
		case k.Contains(k.Navigation.FocusLeft, event.Name()):
			h.App.SetFocusInternal(h.sectionList)
			return nil
		case k.Contains(k.Navigation.MoveDown, event.Name()):
			row, _ := h.keysTable.GetOffset()
			h.keysTable.SetOffset(row+1, 0)
			return nil
		case k.Contains(k.Navigation.MoveUp, event.Name()):
			if row, _ := h.keysTable.GetOffset(); row > 0 {
				h.keysTable.SetOffset(row-1, 0)
			}
			return nil
		}
		return event
	})

	h.searchInput.SetDoneFunc(func(key tcell.Key) {
		h.exitSearchMode(key == tcell.KeyEsc)
	})

	h.searchInput.SetChangedFunc(func(text string) {
		h.filterSections(text)
	})
}

func (h *Help) enterSearchMode() {
	h.searchMode = true
	h.leftFlex.AddItem(h.searchInput, 3, 0, true)
	h.App.SetFocusInternal(h.searchInput)
}

func (h *Help) exitSearchMode(reset bool) {
	h.searchMode = false
	h.leftFlex.RemoveItem(h.searchInput)
	if reset {
		h.searchInput.SetText("")
		h.filteredSections = h.allSections
		h.renderSectionList(0)
	}
	h.App.SetFocusInternal(h.sectionList)
}

func (h *Help) filterSections(query string) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		h.filteredSections = h.allSections
	} else {
		h.filteredSections = nil
		for _, s := range h.allSections {
			if strings.Contains(strings.ToLower(s.Element), query) {
				h.filteredSections = append(h.filteredSections, s)
				continue
			}
			for _, key := range s.Keys {
				if strings.Contains(strings.ToLower(key.Description), query) {
					h.filteredSections = append(h.filteredSections, s)
					break
				}
			}
		}
	}
	h.renderSectionList(0)
	if len(h.filteredSections) > 0 {
		h.renderKeysForSection(0)
	} else {
		h.keysTable.Clear()
	}
}

func (h *Help) Render() {
	allKeys := h.App.GetKeys().GetAvailableKeys()
	h.allSections = h.sortAndFilter(allKeys)
	h.filteredSections = h.allSections

	h.renderSectionList(0)
	if len(h.filteredSections) > 0 {
		h.renderKeysForSection(0)
	}
}

func (h *Help) sortAndFilter(keys []config.OrderedKeys) []config.OrderedKeys {
	orderIndex := make(map[string]int, len(sectionOrder))
	for i, name := range sectionOrder {
		orderIndex[name] = i
	}

	known := make([]config.OrderedKeys, len(sectionOrder))
	var unknown []config.OrderedKeys
	for _, ok := range keys {
		if len(ok.Keys) == 0 {
			continue
		}
		if idx, exists := orderIndex[ok.Element]; exists {
			known[idx] = ok
		} else {
			unknown = append(unknown, ok)
		}
	}

	var result []config.OrderedKeys
	for _, ok := range known {
		if ok.Element != "" {
			result = append(result, ok)
		}
	}
	return append(result, unknown...)
}

func (h *Help) renderSectionList(selectIdx int) {
	h.sectionList.Clear()
	for _, s := range h.filteredSections {
		name := s.Element
		h.sectionList.AddItem(name, "", 0, nil)
	}
	if len(h.filteredSections) > 0 {
		if selectIdx >= len(h.filteredSections) {
			selectIdx = 0
		}
		h.sectionList.SetCurrentItem(selectIdx)
	}
}

func (h *Help) renderKeysForSection(idx int) {
	h.keysTable.Clear()
	if idx >= len(h.filteredSections) {
		return
	}
	section := h.filteredSections[idx]
	for row, key := range section.Keys {
		keyString := formatHelpKeyString(key)
		h.keysTable.SetCell(row, 0,
			tview.NewTableCell(keyString).SetTextColor(h.style.KeyColor.Color()))
		h.keysTable.SetCell(row, 1,
			tview.NewTableCell(key.Description).SetTextColor(h.style.DescriptionColor.Color()))
	}
	h.keysTable.ScrollToBeginning()
}

func formatHelpKeyString(key config.Key) string {
	var parts []string
	if len(key.Keys) > 0 {
		parts = append(parts, strings.Join(key.Keys, ", "))
	}
	if len(key.Runes) > 0 {
		parts = append(parts, strings.Join(key.Runes, ", "))
	}
	return strings.Join(parts, ", ")
}
