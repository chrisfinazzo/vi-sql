package page

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/kopecmaciej/tview"
	"github.com/kopecmaciej/vi-sql/internal/config"
	"github.com/kopecmaciej/vi-sql/internal/manager"
	"github.com/kopecmaciej/vi-sql/internal/tui/component"
	"github.com/kopecmaciej/vi-sql/internal/tui/core"
)

const (
	WelcomePageId = "Welcome"
)

type Welcome struct {
	*core.BaseElement
	*core.Flex

	form          *core.Form
	footer        *component.Footer
	mcpEnabled    bool
	mcpOptions    *core.FormGroup
	editorEnabled bool
	editorOptions *core.FormGroup
	groups        []*core.FormGroup

	onSubmit func()
}

func NewWelcome() *Welcome {
	w := &Welcome{
		BaseElement: core.NewBaseElement(),
		Flex:        core.NewFlex(),
		form:        core.NewForm(),
		footer:      component.NewFooter(),
	}

	w.SetIdentifier(WelcomePageId)

	return w
}

func (w *Welcome) Init(app *core.App) error {
	w.App = app

	if err := w.footer.Init(app); err != nil {
		return err
	}
	w.setLayout()
	w.setStyle()
	w.form.ApplyFormNavKeys(app.GetKeys())
	w.handleEvents()

	return nil
}

func (w *Welcome) setLayout() {
	w.form.SetBorder(true)
	w.form.SetTitle(" Welcome to Vi-SQL ")
	w.form.SetTitleAlign(tview.AlignCenter)
	w.form.SetButtonsAlign(tview.AlignCenter)
	w.footer.SetCentered(true)

	w.form.AddButton("Save and Connect", func() {
		err := w.saveConfig()
		if err != nil {
			showError(w.App.Pages, "Error while saving config", err)
			return
		}
		if w.onSubmit != nil {
			w.onSubmit()
		}
	})

	w.form.AddButton("Exit", func() {
		w.App.Stop()
	})

	w.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		k := w.App.GetKeys()
		if k.Contains(k.Common.Confirm, event.Name()) {
			err := w.saveConfig()
			if err != nil {
				showError(w.App.Pages, "Error while saving config", err)
				return nil
			}
			if w.onSubmit != nil {
				w.onSubmit()
			}
			return nil
		}
		return event
	})
}

func (w *Welcome) setStyle() {
	styles := w.App.GetStyles()
	w.Flex.SetStyle(styles)
	w.form.SetStyle(styles)

	w.form.SetFieldTextColor(styles.Global.TextColor.Color())
	w.form.SetFieldBackgroundColor(styles.Global.ContrastBackgroundColor.Color())
	w.form.SetLabelColor(styles.Global.SecondaryTextColor.Color())
}

func (w *Welcome) handleEvents() {
	go w.HandleEvents(WelcomePageId, func(event manager.EventMsg) {
		switch event.Message.Type {
		case manager.StyleChanged:
			w.setStyle()
			go w.App.QueueUpdateDraw(func() {
				w.Render()
			})
		}
	})
}

func (w *Welcome) Render() {
	w.Clear()
	w.SetDirection(tview.FlexRow)

	centerFlex := tview.NewFlex()
	centerFlex.AddItem(tview.NewBox(), 0, 1, false)
	w.renderForm()
	centerFlex.AddItem(w.form, 0, 3, true)
	centerFlex.AddItem(tview.NewBox(), 0, 1, false)

	w.AddItem(centerFlex, 0, 1, true)
	w.renderFooter()
	w.AddItem(w.footer, 2, 0, false)

	if page, _ := w.App.Pages.GetFrontPage(); page == WelcomePageId {
		w.App.SetFocus(w)
	}
}

func (w *Welcome) renderFooter() {
	k := w.App.GetKeys()
	w.footer.SetKeys([]config.Key{
		k.Navigation.FocusUp,
		k.Navigation.FocusDown,
		k.Common.Confirm,
	})
}

func (w *Welcome) SetOnSubmitFunc(onSubmit func()) {
	w.onSubmit = onSubmit
}

