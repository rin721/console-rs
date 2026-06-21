package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type stdioPromptUI struct {
	reader *bufio.Reader
	stdout io.Writer
}

// NewPromptUI 创建基于标准输入输出流的通用交互 UI。
func NewPromptUI(stdin io.Reader, stdout io.Writer) PromptUI {
	if stdin == nil {
		stdin = strings.NewReader("")
	}
	if stdout == nil {
		stdout = io.Discard
	}
	return &stdioPromptUI{
		reader: bufio.NewReader(stdin),
		stdout: stdout,
	}
}

func newPromptUI(s streams) PromptUI {
	return NewPromptUI(s.stdin, s.stdout)
}

func (ui *stdioPromptUI) Select(ctx context.Context, prompt string, options []SelectOption) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("select prompt requires at least one option")
	}
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		fmt.Fprintln(ui.stdout, prompt)
		for index, option := range options {
			label := firstString(option.Label, option.Value)
			if option.Description != "" {
				fmt.Fprintf(ui.stdout, "  %d. %s - %s\n", index+1, label, option.Description)
			} else {
				fmt.Fprintf(ui.stdout, "  %d. %s\n", index+1, label)
			}
		}
		fmt.Fprintf(ui.stdout, "请选择，回车默认 %s: ", firstString(options[0].Label, options[0].Value))
		answer, err := ui.readLine()
		if err != nil && strings.TrimSpace(answer) == "" {
			return "", err
		}
		answer = strings.TrimSpace(answer)
		if answer == "" {
			return options[0].Value, nil
		}
		if index, err := strconv.Atoi(answer); err == nil && index >= 1 && index <= len(options) {
			return options[index-1].Value, nil
		}
		for _, option := range options {
			if strings.EqualFold(answer, option.Value) || strings.EqualFold(answer, option.Label) {
				return option.Value, nil
			}
		}
		fmt.Fprintln(ui.stdout, "无效选项，请重新输入。")
	}
}

func (ui *stdioPromptUI) Confirm(ctx context.Context, prompt string, defaultValue bool) (bool, error) {
	suffix := "[y/N]"
	if defaultValue {
		suffix = "[Y/n]"
	}
	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		fmt.Fprintf(ui.stdout, "%s %s: ", prompt, suffix)
		answer, err := ui.readLine()
		if err != nil && strings.TrimSpace(answer) == "" {
			return false, err
		}
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer == "" {
			return defaultValue, nil
		}
		switch answer {
		case "y", "yes", "true", "1", "是":
			return true, nil
		case "n", "no", "false", "0", "否":
			return false, nil
		default:
			fmt.Fprintln(ui.stdout, "请输入 yes 或 no。")
		}
	}
}

func (ui *stdioPromptUI) Input(ctx context.Context, prompt string, defaultValue string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if defaultValue != "" {
		fmt.Fprintf(ui.stdout, "%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Fprintf(ui.stdout, "%s: ", prompt)
	}
	answer, err := ui.readLine()
	if err != nil && strings.TrimSpace(answer) == "" {
		return "", err
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return defaultValue, nil
	}
	return answer, nil
}

func (ui *stdioPromptUI) Password(ctx context.Context, prompt string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	fmt.Fprintf(ui.stdout, "%s: ", prompt)
	answer, err := ui.readLine()
	if err != nil && strings.TrimSpace(answer) == "" {
		return "", err
	}
	return strings.TrimSpace(answer), nil
}

func (ui *stdioPromptUI) Info(message string) error {
	fmt.Fprintln(ui.stdout, message)
	return nil
}

func (ui *stdioPromptUI) readLine() (string, error) {
	line, err := ui.reader.ReadString('\n')
	line = strings.TrimRight(line, "\r\n")
	if err != nil {
		if err == io.EOF {
			return line, nil
		}
		return line, err
	}
	return line, nil
}
