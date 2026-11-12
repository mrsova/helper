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

var registry []MenuAction

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
