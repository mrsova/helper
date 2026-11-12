package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type screen int

const (
	screenMenu screen = iota
	screenInput
	screenResult
	screenTable
)

type menuModel struct {
	screen       screen
	actions      []MenuAction
	cursor       int
	menuOffset   int
	result       string
	active       MenuAction
	input        textarea.Model
	resultView   viewport.Model
	resultInited bool
	err          string
	copied       bool
	quit         bool
	width        int
	height       int

	// состояние табличного экрана
	table        *TableResult
	tableCursor  int
	tableOffset  int
	tableConfirm bool
	tableMsg     string

	// живой предпросмотр под полем ввода
	preview string
}

func initialMenu() menuModel {
	ta := textarea.New()
	ta.Placeholder = "вставьте текст и нажмите Enter"
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.Prompt = "│ "
	ta.SetWidth(80)
	ta.SetHeight(8)

	return menuModel{
		screen:  screenMenu,
		actions: Actions(),
		input:   ta,
	}
}

func (m menuModel) Init() tea.Cmd { return nil }

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		w := msg.Width - 6
		if w < 20 {
			w = 20
		}
		h := msg.Height/2 - 6
		if h < 4 {
			h = 4
		}
		if h > 20 {
			h = 20
		}
		m.input.SetWidth(w)
		m.input.SetHeight(h)

		if m.resultInited {
			vw := msg.Width - 8
			if vw < 20 {
				vw = 20
			}
			vh := msg.Height - 12
			if vh < 5 {
				vh = 5
			}
			m.resultView.Width = vw
			m.resultView.Height = vh
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.quit = true
			return m, tea.Quit
		}
		switch m.screen {
		case screenMenu:
			return m.updateMenu(msg)
		case screenInput:
			return m.updateInput(msg)
		case screenResult:
			return m.updateResult(msg)
		case screenTable:
			return m.updateTable(msg)
		}
	}
	return m, nil
}

func (m menuModel) updateMenu(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "q", "esc":
		m.quit = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.actions) {
			m.cursor++
		}
	case "enter":
		if m.cursor == len(m.actions) {
			m.quit = true
			return m, tea.Quit
		}
		a := m.actions[m.cursor]

		// Табличные действия — на интерактивный экран
		if ta, ok := a.(TableAction); ok {
			tbl, err := ta.RunTable()
			if err != nil {
				m.result = "⚠ " + err.Error()
				m.initResultView()
				m.screen = screenResult
				return m, nil
			}
			m.table = tbl
			m.tableCursor = 0
			m.tableOffset = 0
			m.tableConfirm = false
			m.tableMsg = ""
			m.screen = screenTable
			return m, nil
		}

		if a.Input() == InputText {
			m.active = a
			m.input.SetValue("")
			m.input.Placeholder = a.Placeholder()
			m.err = ""
			m.copied = false
			m.preview = ""
			m.screen = screenInput
			return m, m.input.Focus()
		}
		res, err := a.Run("")
		m.finish(a, res, err)
	}
	return m, nil
}

func (m menuModel) updateInput(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.screen = screenMenu
		m.err = ""
		m.input.Blur()
		return m, nil
	case "enter":
		res, err := m.active.Run(m.input.Value())
		m.finish(m.active, res, err)
		m.input.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(key)

	if pa, ok := m.active.(PreviewAction); ok {
		m.preview = pa.Preview(m.input.Value())
	}

	return m, cmd
}

func (m menuModel) updateResult(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "enter", "esc", "q":
		m.result = ""
		m.err = ""
		m.copied = false
		m.resultInited = false
		m.screen = screenMenu
		return m, nil
	case "c":
		if err := clipboard.WriteAll(m.result); err != nil {
			m.err = "не удалось скопировать: " + err.Error()
		} else {
			m.copied = true
			m.err = ""
		}
		return m, nil
	case "g":
		m.resultView.GotoTop()
		return m, nil
	case "G":
		m.resultView.GotoBottom()
		return m, nil
	}

	var cmd tea.Cmd
	m.resultView, cmd = m.resultView.Update(key)
	return m, cmd
}

