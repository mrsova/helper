package cmd

import (
	"fmt"
	"helper/pkg/homedir"
	"os"
	"path/filepath"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

type qrAction struct{}

func (qrAction) Label() string       { return "QR-код (сохранить в PNG)" }
func (qrAction) Order() int          { return 70 }
func (qrAction) Input() InputMode    { return InputText }
func (qrAction) Placeholder() string { return "https://example.com" }
func (qrAction) Key() string         { return "qrcode" }

func (qrAction) Run(input string) (string, error) {
	text := strings.TrimSpace(input)
	if text == "" {
		return "", fmt.Errorf("пустой ввод")
	}

	// Создаём QR-код
	q, err := qrcode.New(text, qrcode.Medium)
	if err != nil {
		return "", err
	}

	// Генерируем PNG размером 512x512 пикселей.
	// Метод PNG возвращает []byte с изображением.
	pngBytes, err := q.PNG(512)
	if err != nil {
		return "", fmt.Errorf("не удалось сгенерировать PNG: %w", err)
	}
	dirPath := homedir.GetHomeDir("tmp")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return "", fmt.Errorf("не удалось создать папку %s: %w", dirPath, err)
	}

	filename := filepath.Join(dirPath, fmt.Sprintf("qrcode_%d.png", time.Now().Unix()))

	if err := os.WriteFile(filename, pngBytes, 0o644); err != nil {
		return "", fmt.Errorf("не удалось сохранить файл: %w", err)
	}

	absPath, _ := filepath.Abs(filename)
	return fmt.Sprintf("✅ QR-код сохранён:\n%s\n\nРазмер: %d байт\nСодержимое: %s",
		absPath, len(pngBytes), text), nil
}

// Preview оставляем без изменений (текстовое превью в терминале)
func (qrAction) Preview(input string) string {
	text := strings.TrimSpace(input)
	if text == "" {
		return ""
	}
	q, err := qrcode.New(text, qrcode.Low)
	if err != nil {
		return "⚠ " + err.Error()
	}
	q.DisableBorder = true
	ascii := q.ToSmallString(true)
	lines := strings.Split(strings.TrimRight(ascii, "\n"), "\n")
	if len(lines) > 10 {
		lines = lines[:10]
		lines = append(lines, "…")
	}
	return strings.Join(lines, "\n")
}

func init() {
	RegisterAction(qrAction{})
}
