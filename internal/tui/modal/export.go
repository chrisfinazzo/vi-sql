package modal

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/kopecmaciej/tview"
	"github.com/kopecmaciej/vi-sql/internal/manager"
	"github.com/kopecmaciej/vi-sql/internal/tui/core"
	"github.com/kopecmaciej/vi-sql/internal/util"
)

const ExportModalId = "ExportModal"

var exportFormats = []util.ExportFormat{
	util.ExportCSV,
	util.ExportJSON,
	util.ExportSQLInsert,
	util.ExportMarkdown,
}

// fmtItemFirst is the index of the first dynamic checkbox; items 0–2 are
// always Format, Filename, Path.
const fmtItemFirst = 3

// ExportModal is a single-step export dialog with format selection,
// filename/path inputs, per-format options, and a query context preview.
type ExportModal struct {
	*core.BaseElement
	*core.Flex

	form      *core.Form
	queryView *tview.TextView

	schema string
	table  string
	query  string
}

func NewExportModal() *ExportModal {
	e := &ExportModal{
		BaseElement: core.NewBaseElement(),
		Flex:        core.NewFlex(),
	}
	e.SetIdentifier(ExportModalId)
	e.SetAfterInitFunc(e.init)
	return e
}

func (e *ExportModal) init() error {
	e.handleEvents()
	return nil
}

func (e *ExportModal) handleEvents() {
	go e.HandleEvents(ExportModalId, func(event manager.EventMsg) {
		if event.Message.Type == manager.StyleChanged {
			e.applyStyle()
		}
	})
}

// Show builds and opens the export dialog for the given query/table context.
func (e *ExportModal) Show(_ context.Context, query, schema, table string) {
	e.query = query
	e.schema = schema
	e.table = table

	e.buildForm()
	e.buildLayout()
	e.applyStyle()

	wrapper := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(e.Flex, 0, 5, true).
			AddItem(nil, 0, 1, false), 0, 3, true).
		AddItem(nil, 0, 1, false)

	e.App.Pages.AddPage(ExportModalId, wrapper, true, true)
	e.App.SetFocusInternal(e.form)
}

func (e *ExportModal) buildForm() {
	e.form = core.NewForm()
	e.form.SetItemPadding(1)

	formatLabels := make([]string, len(exportFormats))
	for i, f := range exportFormats {
		formatLabels[i] = string(f)
	}

	// Pass nil as the selected callback: AddDropDown calls SetCurrentOption
	// immediately after SetOptions, which would fire the callback before
	// Filename and Path items exist. Format changes are handled in wrapDropDown.
	e.form.AddDropDown("Format:   ", formatLabels, 0, nil)
	e.form.AddInputField("Filename: ", e.defaultFilename(e.table, exportFormats[0]), 0, nil, nil)
	e.form.AddInputField("Path:     ", "~/", 0, nil, nil)

	dd, _ := e.form.GetFormItem(0).(*tview.DropDown)
	filenameField, _ := e.form.GetFormItem(1).(*tview.InputField)

	e.wrapDropDown(dd, filenameField)
	e.rebuildCheckboxes(exportFormats[0])

	e.form.AddButton(" Export ", func() { e.doExport() })
	e.form.AddButton(" Cancel ", func() { e.App.Pages.RemovePage(ExportModalId) })
	e.form.SetCancelFunc(func() { e.App.Pages.RemovePage(ExportModalId) })
	e.form.ApplyFormNavKeys(e.App.GetKeys())
}

// wrapDropDown intercepts MoveDown/MoveUp on the DropDown so j/k/arrows cycle
// through options immediately without opening the popup or writing a rune prefix.
// It also owns all format-change side-effects (extension sync, checkbox rebuild).
// FocusDown/FocusUp are handled at the form level via ApplyFormNavKeys.
func (e *ExportModal) wrapDropDown(dd *tview.DropDown, filenameField *tview.InputField) {
	keys := e.App.GetKeys()
	n := len(exportFormats)

	onChange := func(idx int) {
		dd.SetCurrentOption(idx)
		e.syncExt(filenameField, exportFormats[idx])
		e.rebuildCheckboxes(exportFormats[idx])
		e.applyStyle()
	}

	dd.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		idx, _ := dd.GetCurrentOption()
		switch {
		case keys.Contains(keys.Navigation.MoveDown, event.Name()):
			onChange((idx + 1) % n)
			return nil
		case keys.Contains(keys.Navigation.MoveUp, event.Name()):
			onChange((idx - 1 + n) % n)
			return nil
		}
		return event
	})
}

// rebuildCheckboxes removes all dynamic items (index ≥ fmtItemFirst) and adds
// back only the checkboxes that apply to the selected format.
func (e *ExportModal) rebuildCheckboxes(format util.ExportFormat) {
	for e.form.GetFormItemCount() > fmtItemFirst {
		e.form.RemoveFormItem(fmtItemFirst)
	}

	switch format {
	case util.ExportCSV, util.ExportMarkdown:
		e.form.AddCheckbox("Include Headers", true, nil)
		e.form.AddCheckbox("Compress (GZIP)", false, nil)
	case util.ExportJSON:
		e.form.AddCheckbox("Pretty Print", false, nil)
		e.form.AddCheckbox("Compress (GZIP)", false, nil)
	case util.ExportSQLInsert:
		e.form.AddCheckbox("Compress (GZIP)", false, nil)
	}
}

