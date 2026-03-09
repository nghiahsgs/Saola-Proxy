package proxy

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/nguyennghia/saola-proxy/internal/audit"
	"github.com/nguyennghia/saola-proxy/internal/sanitizer"
	"github.com/nguyennghia/saola-proxy/internal/scanner"
)

// dashboardData is the JSON payload served at /api/stats.
type dashboardData struct {
	Uptime          string            `json:"uptime"`
	StartTime       string            `json:"start_time"`
	Patterns        []patternInfo     `json:"patterns"`
	Mappings        map[string]string `json:"mappings"`
	Stats           map[string]int    `json:"stats"`
	TotalSanitized  int               `json:"total_sanitized"`
	TotalRehydrated int               `json:"total_rehydrated"`
}

type patternInfo struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

// dashboardHandler serves the web dashboard and stats API.
type dashboardHandler struct {
	registry  *scanner.PatternRegistry
	table     *sanitizer.MappingTable
	session   *audit.Session
	startTime time.Time
}

func newDashboardHandler(reg *scanner.PatternRegistry, table *sanitizer.MappingTable, session *audit.Session) *dashboardHandler {
	return &dashboardHandler{
		registry:  reg,
		table:     table,
		session:   session,
		startTime: time.Now(),
	}
}

func (d *dashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/stats":
		d.serveStats(w, r)
	default:
		d.servePage(w, r)
	}
}

