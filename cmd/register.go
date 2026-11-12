package cmd

import "sort"

type InputMode int

const (
	InputNone InputMode = iota
	InputText
)

type MenuAction interface {
	Label() string
	Order() int
	Input() InputMode
	Placeholder() string
	Run(input string) (string, error)
}

type TableAction interface {
	MenuAction
	RunTable() (*TableResult, error)
}

type PreviewAction interface {
	MenuAction
	Preview(input string) string
}

var registry []MenuAction

type TableRow struct {
	Cells []string
	Data  any
}

type TableResult struct {
	Title       string
	Headers     []string
	Rows        []TableRow
	Action      func(row TableRow) (msg string, updated *TableResult, err error)
	ActionLabel string
}

func RegisterAction(a MenuAction) {
	registry = append(registry, a)
}

func Actions() []MenuAction {
	out := make([]MenuAction, len(registry))
	copy(out, registry)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Order() < out[j].Order()
	})
	return out
}
