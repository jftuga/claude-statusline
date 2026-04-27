// Claude Code status line renderer. Reads JSON from stdin and writes
// ANSI-colored status output to stdout. Replaces the shell script version
// to avoid spawning multiple subprocesses per invocation.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"
)

const pgmName = "claude-statusline"
const pgmVersion = "1.2.0"
const pgmUrl = "https://github.com/jftuga/claude-statusline"

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
			ResetsAt       int64   `json:"resets_at"`
		} `json:"five_hour"`
		SevenDay *struct {
			UsedPercentage float64 `json:"used_percentage"`
			ResetsAt       int64   `json:"resets_at"`
		} `json:"seven_day"`
	} `json:"rate_limits"`
}

var (
	gradientLow  = [20]int{108, 108, 114, 114, 150, 150, 186, 186, 222, 222, 223, 223, 217, 217, 210, 210, 209, 209, 203, 203}
	gradientMid  = [20]int{108, 114, 150, 150, 186, 186, 222, 222, 223, 217, 217, 210, 210, 209, 203, 203, 167, 167, 131, 131}
	gradientHigh = [20]int{186, 222, 222, 223, 217, 210, 210, 209, 203, 203, 167, 167, 131, 131, 125, 125, 124, 124, 196, 196}
	gradientCrit = [20]int{209, 203, 203, 167, 167, 131, 131, 125, 125, 124, 124, 196, 196, 196, 160, 160, 160, 124, 124, 124}
)

func rateLimitColor(pct int) int {
	switch {
	case pct < 50:
		return 108
	case pct < 80:
		return 222
	default:
		return 196
	}
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	total := int(d.Seconds())
	h := total / 3600
	m := (total % 3600) / 60
	if h >= 24 {
		days := h / 24
		h = h % 24
		return fmt.Sprintf("%dd%dh", days, h)
	}
	return fmt.Sprintf("%dh%dm", h, m)
}

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
	var showVersion, noModel, noBar, noTokens, noCached, noCost, no5h, no7d bool
	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.BoolVar(&noModel, "no-model", false, "hide model name and effort level")
	flag.BoolVar(&noBar, "no-bar", false, "hide progress bar and percentage")
	flag.BoolVar(&noTokens, "no-tokens", false, "hide token counter")
	flag.BoolVar(&noCached, "no-cached", false, "hide cache indicator")
	flag.BoolVar(&noCost, "no-cost", false, "hide session cost")
	flag.BoolVar(&no5h, "no-5h", false, "hide 5-hour rate limit")
	flag.BoolVar(&no7d, "no-7d", false, "hide 7-day rate limit")
	flag.Parse()

	if showVersion {
		fmt.Printf("%s v%s\n", pgmName, pgmVersion)
		fmt.Println(pgmUrl)
		return
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "statusline: read stdin: %v\n", err)
		os.Exit(1)
	}

	var input statusInput
	if err := json.Unmarshal(data, &input); err != nil {
		fmt.Fprintf(os.Stderr, "statusline: parse json: %v\n", err)
		os.Exit(1)
	}

	usage := input.ContextWindow.CurrentUsage
	var sections []string

	if !noModel {
		var s strings.Builder
		modelShort := strings.TrimPrefix(input.Model.DisplayName, "Claude ")
		fmt.Fprintf(&s, "\033[38;5;141m◆\033[0m \033[1;38;5;183m%s\033[0m", modelShort)
		if input.Effort != nil && input.Effort.Level != "" {
			fmt.Fprintf(&s, " \033[38;5;141m(%s)\033[0m", input.Effort.Level)
		}
		sections = append(sections, s.String())
	}

	if !noBar && input.ContextWindow.UsedPercentage != nil {
		var s strings.Builder
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
		for i := range 20 {
			if i < filled {
				fmt.Fprintf(&s, "\033[38;5;%dm━\033[0m", gradient[i])
			} else {
				s.WriteString("\033[38;5;238m╌\033[0m")
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
		fmt.Fprintf(&s, " \033[1;38;5;%dm%d%%\033[0m", pctColor, usedInt)
		sections = append(sections, s.String())
	}

	if !noTokens {
		totalTokens := usage.CacheReadInputTokens + usage.CacheCreationInputTokens + usage.InputTokens + usage.OutputTokens
		sections = append(sections, fmt.Sprintf("\033[38;5;75m⟐\033[0m \033[38;5;117m%s\033[38;5;240m/\033[38;5;60m%s\033[0m", formatTokens(totalTokens), formatTokens(input.ContextWindow.ContextWindowSize)))
	}

	if !noCached && usage.CacheReadInputTokens > 0 {
		sections = append(sections, fmt.Sprintf("\033[38;5;220m⚡\033[38;5;179m%dk cached\033[0m", usage.CacheReadInputTokens/1000))
	}

	if !noCost {
		sections = append(sections, fmt.Sprintf("\033[38;5;156m$%.2f\033[0m", input.Cost.TotalCostUSD))
	}

	if input.RateLimits != nil {
		var rateParts []string
		if !no5h && input.RateLimits.FiveHour != nil {
			pct := int(math.Round(input.RateLimits.FiveHour.UsedPercentage))
			part := fmt.Sprintf("\033[38;5;240m5h:\033[0m \033[38;5;%dm%d%%\033[0m", rateLimitColor(pct), pct)
			if input.RateLimits.FiveHour.ResetsAt > 0 {
				remaining := time.Unix(input.RateLimits.FiveHour.ResetsAt, 0).Sub(time.Now())
				part += fmt.Sprintf(" \033[38;5;245m⟳ %s\033[0m", formatDuration(remaining))
			}
			rateParts = append(rateParts, part)
		}
		if !no7d && input.RateLimits.SevenDay != nil {
			pct := int(math.Round(input.RateLimits.SevenDay.UsedPercentage))
			part := fmt.Sprintf("\033[38;5;240m7d:\033[0m \033[38;5;%dm%d%%\033[0m", rateLimitColor(pct), pct)
			if input.RateLimits.SevenDay.ResetsAt > 0 {
				remaining := time.Unix(input.RateLimits.SevenDay.ResetsAt, 0).Sub(time.Now())
				part += fmt.Sprintf(" \033[38;5;245m⟳ %s\033[0m", formatDuration(remaining))
			}
			rateParts = append(rateParts, part)
		}
		if len(rateParts) > 0 {
			sections = append(sections, strings.Join(rateParts, " \033[38;5;240m|\033[0m "))
		}
	}

	sep := " \033[38;5;240m│\033[0m "
	fmt.Print(strings.Join(sections, sep))
}