func (d *dashboardHandler) serveStats(w http.ResponseWriter, _ *http.Request) {
	patterns := d.registry.GetAll()
	pInfos := make([]patternInfo, len(patterns))
	for i, p := range patterns {
		pInfos[i] = patternInfo{
			Name:        p.Name,
			Category:    p.Category,
			Description: p.Description,
			Enabled:     p.Enabled,
		}
	}

	summary := d.session.Summary()
	data := dashboardData{
		Uptime:          time.Since(d.startTime).Truncate(time.Second).String(),
		StartTime:       d.startTime.Format("2006-01-02 15:04:05"),
		Patterns:        pInfos,
		Mappings:        d.table.GetAll(),
		Stats:           summary.Detections,
		TotalSanitized:  summary.TotalSanitized,
		TotalRehydrated: summary.TotalRehydrated,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (d *dashboardHandler) servePage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Saola Proxy Dashboard</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0f172a; color: #e2e8f0; min-height: 100vh; }
  .container { max-width: 1100px; margin: 0 auto; padding: 24px; }
  header { display: flex; align-items: center; gap: 16px; margin-bottom: 32px; }
  header h1 { font-size: 24px; font-weight: 700; color: #f8fafc; }
  header .badge { background: #22c55e; color: #052e16; font-size: 12px; font-weight: 600; padding: 4px 10px; border-radius: 9999px; }
  .meta { font-size: 13px; color: #94a3b8; margin-left: auto; text-align: right; line-height: 1.5; }
  .cards { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; margin-bottom: 32px; }
  .card { background: #1e293b; border: 1px solid #334155; border-radius: 12px; padding: 20px; }
  .card .label { font-size: 13px; color: #94a3b8; margin-bottom: 4px; }
  .card .value { font-size: 32px; font-weight: 700; color: #f8fafc; }
  .card .value.green { color: #4ade80; }
  .card .value.blue { color: #60a5fa; }
  .card .value.amber { color: #fbbf24; }
  section { margin-bottom: 32px; }
  section h2 { font-size: 18px; font-weight: 600; margin-bottom: 12px; color: #f8fafc; }
  table { width: 100%; border-collapse: collapse; background: #1e293b; border-radius: 12px; overflow: hidden; border: 1px solid #334155; }
  th { text-align: left; padding: 10px 16px; background: #0f172a; font-size: 12px; text-transform: uppercase; color: #94a3b8; font-weight: 600; letter-spacing: 0.05em; }
  td { padding: 10px 16px; border-top: 1px solid #1e293b; font-size: 14px; }
  tr:hover td { background: #334155; }
  .badge-cat { display: inline-block; padding: 2px 8px; border-radius: 6px; font-size: 11px; font-weight: 600; }
  .badge-secret { background: #7f1d1d; color: #fca5a5; }
  .badge-pii { background: #78350f; color: #fde68a; }
  .badge-credential { background: #1e3a5f; color: #93c5fd; }
  .enabled { color: #4ade80; }
  .disabled { color: #ef4444; }
  .mono { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 13px; }
  .empty { text-align: center; color: #64748b; padding: 32px; font-size: 14px; }
  .bar { display: flex; align-items: center; gap: 8px; }
  .bar-fill { height: 6px; border-radius: 3px; background: #3b82f6; min-width: 4px; }
  .bar-label { font-size: 12px; color: #94a3b8; min-width: 24px; text-align: right; }
  .refresh-note { font-size: 11px; color: #475569; text-align: center; margin-top: 16px; }
  @media (max-width: 640px) { .cards { grid-template-columns: 1fr; } }
</style>
</head>
<body>
<div class="container">
  <header>
    <h1>Saola Proxy</h1>
    <span class="badge">RUNNING</span>
    <div class="meta">
      <div>Started: <span id="startTime">-</span></div>
      <div>Uptime: <span id="uptime">-</span></div>
    </div>
  </header>

  <div class="cards">
    <div class="card">
      <div class="label">Total Sanitized</div>
      <div class="value green" id="totalSanitized">0</div>
    </div>
    <div class="card">
      <div class="label">Total Rehydrated</div>
      <div class="value blue" id="totalRehydrated">0</div>
    </div>
    <div class="card">
      <div class="label">Active Mappings</div>
      <div class="value amber" id="totalMappings">0</div>
    </div>
  </div>

  <section>
    <h2>Detection Stats</h2>
    <div id="statsContainer"><div class="empty">No detections yet</div></div>
  </section>

  <section>
    <h2>Mapping Table</h2>
    <div id="mappingContainer"><div class="empty">No mappings yet</div></div>
  </section>

  <section>
    <h2>Patterns</h2>
    <table>
      <thead><tr><th>Name</th><th>Category</th><th>Description</th><th>Status</th></tr></thead>
      <tbody id="patternBody"></tbody>
    </table>
  </section>

  <div class="refresh-note">Auto-refreshes every 2 seconds</div>
</div>

<script>
function catBadge(cat) {
  const cls = cat === 'secret' ? 'badge-secret' : cat === 'pii' ? 'badge-pii' : 'badge-credential';
  return '<span class="badge-cat ' + cls + '">' + cat + '</span>';
}

function maskValue(v) {
  if (v.length <= 6) return '***';
  return v.slice(0, 3) + '*'.repeat(Math.min(v.length - 6, 20)) + v.slice(-3);
}

function refresh() {
  fetch('/api/stats').then(r => r.json()).then(d => {
    document.getElementById('startTime').textContent = d.start_time;
    document.getElementById('uptime').textContent = d.uptime;
    document.getElementById('totalSanitized').textContent = d.total_sanitized;
    document.getElementById('totalRehydrated').textContent = d.total_rehydrated;

    const mappings = d.mappings || {};
    const keys = Object.keys(mappings);
    document.getElementById('totalMappings').textContent = keys.length;

    // Detection stats
    const stats = d.stats || {};
    const statKeys = Object.keys(stats);
    const statsEl = document.getElementById('statsContainer');
    if (statKeys.length === 0) {
      statsEl.innerHTML = '<div class="empty">No detections yet</div>';
    } else {
      const maxVal = Math.max(...statKeys.map(k => stats[k]));
      let html = '<table><thead><tr><th>Pattern</th><th>Count</th><th></th></tr></thead><tbody>';
      statKeys.sort((a, b) => stats[b] - stats[a]);
      for (const k of statKeys) {
        const pct = maxVal > 0 ? (stats[k] / maxVal * 100) : 0;
        html += '<tr><td class="mono">' + k + '</td><td>' + stats[k] + '</td>';
        html += '<td><div class="bar"><div class="bar-fill" style="width:' + pct + '%"></div></div></td></tr>';
      }
      html += '</tbody></table>';
      statsEl.innerHTML = html;
    }

    // Mapping table
    const mapEl = document.getElementById('mappingContainer');
    if (keys.length === 0) {
      mapEl.innerHTML = '<div class="empty">No mappings yet</div>';
    } else {
      let html = '<table><thead><tr><th>Placeholder</th><th>Original (masked)</th></tr></thead><tbody>';
      keys.sort();
      for (const k of keys) {
        html += '<tr><td class="mono">' + k + '</td><td class="mono">' + maskValue(mappings[k]) + '</td></tr>';
      }
      html += '</tbody></table>';
      mapEl.innerHTML = html;
    }

    // Patterns
    const body = document.getElementById('patternBody');
    body.innerHTML = '';
    for (const p of (d.patterns || [])) {
      const tr = document.createElement('tr');
      tr.innerHTML = '<td class="mono">' + p.name + '</td><td>' + catBadge(p.category) + '</td><td>' + p.description + '</td><td class="' + (p.enabled ? 'enabled' : 'disabled') + '">' + (p.enabled ? 'Enabled' : 'Disabled') + '</td>';
      body.appendChild(tr);
    }
  }).catch(() => {});
}

refresh();
setInterval(refresh, 2000);
</script>
</body>
</html>`
