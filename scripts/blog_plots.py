"""Generate plots for the llmprobe benchmark blog post.

Uses the 7-day continuous monitoring dataset (60s intervals, 6 models via OpenRouter).
"""
import json
import statistics
from collections import defaultdict
from datetime import datetime
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np

plt.rcParams.update({
    "figure.facecolor": "#0d1117",
    "axes.facecolor": "#0d1117",
    "axes.edgecolor": "#30363d",
    "axes.labelcolor": "#c9d1d9",
    "text.color": "#c9d1d9",
    "xtick.color": "#8b949e",
    "ytick.color": "#8b949e",
    "grid.color": "#21262d",
    "legend.facecolor": "#161b22",
    "legend.edgecolor": "#30363d",
    "font.family": "sans-serif",
    "font.size": 11,
})

with open("benchmark-7d.jsonl") as f:
    raw = [json.loads(l) for l in f if l.strip()]
data = [d for d in raw if d["status"] != "error"]

short = {
    "openai/gpt-4o-mini": "GPT-4o-mini",
    "anthropic/claude-3.5-haiku": "Claude 3.5 Haiku",
    "google/gemini-2.0-flash-001": "Gemini 2.0 Flash",
    "meta-llama/llama-3.3-70b-instruct": "Llama 3.3 70B",
    "deepseek/deepseek-chat": "DeepSeek Chat",
    "mistralai/mistral-small-2603": "Mistral Small",
}

colors = {
    "GPT-4o-mini": "#10a37f",
    "Claude 3.5 Haiku": "#d4a574",
    "Gemini 2.0 Flash": "#4285f4",
    "Llama 3.3 70B": "#7c3aed",
    "DeepSeek Chat": "#0ea5e9",
    "Mistral Small": "#f97316",
}

models = sorted(set(d["model"] for d in data))

# --- Plot 1: TTFT comparison bar chart ---
fig, ax = plt.subplots(figsize=(12, 6))

names = []
p50s = []
p95s = []
cols = []
for m in models:
    ttft = sorted(d["ttft_ms"] for d in data if d["model"] == m)
    name = short.get(m, m)
    names.append(name)
    p50s.append(statistics.median(ttft))
    p95s.append(ttft[int(len(ttft) * 0.95)])
    cols.append(colors.get(name, "#888"))

x = np.arange(len(names))
w = 0.35
bars1 = ax.bar(x - w/2, p50s, w, label="p50", color=cols, alpha=0.9)
bars2 = ax.bar(x + w/2, p95s, w, label="p95", color=cols, alpha=0.45)

ax.set_ylabel("TTFT (ms)")
ax.set_title("Time to First Token: p50 vs p95 (7 days, ~10k probes per model)", fontsize=14, fontweight="bold", pad=15)
ax.set_xticks(x)
ax.set_xticklabels(names, rotation=20, ha="right", fontsize=10)
ax.legend(fontsize=10)
ax.grid(axis="y", alpha=0.3)

for bar, val in zip(bars1, p50s):
    ax.text(bar.get_x() + bar.get_width()/2, bar.get_height() + 30,
            f"{int(val)}ms", ha="center", va="bottom", fontsize=9, color="#c9d1d9")

plt.tight_layout()
plt.savefig("demo/blog_ttft_comparison.png", dpi=150)
print("Saved demo/blog_ttft_comparison.png")

# --- Plot 2: Throughput comparison ---
fig2, ax2 = plt.subplots(figsize=(12, 6))

tps_vals = []
tps_names = []
tps_cols = []
for m in models:
    tps = [d["tokens_per_sec"] for d in data if d["model"] == m and d["tokens_per_sec"] > 0]
    name = short.get(m, m)
    tps_names.append(name)
    tps_vals.append(statistics.median(tps) if tps else 0)
    tps_cols.append(colors.get(name, "#888"))

bars = ax2.barh(tps_names, tps_vals, color=tps_cols, alpha=0.9, height=0.6)
ax2.set_xlabel("Tokens per second (median)")
ax2.set_title("Generation Throughput by Model (7-day median)", fontsize=14, fontweight="bold", pad=15)
ax2.grid(axis="x", alpha=0.3)
ax2.invert_yaxis()

for bar, val in zip(bars, tps_vals):
    ax2.text(bar.get_width() + 2, bar.get_y() + bar.get_height()/2,
             f"{val:.1f}", ha="left", va="center", fontsize=10, color="#c9d1d9")

