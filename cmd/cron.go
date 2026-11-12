package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// cronAction — помощник по cron-выражениям
type cronAction struct{}

func (cronAction) Label() string {
	return "Cron: проверка и следующие запуски"
}
func (cronAction) Order() int       { return 60 }
func (cronAction) Input() InputMode { return InputText }
func (cronAction) Placeholder() string {
	return "*/5 * * * *  (5 полей: мин час день месяц день_недели)"
}
func (cronAction) Key() string { return "cron" }

func (cronAction) Run(input string) (string, error) {
	expr := strings.TrimSpace(input)
	if expr == "" {
		return "", fmt.Errorf("пустое выражение")
	}
	runs, err := cronNextRuns(expr, 3)
	if err != nil {
		return "", err
	}

	moscow := time.FixedZone("MSK", 3*60*60)
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Выражение: %s\n", expr))
	if d := describeCron(expr); d != "" {
		b.WriteString(fmt.Sprintf("Описание:  %s\n", d))
	}
	b.WriteString("\nБлижайшие 3 запуска:\n")
	for i, t := range runs {
		b.WriteString(fmt.Sprintf("  %d. %s  (UTC: %s)\n",
			i+1,
			t.In(moscow).Format("2006-01-02 15:04:05 MST Mon"),
			t.UTC().Format("15:04:05")))
	}
	if w := cronWarnings(expr); w != "" {
		b.WriteString("\n" + w)
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func (cronAction) Preview(input string) string {
	expr := strings.TrimSpace(input)
	if expr == "" {
		return ""
	}

	runs, err := cronNextRuns(expr, 3)
	if err != nil {
		return "⚠ " + err.Error()
	}

	moscow := time.FixedZone("MSK", 3*60*60)
	var b strings.Builder
	for i, t := range runs {
		b.WriteString(fmt.Sprintf("%d) %s\n",
			i+1,
			t.In(moscow).Format("2006-01-02 15:04 Mon")))
	}

	// Добавляем предупреждение о ловушке dom/dow прямо в превью
	if w := cronWarnings(expr); w != "" {
		b.WriteString(w)
	}

	return strings.TrimRight(b.String(), "\n")
}

func cronNextRuns(expr string, n int) ([]time.Time, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, nil // пусто — не ошибка, просто нет превью
	}

	parser := cron.NewParser(
		cron.SecondOptional | cron.Minute | cron.Hour |
			cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)
	schedule, err := parser.Parse(expr)
	if err != nil {
		return nil, err
	}

	next := time.Now()
	var runs []time.Time
	for i := 0; i < n; i++ {
		next = schedule.Next(next)
		if next.IsZero() {
			break
		}
		runs = append(runs, next)
	}
	if len(runs) == 0 {
		return nil, fmt.Errorf("выражение никогда не сработает")
	}
	return runs, nil
}

// describeCron даёт человекочитаемое описание для частых случаев.
// Для сложных выражений возвращает пустую строку — не пытаемся
// быть умнее парсера.
func describeCron(expr string) string {
	switch strings.TrimSpace(expr) {
	case "@yearly", "@annually":
		return "раз в год (1 января 00:00)"
	case "@monthly":
		return "раз в месяц (1-го числа 00:00)"
	case "@weekly":
		return "раз в неделю (воскресенье 00:00)"
	case "@daily", "@midnight":
		return "каждый день в 00:00"
	case "@hourly":
		return "каждый час"
	case "@every 1m":
		return "каждую минуту"
	}

	fields := strings.Fields(expr)
	// Стандартные 5 полей
	if len(fields) == 5 {
		min, hour, dom, mon, dow := fields[0], fields[1], fields[2], fields[3], fields[4]
		switch {
		case min == "*" && hour == "*" && dom == "*" && mon == "*" && dow == "*":
			return "каждую минуту"
		case strings.HasPrefix(min, "*/") && hour == "*" && dom == "*" && mon == "*" && dow == "*":
			return fmt.Sprintf("каждые %s минут", strings.TrimPrefix(min, "*/"))
		case min == "0" && strings.HasPrefix(hour, "*/") && dom == "*" && mon == "*" && dow == "*":
			return fmt.Sprintf("каждые %s часов", strings.TrimPrefix(hour, "*/"))
		case min == "0" && hour == "*" && dom == "*" && mon == "*" && dow == "*":
			return "каждый час (в начале часа)"
		case min != "*" && hour != "*" && dom == "*" && mon == "*" && dow == "*":
			return fmt.Sprintf("каждый день в %s:%s", pad2(hour), pad2(min))
		case min != "*" && hour != "*" && dom == "*" && mon == "*" && dow != "*":
			return fmt.Sprintf("по дням недели %s в %s:%s", dow, pad2(hour), pad2(min))
		case min != "*" && hour != "*" && dom != "*" && mon == "*" && dow == "*":
			return fmt.Sprintf("каждый месяц %s-го в %s:%s", dom, pad2(hour), pad2(min))
		}
	}
	return ""
}

func pad2(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

// cronWarnings возвращает предупреждения о типичных ошибках.
func cronWarnings(expr string) string {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return ""
	}
	dom, dow := fields[2], fields[4]

	var warnings []string

	// Когда и день месяца, и день недели заданы (не *), cron срабатывает
	// при выполнении ЛЮБОГО из условий, а не обоих. Это классическая ловушка.
	if dom != "*" && dow != "*" {
		warnings = append(warnings,
			"день месяца и день недели заданы одновременно: задача сработает "+
				"при выполнении ЛЮБОГО из условий, а не обоих")
	}

	// Пятница = 5, но многие путают с 6 (суббота) или используют 7 (воскресенье)
	if dow == "7" {
		warnings = append(warnings, "ℹ день недели 7 = воскресенье (то же, что 0)")
	}

	// Проверка на "каждую минуту" — часто случайно
	if fields[0] == "*" && fields[1] == "*" {
		warnings = append(warnings,
			"задача будет запускаться каждую минуту — проверьте, что это задумано")
	}

	if len(warnings) == 0 {
		return ""
	}
	return strings.Join(warnings, "\n") + "\n"
}

func init() {
	RegisterAction(cronAction{})
}
