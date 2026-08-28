package main

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func generateReport(reportPath string, results []scriptResult, totalDuration time.Duration) error {
	reportDir := filepath.Dir(reportPath)
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return err
	}

	passed, failed := 0, 0
	for _, r := range results {
		if r.err != nil {
			failed++
		} else {
			passed++
		}
	}
	total := len(results)
	overall := "PASSED"
	overallColor := "#2b8a3e"
	overallBg := "#ebfbee"
	if failed > 0 {
		overall = "FAILED"
		overallColor = "#e03131"
		overallBg = "#fff5f5"
	}

	// Generate per-script detail pages and build table
	scriptsDir := filepath.Join(filepath.Dir(manifest), "scripts")
	var rows strings.Builder
	for _, r := range results {
		// e.g. script="/abs/test/scripts/subdir/test_abc.py" -> relPath="subdir/test_abc.py"
		relPath, _ := filepath.Rel(scriptsDir, r.script)
		if relPath == "" || strings.HasPrefix(relPath, "..") {
			relPath = filepath.Base(r.script)
		}
		detailRel := strings.TrimSuffix(relPath, ".py") + ".html"
		detailFullPath := filepath.Join(reportDir, detailRel)

		if err := generateDetailPage(detailFullPath, relPath, r); err != nil {
			return fmt.Errorf("generate detail page for %s: %w", relPath, err)
		}

		name := html.EscapeString(relPath)
		dur := r.duration.Round(time.Second).String()
		status := "PASS"
		statusColor := "#2b8a3e"
		errMsg := ""
		if r.err != nil {
			status = "FAIL"
			statusColor = "#e03131"
			errMsg = html.EscapeString(r.err.Error())
		}
		rows.WriteString(fmt.Sprintf(`
    <tr>
      <td style="padding:8px 12px;font-family:monospace;font-size:13px"><a href="%s" style="color:#1971c2;text-decoration:none">%s</a></td>
      <td style="padding:8px 12px;text-align:center">
        <span style="color:%s;font-weight:600">%s</span>
      </td>
      <td style="padding:8px 12px;text-align:center;font-family:monospace;font-size:13px">%s</td>
      <td style="padding:8px 12px;font-size:12px;color:#868e96">%s</td>
    </tr>`, detailRel, name, statusColor, status, dur, errMsg))
	}

	passRate := 0.0
	if total > 0 {
		passRate = float64(passed) / float64(total) * 100
	}

	failedColor := "#adb5bd"
	if failed > 0 {
		failedColor = "#e03131"
	}

	report := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Integration Test Report</title>
