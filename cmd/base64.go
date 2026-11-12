package cmd

import (
	"encoding/base64"
	"fmt"

	"github.com/spf13/cobra"
)

var base64Cmd = &cobra.Command{
	Use:   "base64 <строка>",
	Short: "Расшифровывает Base64-строку",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		out, err := decodeBase64(args[0])
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

func decodeBase64(s string) (string, error) {
	decoders := []*base64.Encoding{
		base64.StdEncoding, base64.URLEncoding,
		base64.RawStdEncoding, base64.RawURLEncoding,
	}
	var lastErr error
	for _, dec := range decoders {
		if b, err := dec.DecodeString(s); err == nil {
			return string(b), nil
		} else {
			lastErr = err
		}
	}
	return "", fmt.Errorf("не удалось декодировать Base64: %w", lastErr)
}

type base64Action struct{}

func (base64Action) Label() string {
	return "Расшифровать Base64"
}

func (base64Action) Order() int {
	return 2
}

func (base64Action) Input() InputMode {
	return InputText
}
func (base64Action) Placeholder() string {
	return "вставьте Base64-строку и нажмите Enter"
}

func (base64Action) Run(input string) (string, error) {
	return decodeBase64(input)
}

func init() {
	rootCmd.AddCommand(base64Cmd)
	RegisterAction(base64Action{})
}