func (m menuModel) updateTable(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.table == nil {
		m.screen = screenMenu
		return m, nil
	}

	// Режим подтверждения действия
	if m.tableConfirm {
		switch key.String() {
		case "y", "Y", "enter":
			row := m.table.Rows[m.tableCursor]
			msg, updated, err := m.table.Action(row)
			if err != nil {
				m.tableMsg = "⚠ " + err.Error()
			} else {
				m.tableMsg = msg
			}
			if updated != nil {
				m.table = updated
				if m.tableCursor >= len(m.table.Rows) && m.tableCursor > 0 {
					m.tableCursor = len(m.table.Rows) - 1
				}
				if m.tableCursor >= len(m.table.Rows) {
					m.tableCursor = 0
				}
				m.tableOffset = 0
			}
			m.tableConfirm = false
		case "n", "N", "esc":
			m.tableConfirm = false
		}
		return m, nil
	}

	switch key.String() {
	case "esc", "q":
		m.screen = screenMenu
		m.table = nil
		m.tableMsg = ""
		return m, nil
	case "up", "k":
		if m.tableCursor > 0 {
			m.tableCursor--
		}
	case "down", "j":
		if m.tableCursor < len(m.table.Rows)-1 {
			m.tableCursor++
		}
	case "g":
		m.tableCursor = 0
	case "G":
		if len(m.table.Rows) > 0 {
			m.tableCursor = len(m.table.Rows) - 1
		}
	case "enter":
		if m.table.Action != nil && len(m.table.Rows) > 0 {
			m.tableConfirm = true
		}
	}
	return m, nil
}

func (m *menuModel) finish(a MenuAction, res string, err error) {
	if err != nil {
		if a.Input() == InputText {
			m.err = err.Error()
			return
		}
		m.result = "⚠ " + err.Error()
	} else {
		m.result = res
		m.err = ""
	}
	m.initResultView()
	m.copied = false
	m.screen = screenResult
}

func (m *menuModel) initResultView() {
	w := m.width - 8
	if w < 20 {
		w = 20
	}
	h := m.height - 12
	if h < 5 {
		h = 5
	}
	m.resultView = viewport.New(w, h)
	m.resultView.SetContent(m.result)
	m.resultInited = true
}

func (m menuModel) View() string {
	if m.quit {
		return ""
	}
	switch m.screen {
	case screenInput:
		return m.viewInput()
	case screenResult:
		return m.viewResult()
	case screenTable:
		return m.viewTable()
	default:
		return m.viewMenu()
	}
}

func (m menuModel) viewMenu() string {
	title := lipgloss.NewStyle().
		Bold(true).Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1).Render("Быстрые команды")

	visible := m.height - 8
	if visible < 3 {
		visible = 3
	}
	if visible > 15 {
		visible = 15
	}

	total := len(m.actions) + 1

	if m.cursor < m.menuOffset {
		m.menuOffset = m.cursor
	}
	if m.cursor >= m.menuOffset+visible {
		m.menuOffset = m.cursor - visible + 1
	}
	if m.menuOffset < 0 {
		m.menuOffset = 0
	}
	if m.menuOffset > total-visible {
		m.menuOffset = total - visible
	}
	if m.menuOffset < 0 {
		m.menuOffset = 0
	}

	active := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).
		Bold(true).Padding(0, 1)

	normal := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC")).Padding(0, 1)

	var body string

	if m.menuOffset > 0 {
		body += lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Render(fmt.Sprintf("  ↑ ещё %d", m.menuOffset)) + "\n"
	}

	for i := m.menuOffset; i < m.menuOffset+visible && i < total; i++ {
		label := "Выход"
		if i < len(m.actions) {
			label = m.actions[i].Label()
		}
		cursor, style := "  ", normal
		if m.cursor == i {
			cursor, style = "▶ ", active
		}
		body += fmt.Sprintf("%s%s\n", cursor, style.Render(label))
	}

	if m.menuOffset+visible < total {
		body += lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Render(fmt.Sprintf("  ↓ ещё %d", total-m.menuOffset-visible)) + "\n"
	}

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).MarginTop(1).
		Render("↑/↓ — навигация • Enter — выбрать • q/Esc — выход")

	return title + "\n" + body + hint
}

func (m menuModel) viewInput() string {
	title := lipgloss.NewStyle().
		Bold(true).Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1).Render(m.active.Label())

	prompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC")).
		Render("Введите или вставьте текст. Enter — выполнить, Esc — отмена.")

	inputView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(0, 1).Render(m.input.View())

	var previewBlock string
	if m.preview != "" {
		color := "#04B575"
		if strings.HasPrefix(m.preview, "⚠") {
			color = "#FF5F87"
		}
		previewBlock = "\n" + lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color(color)).
			PaddingLeft(1).
			Foreground(lipgloss.Color(color)).
			Render(m.preview) + "\n"
	}

	var errLine string
	if m.err != "" {
		errLine = "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Render("⚠ "+m.err) + "\n"
	}

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).MarginTop(1).
		Render("Enter — выполнить • Esc — назад • Ctrl+C — выход")

	return title + "\n" + prompt + "\n\n" + inputView + previewBlock + errLine + hint
}