func (e *ExportModal) buildLayout() {
	e.Flex.Clear()
	e.Flex.SetDirection(tview.FlexColumn)
	e.Flex.SetBorder(true)
	e.Flex.SetTitle(" Export ")
	e.Flex.SetBorderPadding(0, 0, 2, 2)

	bg := e.App.GetStyles().Global.BackgroundColor.Color()

	content := tview.NewFlex().SetDirection(tview.FlexRow)
	content.SetBackgroundColor(bg)
	content.SetBorderPadding(1, 1, 0, 0)
	content.AddItem(e.form, 0, 3, true)

	if e.query != "" {
		e.queryView = tview.NewTextView()
		e.queryView.SetDynamicColors(true)
		e.queryView.SetWrap(true)
		e.queryView.SetScrollable(false)
		e.queryView.SetBorder(true)
		e.queryView.SetTitle(" Query ")
		e.queryView.SetBorderPadding(0, 0, 1, 0)
		content.AddItem(e.queryView, 4, 0, false)
	}

	e.Flex.AddItem(content, 0, 1, true)
}

func (e *ExportModal) applyStyle() {
	if e.App == nil {
		return
	}
	s := e.App.GetStyles()

	e.Flex.SetStyle(s)

	if e.form != nil {
		e.form.SetStyle(s)
		e.form.SetLabelColor(s.Global.SecondaryTextColor.Color())
		e.form.SetFieldBackgroundColor(s.Global.ContrastBackgroundColor.Color())
		e.form.SetFieldTextColor(s.Global.TextColor.Color())
	}

	if e.queryView != nil {
		e.queryView.SetBackgroundColor(s.Global.BackgroundColor.Color())
		e.queryView.SetTextColor(s.Global.DimColor.Color())
		e.queryView.SetBorderColor(s.Global.BorderColor.Color())
		e.queryView.SetTitleColor(s.Global.TitleColor.Color())
		e.queryView.SetText(colorizeSQL(e.query, &s.SQLEditor))
	}
}

func (e *ExportModal) doExport() {
	fmtIdx, _ := e.form.GetFormItem(0).(*tview.DropDown).GetCurrentOption()
	filename := strings.TrimSpace(e.form.GetFormItem(1).(*tview.InputField).GetText())
	path := strings.TrimSpace(e.form.GetFormItem(2).(*tview.InputField).GetText())

	includeHeaders := e.checkboxValue("Include Headers")
	compress := e.checkboxValue("Compress (GZIP)")
	prettyPrint := e.checkboxValue("Pretty Print")

	if filename == "" {
		return
	}
	if path == "" {
		path = "./"
	}
	if home, err := os.UserHomeDir(); err == nil {
		path = strings.Replace(path, "~", home, 1)
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	fullPath := path + filename
	if compress && !strings.HasSuffix(fullPath, ".gz") {
		fullPath += ".gz"
	}

	opts := util.ExportOptions{
		IncludeHeaders: includeHeaders,
		Compress:       compress,
		PrettyPrint:    prettyPrint,
	}

	e.App.Pages.RemovePage(ExportModalId)
	go e.performExport(fullPath, exportFormats[fmtIdx], opts)
}

// checkboxValue returns the checked state of a form checkbox by label,
// or false if the checkbox is not present (format doesn't support it).
func (e *ExportModal) checkboxValue(label string) bool {
	item := e.form.GetFormItemByLabel(label)
	cb, ok := item.(*tview.Checkbox)
	return ok && cb.IsChecked()
}

func (e *ExportModal) performExport(path string, format util.ExportFormat, opts util.ExportOptions) {
	rows, cols, err := e.Driver.ExecuteQuery(context.Background(), e.query)
	if err != nil {
		e.App.QueueUpdateDraw(func() {
			ShowError(e.App.Pages, "Export: query failed", err)
		})
		return
	}

	columns := make([]string, len(cols))
	for i, col := range cols {
		columns[i] = col.Name
	}

	f, err := os.Create(path)
	if err != nil {
		e.App.QueueUpdateDraw(func() {
			ShowError(e.App.Pages, "Export: cannot create file", err)
		})
		return
	}
	defer func() { _ = f.Close() }()

	if err := util.ExportRows(f, format, columns, rows, e.schema, e.table, opts); err != nil {
		e.App.QueueUpdateDraw(func() {
			ShowError(e.App.Pages, "Export: write failed", err)
		})
		return
	}

	e.App.QueueUpdateDraw(func() {
		ShowError(e.App.Pages, fmt.Sprintf("Exported %d rows to %s", len(rows), path), nil)
	})
}

func (e *ExportModal) syncExt(field *tview.InputField, format util.ExportFormat) {
	current := field.GetText()
	for _, ext := range []string{".csv", ".json", ".sql", ".md"} {
		current = strings.TrimSuffix(current, ext)
	}
	field.SetText(current + extForFormat(format))
}

func (e *ExportModal) defaultFilename(table string, format util.ExportFormat) string {
	name := table
	if name == "" {
		name = "query"
	}
	ts := time.Now().Format("20060102_150405")
	return fmt.Sprintf("%s_%s%s", name, ts, extForFormat(format))
}

func extForFormat(f util.ExportFormat) string {
	switch f {
	case util.ExportCSV:
		return ".csv"
	case util.ExportJSON:
		return ".json"
	case util.ExportSQLInsert:
		return ".sql"
	case util.ExportMarkdown:
		return ".md"
	default:
		return ""
	}
}