func (w *Welcome) buildGroups() {
	if w.groups != nil {
		return
	}
	cfg := w.App.GetConfig()
	w.mcpEnabled = cfg.MCP.Enabled

	configFile, _ := cfg.GetCurrentConfigPath()

	w.mcpOptions = core.NewFormGroup(w.mcpEnabled, func() []tview.FormItem {
		mcpPort := fmt.Sprintf("%d", cfg.MCP.Port)
		if cfg.MCP.Port == 0 {
			mcpPort = "9741"
		}
		return []tview.FormItem{
			tview.NewInputField().SetLabel("MCP port").SetText(mcpPort).SetFieldWidth(10),
			tview.NewCheckbox().SetLabel("Allow writes").SetChecked(cfg.MCP.AllowWrite),
			tview.NewTextView().SetLabel("MCP URL").
				SetText(fmt.Sprintf("http://localhost:%s/mcp  (add to Claude Code via: claude mcp add --transport http vi-sql http://localhost:%s/mcp)", mcpPort, mcpPort)).
				SetSize(2, 60).SetDynamicColors(true).SetScrollable(false),
		}
	})

	w.editorEnabled = cfg.Editor.Enabled
	editorCmd, _ := cfg.GetEditorCmd()

	w.editorOptions = core.NewFormGroup(w.editorEnabled, func() []tview.FormItem {
		return []tview.FormItem{
			tview.NewTextView().SetLabel("External editor").
				SetText("Set command (vim, nano etc) or env ($ENV)").
				SetSize(1, 0).SetDynamicColors(true).SetScrollable(false),
			tview.NewInputField().SetLabel("Set editor").SetText(editorCmd).SetFieldWidth(30),
		}
	})

	w.groups = []*core.FormGroup{
		core.NewFormGroup(true, func() []tview.FormItem {
			return []tview.FormItem{
				tview.NewTextView().SetLabel("Welcome info").
					SetText("All configuration can be set in " + configFile + " file. You can also set it here.").
					SetSize(2, 0).SetDynamicColors(true).SetScrollable(false),
				tview.NewTextView().SetLabel(" ").
					SetText("----------------------------------------------------------").
					SetSize(1, 0).SetDynamicColors(true).SetScrollable(false),
				tview.NewCheckbox().SetLabel("Enable $EDITOR").SetChecked(w.editorEnabled).
					SetChangedFunc(func(checked bool) {
						w.editorEnabled = checked
						w.editorOptions.SetVisible(checked)
						w.form.RenderGroups(w.groups)
						w.form.ApplyDropdownNavKeys(w.App.GetKeys())
						if idx := w.form.GetFormItemIndex("Enable $EDITOR"); idx >= 0 {
							w.form.SetFocus(idx)
						}
						w.App.SetFocusInternal(w.form)
					}),
			}
		}),
		w.editorOptions,
		core.NewFormGroup(true, func() []tview.FormItem {
			logLevels := []string{"debug", "info", "warn", "error", "fatal", "panic"}
			return []tview.FormItem{
				tview.NewTextView().SetLabel("Logs").
					SetText("Requires restart if changed").
					SetSize(1, 0).SetDynamicColors(true).SetScrollable(false),
				tview.NewInputField().SetLabel("Log File").SetText(cfg.Log.Path).SetFieldWidth(30),
				tview.NewButtonGroup("Log Level", logLevels, getLogLevelIndex(cfg.Log.Level, logLevels), nil),
				tview.NewCheckbox().SetLabel("Nerd Font icons").SetChecked(cfg.Styles.BetterSymbols),
				tview.NewTextView().SetLabel("Show on start").
					SetText("Set pages to show on every start").
					SetSize(1, 60).SetDynamicColors(true).SetScrollable(false),
				tview.NewCheckbox().SetLabel("Connection page").SetChecked(cfg.ShowConnectionPage),
				tview.NewTextView().SetLabel("Welcome page").
					SetText("This page can be shown anytime via the -w flag").
					SetSize(1, 60).SetDynamicColors(true).SetScrollable(false),
				tview.NewTextView().SetLabel("Keybindings").
					SetText(fmt.Sprintf("Press: '%s' help page, %s to expand footer keys", w.App.GetKeys().Global.FullScreenHelp.String(), w.App.GetKeys().Global.ToggleFooter.String())).
					SetSize(1, 60).SetDynamicColors(true).SetScrollable(false),
				tview.NewTextView().SetLabel("Motions").
					SetText("Use basic vim motions or normal arrow keys to move around").
					SetSize(2, 60).SetDynamicColors(true).SetScrollable(false),
			}
		}),
		core.NewFormGroup(true, func() []tview.FormItem {
			return []tview.FormItem{
				tview.NewTextView().SetLabel(" ").
					SetText("----------------------------------------------------------").
					SetSize(1, 0).SetDynamicColors(true).SetScrollable(false),
				tview.NewTextView().SetLabel("MCP Server").
					SetText("Expose database tools to Claude Code (or any MCP client)").
					SetSize(1, 60).SetDynamicColors(true).SetScrollable(false),
				tview.NewCheckbox().SetLabel("MCP enabled").SetChecked(w.mcpEnabled).
					SetChangedFunc(func(checked bool) {
						w.mcpEnabled = checked
						w.mcpOptions.SetVisible(checked)
						w.form.RenderGroups(w.groups)
						w.form.ApplyDropdownNavKeys(w.App.GetKeys())
						if idx := w.form.GetFormItemIndex("MCP enabled"); idx >= 0 {
							w.form.SetFocus(idx)
						}
						w.App.SetFocusInternal(w.form)
					}),
			}
		}),
		w.mcpOptions,
	}
}