<style>
  * { margin:0; padding:0; box-sizing:border-box; }
  body { font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif; background:#f8f9fa; color:#212529; padding:24px; }
  .container { max-width:960px; margin:0 auto; }
  .header { background:%s; border:1px solid %s33; border-radius:12px; padding:20px 28px; margin-bottom:24px; display:flex; align-items:center; justify-content:space-between; }
  .header h1 { font-size:20px; color:#212529; }
  .badge { display:inline-block; padding:4px 14px; border-radius:20px; font-size:13px; font-weight:600; color:#fff; background:%s; }
  .cards { display:flex; gap:20px; margin-bottom:24px; }
  .card { flex:1; background:#fff; border:1px solid #dee2e6; border-radius:12px; padding:20px; text-align:center; }
  .card h3 { font-size:13px; color:#868e96; margin-bottom:8px; text-transform:uppercase; letter-spacing:0.5px; }
  .card .value { font-size:28px; font-weight:700; }
  .card .sub { font-size:12px; color:#adb5bd; margin-top:4px; }
  .section { background:#fff; border:1px solid #dee2e6; border-radius:12px; padding:20px 24px; margin-bottom:20px; }
  .section h2 { font-size:16px; margin-bottom:14px; color:#343a40; }
  table { width:100%%; border-collapse:collapse; }
  thead th { padding:8px 12px; text-align:left; font-size:12px; color:#868e96; text-transform:uppercase; letter-spacing:0.5px; border-bottom:2px solid #e9ecef; }
  tbody tr { border-bottom:1px solid #f1f3f5; }
  tbody tr:last-child { border-bottom:none; }
  .footer { text-align:center; color:#adb5bd; font-size:12px; margin-top:24px; }
</style>
</head>
<body>
<div class="container">
  <div class="header">
    <div>
      <h1>Integration Test Report</h1>
      <span style="font-size:13px;color:#868e96">%s &nbsp;|&nbsp; Duration: %s &nbsp;|&nbsp; <a href="coverage.html" style="color:#1971c2;text-decoration:none;font-weight:500">📊 API Coverage Report →</a></span>
    </div>
    <span class="badge">%s</span>
  </div>

  <div class="cards">
    <div class="card">
      <h3>Total</h3>
      <div class="value">%d</div>
      <div class="sub">scripts</div>
    </div>
    <div class="card">
      <h3>Passed</h3>
      <div class="value" style="color:#2b8a3e">%d</div>
      <div class="sub">%s</div>
    </div>
    <div class="card">
      <h3>Failed</h3>
      <div class="value" style="color:%s">%d</div>
      <div class="sub">&nbsp;</div>
    </div>
    <div class="card">
      <h3>Duration</h3>
      <div class="value" style="font-size:22px">%s</div>
      <div class="sub">total</div>
    </div>
  </div>

  <div class="section">
    <h2>Summary</h2>
    <table>
      <thead>
        <tr>
          <th>Script</th>
          <th style="text-align:center">Status</th>
          <th style="text-align:center">Duration</th>
          <th>Error</th>
        </tr>
      </thead>
      <tbody>
        %s
      </tbody>
    </table>
  </div>

  <div class="footer">Generated by hatchery integration test tool</div>
</div>
</body>
</html>`,
		overallBg, overallColor, overallColor,
		time.Now().Format("2006-01-02 15:04:05"), totalDuration.Round(time.Second), overall,
		total,
		passed, fmt.Sprintf("%.0f%%", passRate),
		failedColor, failed,
		totalDuration.Round(time.Second),
		rows.String(),
	)

	return os.WriteFile(reportPath, []byte(report), 0o644)
}

func generateDetailPage(detailFullPath, displayName string, r scriptResult) error {
	if err := os.MkdirAll(filepath.Dir(detailFullPath), 0o755); err != nil {
		return err
	}

	name := html.EscapeString(displayName)
	status := "PASS"
	statusColor := "#2b8a3e"
	statusBg := "#ebfbee"
	errSection := ""
	if r.err != nil {
		status = "FAIL"
		statusColor = "#e03131"
		statusBg = "#fff5f5"
		errSection = fmt.Sprintf(`
  <div style="background:#fff5f5;border:1px solid #e0313133;border-radius:8px;padding:14px;margin-bottom:20px;font-size:13px;color:#e03131">
    <strong>Error:</strong> %s
  </div>`, html.EscapeString(r.err.Error()))
	}

	outputText := html.EscapeString(r.output)
	if outputText == "" {
		outputText = "(no output)"
	}

	// Compute relative path back to index.html
	rel, _ := filepath.Rel(filepath.Dir(detailFullPath), filepath.Dir(detailFullPath))
	backLink := "index.html"
	if dir := filepath.Dir(displayName); dir != "." {
		// e.g. "subdir/test_abc.py" -> back link needs "../index.html"
		depth := strings.Count(dir, string(filepath.Separator)) + 1
		backLink = strings.Repeat("../", depth) + "index.html"
	}
	_ = rel

	page := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s - Integration Test</title>
<style>
  * { margin:0; padding:0; box-sizing:border-box; }
  body { font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif; background:#f8f9fa; color:#212529; padding:24px; }
  .container { max-width:960px; margin:0 auto; }
  .back { display:inline-block; margin-bottom:16px; color:#1971c2; text-decoration:none; font-size:13px; }
  .back:hover { text-decoration:underline; }
  .header { background:%s; border:1px solid %s33; border-radius:12px; padding:20px 28px; margin-bottom:24px; display:flex; align-items:center; justify-content:space-between; }
  .header h1 { font-size:18px; color:#212529; font-family:monospace; }
  .badge { display:inline-block; padding:4px 14px; border-radius:20px; font-size:13px; font-weight:600; color:#fff; background:%s; }
  .meta { font-size:13px; color:#868e96; margin-bottom:20px; }
  pre { background:#fff; border:1px solid #dee2e6; border-radius:8px; padding:16px; font-size:12px; line-height:1.7; overflow-x:auto; white-space:pre-wrap; word-wrap:break-word; }
</style>
</head>
<body>
<div class="container">
  <a class="back" href="%s">&larr; Back to summary</a>
  <div class="header">
    <h1>%s</h1>
    <span class="badge">%s</span>
  </div>
  <div class="meta">Duration: %s</div>
  %s
  <pre>%s</pre>
</div>
</body>
</html>`,
		name,
		statusBg, statusColor, statusColor,
		backLink,
		name, status,
		r.duration.Round(time.Second),
		errSection,
		outputText,
	)

	return os.WriteFile(detailFullPath, []byte(page), 0o644)
}
