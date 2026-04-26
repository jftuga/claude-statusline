#!/bin/bash

# Claude Code Status Line — Aesthetic Edition
# Gradient progress bar · Color-coded sections · Unicode polish

input=$(cat)

# ── Extract data ──────────────────────────────────────────
model_name=$(echo "$input" | jq -r '.model.display_name')
ctx_used_pct=$(echo "$input" | jq -r '.context_window.used_percentage // empty')
ctx_window_size=$(echo "$input" | jq -r '.context_window.context_window_size // 0')

cache_read=$(echo "$input" | jq -r '.context_window.current_usage.cache_read_input_tokens // 0')
cache_creation=$(echo "$input" | jq -r '.context_window.current_usage.cache_creation_input_tokens // 0')
input_tokens=$(echo "$input" | jq -r '.context_window.current_usage.input_tokens // 0')
output_tokens=$(echo "$input" | jq -r '.context_window.current_usage.output_tokens // 0')

# ── Model badge ───────────────────────────────────────────
model_short=$(echo "$model_name" | sed 's/Claude //')
effort=$(echo "$input" | jq -r '.effort.level // empty')
out="\033[38;5;141m◆\033[0m \033[1;38;5;183m${model_short}\033[0m"
if [ -n "$effort" ]; then
  out+=" \033[38;5;141m(${effort})\033[0m"
fi

# ── Gradient progress bar ─────────────────────────────────
if [ -n "$ctx_used_pct" ]; then
  used_int=$(printf "%.0f" "$ctx_used_pct")

  # Gradient colors: green → cyan → yellow → orange → red
  # Maps to 256-color codes across the bar
  gradient_low=(108 108 114 114 150 150 186 186 222 222 223 223 217 217 210 210 209 209 203 203)
  gradient_mid=(108 114 150 150 186 186 222 222 223 217 217 210 210 209 203 203 167 167 131 131)
  gradient_high=(186 222 222 223 217 210 210 209 203 203 167 167 131 131 125 125 124 124 196 196)
  gradient_crit=(209 203 203 167 167 131 131 125 125 124 124 196 196 196 160 160 160 124 124 124)

  # Pick gradient based on usage level
  if [ "$used_int" -lt 40 ]; then
    gradient=("${gradient_low[@]}")
  elif [ "$used_int" -lt 70 ]; then
    gradient=("${gradient_mid[@]}")
  elif [ "$used_int" -lt 90 ]; then
    gradient=("${gradient_high[@]}")
  else
    gradient=("${gradient_crit[@]}")
  fi

  segments=20
  filled=$(( (used_int * segments) / 100 ))

  bar=""
  for ((i=0; i<segments; i++)); do
    color="${gradient[$i]}"
    if [ $i -lt $filled ]; then
      bar+="\033[38;5;${color}m━\033[0m"
    else
      bar+="\033[38;5;238m╌\033[0m"
    fi
  done

  # Percentage color matches bar state
  if [ "$used_int" -lt 40 ]; then
    pct_color="108"
  elif [ "$used_int" -lt 70 ]; then
    pct_color="222"
  elif [ "$used_int" -lt 90 ]; then
    pct_color="209"
  else
    pct_color="196"
  fi

  out+=" \033[38;5;240m│\033[0m ${bar} \033[1;38;5;${pct_color}m${used_int}%%\033[0m"
fi

# ── Token counter ─────────────────────────────────────────
total_tokens=$(awk "BEGIN {printf \"%.0f\", $cache_read + $cache_creation + $input_tokens + $output_tokens}")

if [ "$total_tokens" -ge 1000000 ]; then
  tokens_display=$(awk "BEGIN {printf \"%.1f\", $total_tokens / 1000000}")M
elif [ "$total_tokens" -ge 1000 ]; then
  tokens_display=$(awk "BEGIN {printf \"%.0f\", $total_tokens / 1000}")k
else
  tokens_display="${total_tokens}"