func (w *Welcome) renderForm() {
	w.buildGroups()
	w.form.RenderGroups(w.groups)
	w.form.ApplyDropdownNavKeys(w.App.GetKeys())
}

func (w *Welcome) saveConfig() error {
	logFile := w.form.GetFormItemByLabel("Log File").(*tview.InputField).GetText()
	_, logLevelIdx := w.form.GetFormItemByLabel("Log Level").(*tview.ButtonGroup).GetCurrentOption()
	logLevels := []string{"debug", "info", "warn", "error", "fatal", "panic"}
	logLevel := logLevels[logLevelIdx]

	c := w.App.GetConfig()

	c.Editor.Enabled = w.form.GetFormItemByLabel("Enable $EDITOR").(*tview.Checkbox).IsChecked()
	if w.editorOptions != nil && w.editorOptions.IsVisible() {
		editorCmd := w.form.GetFormItemByLabel("Set editor").(*tview.InputField).GetText()
		splitEditorCmd := strings.Split(editorCmd, "$")
		if len(splitEditorCmd) > 1 {
			c.Editor.Command = ""
			c.Editor.Env = splitEditorCmd[1]
		} else {
			c.Editor.Env = ""
			c.Editor.Command = editorCmd
		}
	}
	c.Log.Path = logFile
	c.Log.Level = logLevel
	c.ShowConnectionPage = w.form.GetFormItemByLabel("Connection page").(*tview.Checkbox).IsChecked()

	betterSymbols := w.form.GetFormItemByLabel("Nerd Font icons").(*tview.Checkbox).IsChecked()
	if betterSymbols != c.Styles.BetterSymbols {
		c.Styles.BetterSymbols = betterSymbols
		_ = w.App.SetStyle(c.Styles.CurrentStyle)
	}

	c.MCP.Enabled = w.form.GetFormItemByLabel("MCP enabled").(*tview.Checkbox).IsChecked()
	if w.mcpOptions != nil && w.mcpOptions.IsVisible() {
		mcpPort := 9741
		if _, err := fmt.Sscanf(w.form.GetFormItemByLabel("MCP port").(*tview.InputField).GetText(), "%d", &mcpPort); err != nil {
			mcpPort = 9741
		}
		c.MCP.Port = mcpPort
		c.MCP.AllowWrite = w.form.GetFormItemByLabel("Allow writes").(*tview.Checkbox).IsChecked()
	}

	return w.App.GetConfig().UpdateConfig()
}

func getLogLevelIndex(currentLevel string, levels []string) int {
	for i, level := range levels {
		if level == currentLevel {
			return i
		}
	}
	return 0
}