plt.tight_layout()
plt.savefig("demo/blog_throughput.png", dpi=150)
print("Saved demo/blog_throughput.png")

# --- Plot 3: TTFT distribution box plot ---
fig3, ax3 = plt.subplots(figsize=(12, 6))

all_ttft = []
positions = []
vlabels = []
vcolors = []
for i, m in enumerate(models):
    ttft = [d["ttft_ms"] for d in data if d["model"] == m]
    all_ttft.append(ttft)
    positions.append(i)
    name = short.get(m, m)
    vlabels.append(name)
    vcolors.append(colors.get(name, "#888"))

bp = ax3.boxplot(all_ttft, positions=positions, vert=True, patch_artist=True,
                 widths=0.5, showfliers=False,
                 medianprops=dict(color="white", linewidth=2))

for patch, col in zip(bp["boxes"], vcolors):
    patch.set_facecolor(col)
    patch.set_alpha(0.7)

ax3.set_xticks(positions)
ax3.set_xticklabels(vlabels, rotation=20, ha="right", fontsize=10)
ax3.set_ylabel("TTFT (ms)")
ax3.set_title("TTFT Distribution Over 7 Days (~10k probes per model)", fontsize=14, fontweight="bold", pad=15)
ax3.grid(axis="y", alpha=0.3)

plt.tight_layout()
plt.savefig("demo/blog_ttft_distribution.png", dpi=150)
print("Saved demo/blog_ttft_distribution.png")

# --- Plot 4: Latency breakdown ---
fig4, ax4 = plt.subplots(figsize=(12, 6))

ttft_meds = []
gen_meds = []
names4 = []
cols4 = []
for m in models:
    pts = [d for d in data if d["model"] == m]
    ttft = statistics.median([d["ttft_ms"] for d in pts])
    lat = statistics.median([d["latency_ms"] for d in pts])
    name = short.get(m, m)
    names4.append(name)
    ttft_meds.append(ttft)
    gen_meds.append(lat - ttft)
    cols4.append(colors.get(name, "#888"))

x4 = np.arange(len(names4))
ax4.bar(x4, ttft_meds, 0.6, label="TTFT (time to first token)", color=cols4, alpha=0.9)
ax4.bar(x4, gen_meds, 0.6, bottom=ttft_meds, label="Generation time", color=cols4, alpha=0.35)

ax4.set_ylabel("Total latency (ms)")
ax4.set_title("Latency Breakdown: TTFT vs Generation", fontsize=14, fontweight="bold", pad=15)
ax4.set_xticks(x4)
ax4.set_xticklabels(names4, rotation=20, ha="right", fontsize=10)
ax4.legend(fontsize=10)
ax4.grid(axis="y", alpha=0.3)

plt.tight_layout()
plt.savefig("demo/blog_latency_breakdown.png", dpi=150)
print("Saved demo/blog_latency_breakdown.png")

# --- Plot 5: TTFT timeline (hourly rolling median) ---
fig5, ax5 = plt.subplots(figsize=(16, 6))

for m in models:
    pts = [(datetime.strptime(d["timestamp"], "%Y-%m-%dT%H:%M:%SZ"), d["ttft_ms"])
           for d in data if d["model"] == m]
    pts.sort()

    window = 60
    times = [p[0] for p in pts]
    vals = [p[1] for p in pts]
    smoothed = []
    smooth_times = []
    for i in range(0, len(vals), window):
        chunk = vals[i:i+window]
        if chunk:
            smoothed.append(statistics.median(chunk))
            smooth_times.append(times[i + len(chunk)//2])

    name = short.get(m, m)
    ax5.plot(smooth_times, smoothed, label=name, color=colors.get(name, "#888"),
             linewidth=1.2, alpha=0.85)

ax5.set_xlabel("Date")
ax5.set_ylabel("TTFT (ms), 1-hour rolling median")
ax5.set_title("TTFT Over 7 Days (hourly median, 6 models via OpenRouter)", fontsize=14, fontweight="bold", pad=15)
ax5.legend(loc="upper right", fontsize=9)
ax5.set_ylim(0, 6000)
ax5.grid(axis="y", alpha=0.3)
fig5.autofmt_xdate()
plt.tight_layout()
plt.savefig("demo/blog_ttft_timeline.png", dpi=150)
print("Saved demo/blog_ttft_timeline.png")