fi

if [ "$ctx_window_size" -ge 1000000 ]; then
  ctx_display=$(awk "BEGIN {printf \"%.1f\", $ctx_window_size / 1000000}")M
elif [ "$ctx_window_size" -ge 1000 ]; then
  ctx_display=$(awk "BEGIN {printf \"%.0f\", $ctx_window_size / 1000}")k
else
  ctx_display="${ctx_window_size}"
fi

out+=" \033[38;5;240m│\033[0m \033[38;5;75m⟐\033[0m \033[38;5;117m${tokens_display}\033[38;5;240m/\033[38;5;60m${ctx_display}\033[0m"

# ── Cache indicator ───────────────────────────────────────
if [ "$cache_read" -gt 0 ]; then
  cache_k=$(awk "BEGIN {printf \"%.0f\", $cache_read / 1000}")
  out+=" \033[38;5;240m│\033[0m \033[38;5;220m⚡\033[38;5;179m${cache_k}k cached\033[0m"
fi

# ── Session cost ──────────────────────────────────────────
cost_usd=$(echo "$input" | jq -r '.cost.total_cost_usd // 0')
cost_display=$(awk "BEGIN {printf \"%.2f\", $cost_usd}")
out+=" \033[38;5;240m│\033[0m \033[38;5;156m\$${cost_display}\033[0m"

# ── Rate limits ───────────────────────────────────────────
rate_5h=$(echo "$input" | jq -r '.rate_limits.five_hour.used_percentage // empty')
rate_5h_resets=$(echo "$input" | jq -r '.rate_limits.five_hour.resets_at // empty')
rate_7d=$(echo "$input" | jq -r '.rate_limits.seven_day.used_percentage // empty')
rate_7d_resets=$(echo "$input" | jq -r '.rate_limits.seven_day.resets_at // empty')

rate_limit_color() {
  if [ "$1" -lt 50 ]; then
    echo "108"
  elif [ "$1" -lt 80 ]; then
    echo "222"
  else
    echo "196"
  fi
}

format_reset_duration() {
  local resets_at=$1
  local now
  now=$(date +%s)
  local remaining=$((resets_at - now))
  if [ "$remaining" -le 0 ]; then
    echo "now"
  else
    local hours=$((remaining / 3600))
    local minutes=$(( (remaining % 3600) / 60 ))
    if [ "$hours" -ge 24 ]; then
      local days=$((hours / 24))
      hours=$((hours % 24))
      echo "${days}d${hours}h"
    else
      echo "${hours}h${minutes}m"
    fi
  fi
}

rate_parts=""
if [ -n "$rate_5h" ]; then
  rate_int=$(printf "%.0f" "$rate_5h")
  rc=$(rate_limit_color "$rate_int")
  rate_parts+="\033[38;5;240m5h:\033[0m \033[38;5;${rc}m${rate_int}%%\033[0m"
  if [ -n "$rate_5h_resets" ]; then
    dur=$(format_reset_duration "$rate_5h_resets")
    rate_parts+=" \033[38;5;245m⟳ ${dur}\033[0m"
  fi
fi
if [ -n "$rate_7d" ]; then
  rate_int=$(printf "%.0f" "$rate_7d")
  rc=$(rate_limit_color "$rate_int")
  if [ -n "$rate_parts" ]; then
    rate_parts+=" \033[38;5;240m|\033[0m "
  fi
  rate_parts+="\033[38;5;240m7d:\033[0m \033[38;5;${rc}m${rate_int}%%\033[0m"
  if [ -n "$rate_7d_resets" ]; then
    dur=$(format_reset_duration "$rate_7d_resets")
    rate_parts+=" \033[38;5;245m⟳ ${dur}\033[0m"
  fi
fi
if [ -n "$rate_parts" ]; then
  out+=" \033[38;5;240m│\033[0m ${rate_parts}"
fi

printf "$out"
