package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	jwtShowRaw   bool
	jwtShowClaim string
)

var jwtCmd = &cobra.Command{
	Use:   "jwt <токен>",
	Short: "Декодирует JWT (без проверки подписи)",
	Long: `Разбирает JWT на header, payload и signature.
Показывает стандартные claims (exp, iat, nbf, iss, sub, aud) с расшифровкой дат.
Подпись НЕ проверяется — только декодируется.

Примеры:
  test jwt eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjMifQ.signature
  test jwt --raw <токен>           # без форматирования payload
  test jwt --claim exp <токен>     # только конкретный claim
  echo <токен> | test jwt -`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var token string
		switch {
		case len(args) == 1 && args[0] == "-":
			b, err := readAllStdin()
			if err != nil {
				return err
			}
			token = strings.TrimSpace(string(b))
		case len(args) == 1:
			token = strings.TrimSpace(args[0])
		default:
			return fmt.Errorf("укажите JWT-токен или '-' для чтения из stdin")
		}

		out, err := decodeJWT(token, jwtShowRaw, jwtShowClaim)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

// JWTClaims — стандартные имена claims с описанием.
var jwtClaimsDocs = map[string]string{
	"iss": "issuer — кто выпустил",
	"sub": "subject — кому выпущен",
	"aud": "audience — для кого",
	"exp": "expiration — истекает",
	"nbf": "not before — не раньше",
	"iat": "issued at — выпущен",
	"jti": "JWT ID — уникальный идентификатор",
}

// decodeJWT — общая функция: используется и подкомандой, и меню.
func decodeJWT(token string, raw bool, onlyClaim string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("JWT должен состоять из 3 частей через точку, получено %d", len(parts))
	}

	header, err := decodeJWTPart(parts[0])
	if err != nil {
		return "", fmt.Errorf("header: %w", err)
	}
	payload, err := decodeJWTPart(parts[1])
	if err != nil {
		return "", fmt.Errorf("payload: %w", err)
	}

	// если запрошен конкретный claim — выводим только его
	if onlyClaim != "" {
		v, ok := payload[onlyClaim]
		if !ok {
			return "", fmt.Errorf("claim %q отсутствует", onlyClaim)
		}
		return formatClaimValue(onlyClaim, v), nil
	}

	var b strings.Builder

	b.WriteString("=== HEADER ===\n")
	b.WriteString(mustPrettyJSON(header))
	b.WriteString("\n\n=== PAYLOAD ===\n")
	if raw {
		rawJSON, _ := json.Marshal(payload)
		b.Write(rawJSON)
		b.WriteString("\n")
	} else {
		b.WriteString(mustPrettyJSON(payload))
		b.WriteString("\n")
	}

	// расшифровка стандартных claims
	if notes := explainClaims(payload); notes != "" {
		b.WriteString("\n=== CLAIMS ===\n")
		b.WriteString(notes)
	}

	b.WriteString("\n=== SIGNATURE ===\n")
	b.WriteString(parts[2])
	b.WriteString("\n(подпись не проверяется — только отображается)\n")

	return b.String(), nil
}

// decodeJWTPart декодирует одну часть (header или payload) в map.
// JWT использует Base64URL без padding.
func decodeJWTPart(s string) (map[string]interface{}, error) {
	// добавляем padding, если его нет
	if pad := len(s) % 4; pad != 0 {
		s += strings.Repeat("=", 4-pad)
	}
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		// fallback: raw (без padding)
		b, err = base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
		if err != nil {
			return nil, fmt.Errorf("base64url: %w", err)
		}
	}
	var m map[string]interface{}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.UseNumber() // сохраняем точность чисел (важно для exp/iat — они int64)
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("JSON: %w", err)
	}
	return m, nil
}

func mustPrettyJSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

// explainClaims формирует человекочитаемое описание стандартных claims.
func explainClaims(payload map[string]interface{}) string {
	var b strings.Builder
	now := time.Now()

	for _, key := range []string{"iss", "sub", "aud", "exp", "nbf", "iat", "jti"} {
		v, ok := payload[key]
		if !ok {
			continue
		}
		doc := jwtClaimsDocs[key]
		val := formatClaimValue(key, v)

		extra := ""
		switch key {
		case "exp":
			if t, ok := claimTime(v); ok {
				if now.After(t) {
					extra = fmt.Sprintf("  ← ⚠ ПРОСРОЧЕН %s назад", humanDuration(now.Sub(t)))
				} else {
					extra = fmt.Sprintf("  ← осталось %s", humanDuration(t.Sub(now)))
				}
			}
		case "nbf":
			if t, ok := claimTime(v); ok && now.Before(t) {
				extra = fmt.Sprintf("  ← ⚠ ещё не активен %s", humanDuration(t.Sub(now)))
			}
		case "iat":
			if t, ok := claimTime(v); ok {
				extra = fmt.Sprintf("  ← %s назад", humanDuration(now.Sub(t)))
			}
		}

		fmt.Fprintf(&b, "  %-4s %-35s (%s)%s\n", key, val, doc, extra)
	}
	return b.String()
}

func formatClaimValue(key string, v interface{}) string {
	// aud может быть массивом
	switch x := v.(type) {
	case []interface{}:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = fmt.Sprintf("%v", e)
		}
		return strings.Join(parts, ", ")
	case string:
		return x
	case json.Number:
		if t, ok := claimTime(v); ok {
			return fmt.Sprintf("%s (%d)", t.Format(time.RFC3339), t.Unix())
		}
		return x.String()
	default:
		return fmt.Sprintf("%v", x)
	}
}

// claimTime пытается интерпретировать значение как Unix-время (секунды).
func claimTime(v interface{}) (time.Time, bool) {
	switch x := v.(type) {
	case json.Number:
		n, err := x.Int64()
		if err != nil {
			return time.Time{}, false
		}
		return time.Unix(n, 0), true
	case float64:
		return time.Unix(int64(x), 0), true
	}
	return time.Time{}, false
}

func humanDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dд %dч", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dч %dм", hours, mins)
	default:
		return fmt.Sprintf("%dм", mins)
	}
}

func readAllStdin() ([]byte, error) {
	return ioReadAll(os.Stdin)
}

// jwtAction реализует MenuAction.
type jwtAction struct{}

func (jwtAction) Label() string {
	return "Разобрать JWT"
}

func (jwtAction) Order() int {
	return 5
}

func (jwtAction) Input() InputMode {
	return InputText
}

func (jwtAction) Placeholder() string {
	return "вставьте JWT-токен и нажмите Enter"
}
func (jwtAction) Run(input string) (string, error) {
	return decodeJWT(strings.TrimSpace(input), false, "")
}

func init() {
	jwtCmd.Flags().BoolVarP(&jwtShowRaw, "raw", "r", false, "payload без форматирования")
	jwtCmd.Flags().StringVarP(&jwtShowClaim, "claim", "c", "", "вывести только указанный claim")
	rootCmd.AddCommand(jwtCmd)
	RegisterAction(jwtAction{})
}
