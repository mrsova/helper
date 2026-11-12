package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var (
	jsonIndent  string
	jsonCompact bool
)

var jsonCmd = &cobra.Command{
	Use:   "json [файл|-]",
	Short: "Форматирует JSON",
	RunE: func(cmd *cobra.Command, args []string) error {
		raw, err := readJSONInput(args)
		if err != nil {
			return err
		}
		out, err := formatJSON(raw, jsonIndent, jsonCompact)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

func readJSONInput(args []string) ([]byte, error) {
	switch {
	case len(args) == 1 && args[0] == "-":
		return io.ReadAll(os.Stdin)
	case len(args) == 1:
		if _, err := os.Stat(args[0]); err == nil {
			return os.ReadFile(args[0])
		}
		return []byte(args[0]), nil
	case len(args) == 0:
		return io.ReadAll(os.Stdin)
	default:
		return nil, fmt.Errorf("ожидается один аргумент")
	}
}

func formatJSON(raw []byte, indent string, compact bool) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", fmt.Errorf("пустой ввод")
	}
	if !json.Valid(raw) {
		return "", fmt.Errorf("невалидный JSON")
	}
	var buf bytes.Buffer
	if compact {
		if err := json.Compact(&buf, raw); err != nil {
			return "", err
		}
		return buf.String(), nil
	}
	var v interface{}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return "", err
	}
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if indent == "" {
		indent = "  "
	}
	enc.SetIndent("", indent)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type jsonAction struct{}

func (jsonAction) Label() string {
	return "Форматировать JSON"
}
func (jsonAction) Order() int {
	return 1
}
func (jsonAction) Input() InputMode {
	return InputText
}
func (jsonAction) Placeholder() string {
	return `вставьте JSON, например {"a":1}`
}
func (jsonAction) Run(input string) (string, error) {
	return formatJSON([]byte(input), "  ", false)
}

func init() {
	jsonCmd.Flags().StringVarP(&jsonIndent, "indent", "i", "  ", "отступ")
	jsonCmd.Flags().BoolVarP(&jsonCompact, "compact", "c", false, "компактный вывод")
	rootCmd.AddCommand(jsonCmd)
	RegisterAction(jsonAction{})
}
