package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var timestampCmd = &cobra.Command{
	Use:   "unix",
	Short: "Парсить unix timestamp",
	Long:  `Парсит unix timestamp`,
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := parseUnixTimestamp(args[0])

		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

type timestampAction struct{}

func (timestampAction) Label() string {
	return "Парсинг unix timestamp"
}

func (timestampAction) Order() int {
	return 8
}

func (timestampAction) Input() InputMode {
	return InputText
}

func (timestampAction) Placeholder() string {
	return "Вставьте строку в формате 1728723808 или просто нажмите enter для просмотра текущего времени"
}

func (timestampAction) Run(input string) (string, error) {
	out, err := parseUnixTimestamp(input)
	if err != nil {
		return "", err
	}
	return out, nil
}

func parseUnixTimestamp(input string) (string, error) {
	input = strings.TrimSpace(input)

	var t time.Time

	if input == "" {
		t = time.Now()
	} else {
		if strings.Contains(input, ".") {
			f, err := strconv.ParseFloat(input, 64)
			if err != nil {
				return "", fmt.Errorf("некорректное Unix-время: %w", err)
			}
			sec := int64(f)
			nsec := int64((f - float64(sec)) * 1e9)
			t = time.Unix(sec, nsec)
		} else {
			n, err := strconv.ParseInt(input, 10, 64)
			if err != nil {
				return "", fmt.Errorf("некорректное Unix-время: %w", err)
			}
			// Определяем единицу по количеству цифр
			switch {
			case n > 1e15: // микросекунды (16 цифр)
				t = time.Unix(0, n*1e3)
			case n > 1e12: // миллисекунды (13 цифр)
				t = time.Unix(0, n*1e6)
			default: // секунды (10 цифр)
				t = time.Unix(n, 0)
			}
		}
	}

	moscow := time.FixedZone("MSK", 3*60*60)

	utcStr := t.UTC().Format("2006-01-02 15:04:05 MST")
	mskStr := t.In(moscow).Format("2006-01-02 15:04:05 MST")
	localStr := t.Local().Format("2006-01-02 15:04:05 MST")

	// ISO 8601 для удобства копирования
	isoUTC := t.UTC().Format(time.RFC3339)

	return fmt.Sprintf(
		"Unix:      %d\n"+
			"UTC:       %s\n"+
			"Москва:    %s\n"+
			"Локальное: %s\n"+
			"ISO (UTC): %s",
		t.Unix(), utcStr, mskStr, localStr, isoUTC,
	), nil
}

func init() {
	rootCmd.AddCommand(timestampCmd)
	RegisterAction(timestampAction{})
}
