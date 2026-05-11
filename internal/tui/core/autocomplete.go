package core

import (
	"strings"

	"github.com/kopecmaciej/tview"
	"github.com/kopecmaciej/vi-sql/internal/config"
	"github.com/kopecmaciej/vi-sql/internal/sql/completion"
)

// AutocompleteMaxItems is the max visible rows in any autocomplete dropdown.
const AutocompleteMaxItems = 10

// BuildAutocompleteItems converts completion symbols into display items: a
// colored icon prefix, the name padded for alignment, and (for columns) a
// right-aligned type label.
func BuildAutocompleteItems(symbols []completion.Symbol, styles *config.Styles) []tview.AutocompleteItem {
	maxLen := 0
	for _, sym := range symbols {
		if n := len(sym.Name); n > maxLen {
			maxLen = n
		}
	}
	items := make([]tview.AutocompleteItem, len(symbols))
	for i, sym := range symbols {
		items[i] = tview.AutocompleteItem{Main: buildAutocompleteDisplay(sym, maxLen, styles)}
	}
	return items
}

func buildAutocompleteDisplay(sym completion.Symbol, maxNameLen int, styles *config.Styles) string {
	icons := &styles.Icons
	var iconColor config.Style
	if sym.IsPK {
		iconColor = styles.SQLEditor.NumberColor
	} else {
		switch sym.Kind {
		case completion.KindColumn:
			iconColor = styles.Others.LeafIconColor
		case completion.KindTable:
			iconColor = styles.Others.LeafIconColor
		case completion.KindSchema:
			iconColor = styles.Global.SecondaryTextColor
		case completion.KindKeyword, completion.KindDDLObject:
			iconColor = styles.SQLEditor.KeywordColor
		case completion.KindFunction:
			iconColor = styles.SQLEditor.StringColor
		case completion.KindCTE, completion.KindAlias:
			iconColor = styles.Global.DimColor
		default:
			iconColor = styles.Global.DimColor
		}
	}

	var glyph config.Style
	if sym.IsPK {
		glyph = icons.PrimaryKey
	} else {
		switch sym.Kind {
		case completion.KindColumn:
			glyph = config.Style(icons.TypeSymbol(sym.TypeHint))
		case completion.KindTable:
			glyph = icons.Leaf
		case completion.KindSchema:
			glyph = icons.ClosedNode
		case completion.KindCTE:
			glyph = icons.CompletionCTE
		case completion.KindAlias:
			glyph = icons.CompletionAlias
		case completion.KindFunction:
			glyph = icons.CompletionFunction
		case completion.KindKeyword, completion.KindDDLObject:
			glyph = icons.CompletionKeyword
		default:
			glyph = icons.TypeDefault
		}
	}

	icon := icons.IconWithColor(glyph, iconColor)

	label := ""
	if sym.Kind == completion.KindColumn && sym.TypeHint != "" {
		label = sym.TypeHint
	}
	if label == "" {
		return icon + sym.Name
	}
	labelColor := styles.Autocomplete.SecondaryTextColor.String()
	pad := maxNameLen - len(sym.Name) + 2
	return icon + sym.Name + strings.Repeat(" ", pad) + "[" + labelColor + "]" + label + "[-]"
}
