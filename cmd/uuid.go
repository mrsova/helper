package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	uuidCount  int
	uuidNoDash bool
	uuidUpper  bool
)

var uuidCmd = &cobra.Command{
	Use:   "uuid",
	Short: "Генерирует UUID v4",
	Long: `Генерирует один или несколько случайных UUID версии 4.

Примеры:
  test uuid                      # один UUID
  test uuid --count 5            # пять UUID
  test uuid --no-dash            # без дефисов
  test uuid --upper --count 3    # в верхнем регистре`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if uuidCount < 1 {
			return fmt.Errorf("--count должен быть >= 1")
		}
		if uuidCount > 10000 {
			return fmt.Errorf("--count слишком большой (максимум 10000)")
		}

		lines := make([]string, 0, uuidCount)
		for i := 0; i < uuidCount; i++ {
			u, err := newUUIDv4()
			if err != nil {
				return err
			}
			lines = append(lines, formatUUID(u, uuidNoDash, uuidUpper))
		}
		fmt.Println(strings.Join(lines, "\n"))
		return nil
	},
}

// newUUIDv4 генерирует 16 случайных байт и проставляет версию/вариант по RFC 4122.
func newUUIDv4() ([16]byte, error) {
	var u [16]byte
	if _, err := rand.Read(u[:]); err != nil {
		return u, fmt.Errorf("не удалось получить случайные байты: %w", err)
	}
	// версия 4: старшие 4 бита 7-го байта = 0100
	u[6] = (u[6] & 0x0F) | 0x40
	// вариант RFC 4122: старшие 2 бита 9-го байта = 10
	u[8] = (u[8] & 0x3F) | 0x80
	return u, nil
}

// formatUUID форматирует 16 байт в строку 8-4-4-4-12.
func formatUUID(u [16]byte, noDash, upper bool) string {
	if noDash {
		s := hex.EncodeToString(u[:])
		if upper {
			s = strings.ToUpper(s)
		}
		return s
	}
	s := fmt.Sprintf("%x-%x-%x-%x-%x",
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
	if upper {
		s = strings.ToUpper(s)
	}
	return s
}

// uuidAction реализует MenuAction.
type uuidAction struct{}

func (uuidAction) Label() string {
	return "Сгенерировать UUID"
}

func (uuidAction) Order() int {
	return 3
}

func (uuidAction) Input() InputMode {
	return InputNone
}

func (uuidAction) Placeholder() string {
	return ""
}

func (uuidAction) Run(string) (string, error) {
	u, err := newUUIDv4()
	if err != nil {
		return "", err
	}
	return formatUUID(u, false, false), nil
}

func init() {
	uuidCmd.Flags().IntVarP(&uuidCount, "count", "n", 1, "сколько UUID сгенерировать")
	uuidCmd.Flags().BoolVar(&uuidNoDash, "no-dash", false, "без дефисов")
	uuidCmd.Flags().BoolVar(&uuidUpper, "upper", false, "в верхнем регистре")
	rootCmd.AddCommand(uuidCmd)
	RegisterAction(uuidAction{})
}