func (m menuModel) viewResult() string {
	title := lipgloss.NewStyle().
		Bold(true).Foreground(lipgloss.Color("#04B575")).
		MarginBottom(1).Render("Результат")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(0, 1).Render(m.resultView.View())

	var scrollInfo string
	if m.resultView.TotalLineCount() > m.resultView.VisibleLineCount() {
		pct := int(m.resultView.ScrollPercent() * 100)
		scrollInfo = fmt.Sprintf(" • %d%%", pct)
	}

	copyHint := "c — копировать"
	if m.copied {
		copyHint = "✓ скопировано"
	}

	var errLine string
	if m.err != "" {
		errLine = "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Render("⚠ "+m.err)
	}

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).MarginTop(1).
		Render("↑/↓/PgUp/PgDn — прокрутка" + scrollInfo +
			" • " + copyHint + " • Enter/Esc — назад")

	return title + "\n" + box + "\n" + errLine + "\n" + hint
}

func (m menuModel) viewTable() string {
	t := m.table
	if t == nil {
		return ""
	}

	title := lipgloss.NewStyle().
		Bold(true).Foreground(lipgloss.Color("#7D56F4")).
		MarginBottom(1).Render(t.Title)

	if len(t.Rows) == 0 {
		return title + "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Render("пусто") + "\n\n" +
			lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).
				Render("Esc — назад")
	}

	// Ширины колонок = максимум из заголовка и ячеек
	widths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		widths[i] = len(h)
	}
	for _, r := range t.Rows {
		for i, c := range r.Cells {
			if i < len(widths) && len(c) > widths[i] {
				widths[i] = len(c)
			}
		}
	}

	// Видимая область
	visible := m.height - 10
	if visible < 3 {
		visible = 3
	}
	if visible > len(t.Rows) {
		visible = len(t.Rows)
	}
	if m.tableCursor < m.tableOffset {
		m.tableOffset = m.tableCursor
	}
	if m.tableCursor >= m.tableOffset+visible {
		m.tableOffset = m.tableCursor - visible + 1
	}
	if m.tableOffset < 0 {
		m.tableOffset = 0
	}

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	active := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7D56F4")).Bold(true)
	normal := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))

	formatRow := func(cells []string) string {
		s := ""
		for i, w := range widths {
			c := ""
			if i < len(cells) {
				c = cells[i]
			}
			if len(c) > w {
				c = c[:w]
			}
			s += fmt.Sprintf("%-*s ", w, c)
		}
		return strings.TrimRight(s, " ")
	}

	body := header.Render(formatRow(t.Headers)) + "\n"
	body += lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).
		Render(strings.Repeat("─", sum(widths)+len(widths))) + "\n"

	for i := m.tableOffset; i < m.tableOffset+visible; i++ {
		line := formatRow(t.Rows[i].Cells)
		if i == m.tableCursor {
			body += active.Render(line) + "\n"
		} else {
			body += normal.Render(line) + "\n"
		}
	}

	if len(t.Rows) > visible {
		body += lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")).
			Render(fmt.Sprintf("\n  %d–%d из %d",
				m.tableOffset+1, m.tableOffset+visible, len(t.Rows))) + "\n"
	}

	var msgLine string
	if m.tableMsg != "" {
		color := "#04B575"
		if strings.HasPrefix(m.tableMsg, "⚠") {
			color = "#FF5F87"
		}
		msgLine = "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color(color)).Render(m.tableMsg) + "\n"
	}

	var confirmLine string
	if m.tableConfirm && m.table.Action != nil {
		row := t.Rows[m.tableCursor]
		confirmLine = "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFAF00")).Bold(true).
			Render(fmt.Sprintf("%s: %s? [y/N]",
				t.ActionLabel, strings.Join(row.Cells, " "))) + "\n"
	}

	actionHint := ""
	if t.Action != nil {
		actionHint = fmt.Sprintf(" • Enter — %s", t.ActionLabel)
	}

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#555555")).MarginTop(1).
		Render("↑/↓ — выбор" + actionHint + " • Esc — назад")

	return title + "\n" + body + msgLine + confirmLine + hint
}

func sum(xs []int) int {
	s := 0
	for _, x := range xs {
		s += x
	}
	return s
}

func runMenu() {
	p := tea.NewProgram(initialMenu(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка: %v\n", err)
		os.Exit(1)
	}
}

var menuCmd = &cobra.Command{
	Use:   "menu",
	Short: "Быстрые команды",
	Run: func(cmd *cobra.Command, args []string) {
		runMenu()
	},
}

func init() {
	rootCmd.AddCommand(menuCmd)
}
