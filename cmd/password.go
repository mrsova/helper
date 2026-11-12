package cmd

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	passLength  int
	passSymbols bool
	passDigits  bool
	passUpper   bool
	passLower   bool
)

const (
	lowerChars  = "abcdefghijklmnopqrstuvwxyz"
	upperChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digitChars  = "0123456789"
	symbolChars = "!@#$%^&*()-_=+[]{};:,.<>?/|~"
	defaultPass = 16
	minPassLen  = 4
	maxPassLen  = 4096
)

var passwordCmd = &cobra.Command{
	Use:   "password [длина]",
	Short: "Генерирует случайный пароль",
	Long: `Генерирует криптостойкий пароль заданной длины.

Примеры:
  test password                    # 16 символов (буквы + цифры)
  test password 32                 # 32 символа
  test password 24 --symbols       # + спецсимволы
  test password 20 --no-digits     # только буквы
  test password 32 --upper --lower --digits --symbols`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		length := passLength
		if len(args) == 1 {
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("длина должна быть числом: %w", err)
			}
			length = n
		}

		out, err := generatePassword(length, passUpper, passLower, passDigits, passSymbols)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

// generatePassword — общая функция: используется и подкомандой, и меню.
func generatePassword(length int, upper, lower, digits, symbols bool) (string, error) {
	if length < minPassLen {
		return "", fmt.Errorf("длина должна быть >= %d", minPassLen)
	}
	if length > maxPassLen {
		return "", fmt.Errorf("длина должна быть <= %d", maxPassLen)
	}

	// собираем алфавит из выбранных групп
	var alphabet string
	var required []string // по одному символу из каждой группы — гарантия разнообразия

	if upper {
		alphabet += upperChars
		required = append(required, upperChars)
	}
	if lower {
		alphabet += lowerChars
		required = append(required, lowerChars)
	}
	if digits {
		alphabet += digitChars
		required = append(required, digitChars)
	}
	if symbols {
		alphabet += symbolChars
		required = append(required, symbolChars)
	}

	if alphabet == "" {
		return "", fmt.Errorf("не выбрано ни одной группы символов")
	}
	if length < len(required) {
		return "", fmt.Errorf("длина %d меньше числа групп (%d)", length, len(required))
	}

	runes := []rune(alphabet)
	password := make([]rune, 0, length)

	// 1) по одному символу из каждой группы
	for _, group := range required {
		ch, err := randomRune([]rune(group))
		if err != nil {
			return "", err
		}
		password = append(password, ch)
	}

	// 2) добиваем остаток случайными из общего алфавита
	for len(password) < length {
		ch, err := randomRune(runes)
		if err != nil {
			return "", err
		}
		password = append(password, ch)
	}

	// 3) перемешиваем (Fisher–Yates), чтобы обязательные символы не стояли в начале
	if err := shuffleRunes(password); err != nil {
		return "", err
	}

	return string(password), nil
}

// randomRune возвращает случайный руну из набора через crypto/rand.
func randomRune(set []rune) (rune, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
	if err != nil {
		return 0, fmt.Errorf("crypto/rand: %w", err)
	}
	return set[n.Int64()], nil
}

// shuffleRunes перемешивает срез криптостойко (Fisher–Yates).
func shuffleRunes(s []rune) error {
	for i := len(s) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		k := j.Int64()
		s[i], s[k] = s[k], s[i]
	}
	return nil
}

// passLengthFromInput парсит ввод пользователя: только число или "число" с мусором.
func passLengthFromInput(input string) (int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultPass, nil
	}
	n, err := strconv.Atoi(input)
	if err != nil {
		return 0, fmt.Errorf("ожидается число (длина пароля), получено %q", input)
	}
	return n, nil
}

// passwordAction реализует MenuAction.
type passwordAction struct{}

func (passwordAction) Label() string {
	return "Сгенерировать пароль"
}

func (passwordAction) Order() int {
	return 4
}

func (passwordAction) Input() InputMode {
	return InputText
}

func (passwordAction) Placeholder() string {
	return "введите длину пароля (например, 4) и нажмите Enter"
}

func (passwordAction) Run(input string) (string, error) {
	length, err := passLengthFromInput(input)
	if err != nil {
		return "", err
	}
	return generatePassword(length, true, true, true, true)
}

func init() {
	passwordCmd.Flags().IntVarP(&passLength, "length", "l", defaultPass, "длина пароля")
	passwordCmd.Flags().BoolVar(&passUpper, "upper", true, "заглавные буквы")
	passwordCmd.Flags().BoolVar(&passLower, "lower", true, "строчные буквы")
	passwordCmd.Flags().BoolVar(&passDigits, "digits", true, "цифры")
	passwordCmd.Flags().BoolVar(&passSymbols, "symbols", false, "спецсимволы")
	passwordCmd.Flags().BoolVar(&passUpper, "no-upper", false, "исключить заглавные")
	passwordCmd.Flags().BoolVar(&passLower, "no-lower", false, "исключить строчные")
	passwordCmd.Flags().BoolVar(&passDigits, "no-digits", false, "исключить цифры")
	rootCmd.AddCommand(passwordCmd)
	RegisterAction(passwordAction{})
}
