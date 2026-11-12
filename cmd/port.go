package cmd

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cobra"
	"sort"
)

var portCmd = &cobra.Command{
	Use:   "port",
	Short: "Просмотр занятых портов",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("")
		return nil
	},
}

type PortInfo struct {
	Port    uint32
	PID     int32
	Name    string
	Command string
}

func getListeningPorts() ([]PortInfo, error) {
	conns, err := net.Connections("tcp")
	if err != nil {
		return nil, err
	}

	var result []PortInfo
	for _, c := range conns {
		if c.Status != "LISTEN" || c.Pid == 0 {
			continue
		}

		name := "-"
		cmdline := "-"
		if p, err := process.NewProcess(c.Pid); err == nil {
			if n, err := p.Name(); err == nil {
				name = n
			}
			if cmd, err := p.Cmdline(); err == nil {
				cmdline = cmd
			}
		}

		result = append(result, PortInfo{
			Port:    c.Laddr.Port,
			PID:     c.Pid,
			Name:    name,
			Command: cmdline,
		})
	}

	// Сортировка по номеру порта
	sort.Slice(result, func(i, j int) bool {
		return result[i].Port < result[j].Port
	})
	return result, nil
}

func portsToTable(ports []PortInfo) *TableResult {
	rows := make([]TableRow, 0, len(ports))
	for _, p := range ports {
		cmd := p.Command
		if len(cmd) > 50 {
			cmd = cmd[:47] + "..."
		}
		rows = append(rows, TableRow{
			Cells: []string{
				fmt.Sprintf("%d", p.Port),
				fmt.Sprintf("%d", p.PID),
				p.Name,
				cmd,
			},
			Data: p,
		})
	}

	return &TableResult{
		Title:       "Занятые порты",
		Headers:     []string{"PORT", "PID", "PROCESS", "COMMAND"},
		Rows:        rows,
		ActionLabel: "kill",
		Action: func(row TableRow) (string, *TableResult, error) {
			p, ok := row.Data.(PortInfo)
			if !ok {
				return "", nil, fmt.Errorf("некорректные данные строки")
			}
			if err := killProcess(p.PID); err != nil {
				return "", nil, err
			}
			updated, err := getListeningPorts()
			if err != nil {
				return fmt.Sprintf("✓ процесс %d завершён", p.PID), nil, nil
			}
			return fmt.Sprintf("✓ процесс %d завершён", p.PID), portsToTable(updated), nil
		},
	}
}

func killProcess(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("процесс с PID %d не найден: %w", pid, err)
	}
	if err := p.Terminate(); err != nil {
		return fmt.Errorf("не удалось завершить процесс: %w", err)
	}
	return nil
}

type portAction struct{}

func (portAction) Label() string {
	return "Посмотреть занятые порты"
}

func (portAction) Order() int {
	return 10
}

func (portAction) Input() InputMode {
	return InputNone
}

func (portAction) Placeholder() string {
	return ""
}

func (portAction) RunTable() (*TableResult, error) {
	ports, err := getListeningPorts()
	if err != nil {
		return nil, err
	}
	return portsToTable(ports), nil
}

func (portAction) Run(string) (string, error) {
	return "", nil
}

func init() {
	rootCmd.AddCommand(portCmd)
	RegisterAction(portAction{})
}
