// Claude Code status line renderer. Reads JSON from stdin and writes
// ANSI-colored status output to stdout. Replaces the shell script version
// to avoid spawning multiple subprocesses per invocation.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

type statusInput struct {
	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	ContextWindow struct {
		UsedPercentage    *float64 `json:"used_percentage"`
		ContextWindowSize int      `json:"context_window_size"`
		CurrentUsage      struct {
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
		} `json:"current_usage"`
	} `json:"context_window"`
	Effort *struct {
		Level string `json:"level"`
	} `json:"effort"`
	Cost struct {
		TotalCostUSD float64 `json:"total_cost_usd"`
	} `json:"cost"`
	RateLimits *struct {
		FiveHour *struct {
			UsedPercentage float64 `json:"used_percentage"`
		} `json:"five_hour"`
	} `json:"rate_limits"`
}

var (
	gradientLow  = [20]int{108, 108, 114, 114, 150, 150, 186, 186, 222, 222, 223, 223, 217, 217, 210, 210, 209, 209, 203, 203}
	gradientMid  = [20]int{108, 114, 150, 150, 186, 186, 222, 222, 223, 217, 217, 210, 210, 209, 203, 203, 167, 167, 131, 131}
	gradientHigh = [20]int{186, 222, 222, 223, 217, 210, 210, 209, 203, 203, 167, 167, 131, 131, 125, 125, 124, 124, 196, 196}
	gradientCrit = [20]int{209, 203, 203, 167, 167, 131, 131, 125, 125, 124, 124, 196, 196, 196, 160, 160, 160, 124, 124, 124}
)

func formatTokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%dk", n/1000)
	}
	return fmt.Sprintf("%d", n)
}

func main() {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(1)
	}

	var input statusInput
	if err := json.Unmarshal(data, &input); err != nil {
		os.Exit(1)
	}

	var out strings.Builder

	modelShort := strings.TrimPrefix(input.Model.DisplayName, "Claude ")
	fmt.Fprintf(&out, "\033[38;5;141m◆\033[0m \033[1;38;5;183m%s\033[0m", modelShort)

	if input.Effort != nil && input.Effort.Level != "" {
		fmt.Fprintf(&out, " \033[38;5;141m(%s)\033[0m", input.Effort.Level)
	}

	if input.ContextWindow.UsedPercentage != nil {
		usedPct := *input.ContextWindow.UsedPercentage
		usedInt := int(math.Round(usedPct))

		var gradient [20]int
		switch {
		case usedInt < 40:
			gradient = gradientLow
		case usedInt < 70:
			gradient = gradientMid
		case usedInt < 90:
			gradient = gradientHigh
		default:
			gradient = gradientCrit
		}

		filled := (usedInt * 20) / 100

		out.WriteString(" \033[38;5;240m│\033[0m ")
		for i := range 20 {
			if i < filled {
				fmt.Fprintf(&out, "\033[38;5;%dm━\033[0m", gradient[i])
			} else {
				out.WriteString("\033[38;5;238m╌\033[0m")
			}
		}

		var pctColor int
		switch {
		case usedInt < 40:
			pctColor = 108
		case usedInt < 70:
			pctColor = 222
		case usedInt < 90:
			pctColor = 209
		default:
			pctColor = 196
		}

		fmt.Fprintf(&out, " \033[1;38;5;%dm%d%%\033[0m", pctColor, usedInt)
	}

	usage := input.ContextWindow.CurrentUsage
	totalTokens := usage.CacheReadInputTokens + usage.CacheCreationInputTokens + usage.InputTokens + usage.OutputTokens
	fmt.Fprintf(&out, " \033[38;5;240m│\033[0m \033[38;5;75m⟐\033[0m \033[38;5;117m%s\033[38;5;240m/\033[38;5;60m%s\033[0m", formatTokens(totalTokens), formatTokens(input.ContextWindow.ContextWindowSize))

	if usage.CacheReadInputTokens > 0 {
		fmt.Fprintf(&out, " \033[38;5;240m│\033[0m \033[38;5;220m⚡\033[38;5;179m%dk cached\033[0m", usage.CacheReadInputTokens/1000)
	}

	fmt.Fprintf(&out, " \033[38;5;240m│\033[0m \033[38;5;156m$%.2f\033[0m", input.Cost.TotalCostUSD)

	if input.RateLimits != nil && input.RateLimits.FiveHour != nil {
		pct := int(math.Round(input.RateLimits.FiveHour.UsedPercentage))
		var color int
		switch {
		case pct < 50:
			color = 108
		case pct < 80:
			color = 222
		default:
			color = 196
		}
		fmt.Fprintf(&out, " \033[38;5;240m│\033[0m \033[38;5;%dm(%d%%)\033[0m", color, pct)
	}

	fmt.Print(out.String())
}
