package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/KirillSachkov/gvardia/internal/adapters"
	"github.com/KirillSachkov/gvardia/internal/model"
	"github.com/KirillSachkov/gvardia/internal/runners"
	"github.com/KirillSachkov/gvardia/internal/runs"
)

// footerHeight is the number of lines reserved below the body: a status line and
// the keybind footer.
const footerHeight = 2
const workPaneHeaderLines = 2

type workTabSpec struct {
	tab   workTab
	key   string
	label string
}

var workTabs = []workTabSpec{
	{tab: tabAgents, key: "1", label: "agents"},
	{tab: tabTasks, key: "2", label: "tasks"},
	{tab: tabWorktrees, key: "3", label: "worktrees"},
	{tab: tabTools, key: "4", label: "tools"},
	{tab: tabHistory, key: "5", label: "history"},
}

var (
	borderActive   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	borderInactive = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	dim            = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	warn           = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	tabActive      = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62"))
	tabInactive    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

// geo holds the derived pane geometry for the current terminal size.
type geo struct {
	bodyH                              int
	leftOuterW, leftInnerW, leftInnerH int
	rightOuterW, rightInnerW           int
	sessOuterH, sessInnerH             int
	diffOuterH, diffInnerH             int
}

// geometry derives pane sizes from the terminal dimensions. It is the single
// source of layout truth, used by both layout() and View().
func (m Model) geometry() geo {
	bodyH := m.height - footerHeight
	if bodyH < 3 {
		bodyH = 3
	}
	leftW := m.width * 34 / 100
	if leftW < 28 {
		leftW = 28
	}
	if leftW > m.width-20 {
		leftW = m.width - 20
	}
	if !m.showProjects {
		leftW = 0
	}
	rightW := m.width - leftW
	sessH := bodyH / 2

	return geo{
		bodyH:       bodyH,
		leftOuterW:  leftW,
		leftInnerW:  max1(leftW - 2),
		leftInnerH:  max1(bodyH - 2),
		rightOuterW: max1(rightW),
		rightInnerW: max1(rightW - 2),
		sessOuterH:  max1(sessH),
		sessInnerH:  max1(sessH - 2),
		diffOuterH:  max1(bodyH - sessH),
		diffInnerH:  max1(bodyH - sessH - 2),
	}
}

// View renders the cockpit into a full-screen (alt-screen) view.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// render composes the three bordered panes plus a status line and footer.
func (m Model) render() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.loading && len(m.projects) == 0 {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center, "collecting fleet…")
	}

	g := m.geometry()
	sess := pane(m.level == levelWork, g.rightOuterW, g.sessOuterH, m.workPaneView())
	diff := pane(m.level == levelDetail || m.showActions, g.rightOuterW, g.diffOuterH, m.detailPaneView())
	right := lipgloss.JoinVertical(lipgloss.Left, sess, diff)
	body := right
	if m.showProjects {
		left := pane(m.level == levelProjects, g.leftOuterW, g.bodyH, m.projList.View())
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	return lipgloss.JoinVertical(lipgloss.Left, body, m.statusLine(), m.footer())
}

func (m Model) workPaneView() string {
	return m.tabsLine() + "\n" + m.workPaneTitle() + "\n" + m.sessions.View()
}

func (m Model) workPaneTitle() string {
	p := m.selectedProject()
	runsCount, reviewCount := 0, 0
	if p != nil {
		for _, r := range m.runsByProject[p.Path] {
			runsCount++
			if r.Status == runs.StatusReview {
				reviewCount++
			}
		}
	}
	active := m.activeTabLabel()
	return dim.Render(fmt.Sprintf(" %s · runs %d (%d review) · sessions %d · worktrees %d ",
		strings.ToUpper(active), runsCount, reviewCount, len(m.sessionList), len(m.worktreeList)))
}

func (m Model) tabsLine() string {
	parts := make([]string, 0, len(workTabs))
	for _, spec := range workTabs {
		label := fmt.Sprintf(" %s %s ", spec.key, spec.label)
		if spec.tab == m.activeTab {
			parts = append(parts, tabActive.Render(label))
			continue
		}
		parts = append(parts, tabInactive.Render(label))
	}
	return strings.Join(parts, dim.Render(" | "))
}

func (m Model) activeTabLabel() string {
	for _, spec := range workTabs {
		if spec.tab == m.activeTab {
			return spec.label
		}
	}
	return "agents"
}

func (m Model) detailPaneView() string {
	if m.showActions {
		return m.actionsHelp()
	}
	return m.diff.View()
}

// pane wraps content in a fixed-size rounded border, brighter when focused.
func pane(focused bool, width, height int, content string) string {
	width = max1(width)
	height = max1(height)
	innerW := max1(width - 2)
	innerH := max1(height - 2)
	content = fitBlock(content, innerW, innerH)
	inner := lipgloss.Place(innerW, innerH, lipgloss.Left, lipgloss.Top, content)
	if focused {
		return borderActive.Render(inner)
	}
	return borderInactive.Render(inner)
}

func fitBlock(content string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i := range lines {
		lines[i] = truncate(lines[i], width)
	}
	return strings.Join(lines, "\n")
}

// statusLine shows a modal prompt, the filter, an adapter banner, or a summary.
func (m Model) statusLine() string {
	switch {
	case m.confirm != nil:
		return warn.Render(truncate(m.confirm.message+" (y/n)", m.width))
	case m.launch != nil:
		return dim.Render(truncate(m.launchStatus(), m.width))
	case m.prompt != nil:
		return dim.Render("new "+m.prompt.harness+" agent: ") + m.prompt.input.View()
	case m.pathPrompt != nil:
		label := "add project: "
		if m.pathPrompt.mode == pathCreate {
			label = "create project: "
		}
		return dim.Render(label) + m.pathPrompt.input.View()
	case m.filtering:
		return dim.Render("filter: ") + m.filter.Value() + dim.Render("▏")
	case m.toast != "":
		return dim.Render(truncate("✓ "+m.toast, m.width))
	case m.banner != "":
		return warn.Render(truncate("⚠ "+m.banner, m.width))
	default:
		agents := 0
		activeRuns := 0
		reviewRuns := 0
		for _, p := range m.projects {
			agents += p.LiveAgents
			for _, r := range m.runsByProject[p.Path] {
				if r.Status == runs.StatusRunning {
					activeRuns++
				}
				if r.Status == runs.StatusReview {
					reviewRuns++
				}
			}
		}
		line := fmt.Sprintf("%d projects · %d active runs · %d review · %d live agents", len(m.projects), activeRuns, reviewRuns, agents)
		if !m.curated {
			line += " · A to curate"
		}
		return dim.Render(truncate(line, m.width))
	}
}

// footer renders the keybind hints for the current mode.
func (m Model) footer() string {
	keys := "1 agents · 2 tasks · 3 worktrees · 4 tools · 5 history · tab/shift+tab pane · enter actions · / filter · q"
	switch {
	case m.confirm != nil:
		keys = "y confirm · n cancel"
	case m.launch != nil:
		keys = "j/k task · tab runner · enter launch · esc cancel"
	case m.prompt != nil:
		keys = "tab harness · enter create · esc cancel"
	case m.pathPrompt != nil:
		keys = "enter confirm · esc cancel"
	case m.filtering:
		keys = "type to filter · enter apply · esc cancel"
	case m.showActions:
		keys = "↑↓ choose · enter run · 1..9 run · esc close · ctrl+c quit"
	default:
		keys = m.contextFooter()
	}
	return dim.Render(truncate(keys, m.width))
}

func (m Model) contextFooter() string {
	base := "1 agents · 2 tasks · 3 worktrees · 4 tools · 5 history"
	switch m.activeTab {
	case tabTasks:
		return base + " · tab/shift+tab pane · p projects · enter actions · s scope · n launch · / filter"
	case tabWorktrees:
		return base + " · tab/shift+tab pane · p projects · enter actions · d diff · g gc"
	case tabTools:
		return base + " · tab/shift+tab pane · p projects · enter actions · / filter · R refresh"
	case tabHistory:
		return base + " · tab/shift+tab pane · p projects · enter actions · d diff · a attach"
	default:
		return base + " · tab/shift+tab pane · p projects · enter actions · a attach · d diff · o report · n launch"
	}
}

func (m Model) actionsHelp() string {
	if m.actionMenu != nil {
		return m.actionMenuView()
	}
	var b strings.Builder
	b.WriteString("Actions\n\n")
	b.WriteString("1..5  switch tabs\n")
	b.WriteString("tab    cycle panes\n")
	b.WriteString("←/→    move focus between panes\n")
	b.WriteString("enter  open selected pane/item, esc goes back\n")
	b.WriteString("p      show/hide projects drawer\n")
	b.WriteString("/      filter current context\n")
	b.WriteString("R      refresh\n")
	b.WriteString("n      launch selected project task with a runner profile\n")
	switch m.activeTab {
	case tabTasks:
		b.WriteString("s      toggle all tasks / selected project scope\n")
		b.WriteString("n      launch a run from this project\n")
	case tabWorktrees:
		b.WriteString("d      open lazygit/git diff for selected worktree\n")
		b.WriteString("g      confirm worktree cleanup\n")
	case tabTools:
		b.WriteString("R      re-detect installed agent tools\n")
	case tabHistory:
		b.WriteString("a      attach when the selected historical row still has a target\n")
		b.WriteString("d      open diff for the selected session worktree\n")
	default:
		b.WriteString("a      attach selected run/session\n")
		b.WriteString("o      open report.md for selected run\n")
		b.WriteString("k      confirm kill selected run/session\n")
		b.WriteString("r      copy resume command for selected live session\n")
	}
	b.WriteString("\nProject curation: A add existing repo · C create repo · X untrack\n")
	return b.String()
}

func (m Model) actionMenuView() string {
	var b strings.Builder
	b.WriteString("Actions")
	if m.actionMenu.title != "" {
		b.WriteString(" - " + m.actionMenu.title)
	}
	b.WriteString("\n\n")
	for i, item := range m.actionMenu.items {
		cursor := "  "
		if i == m.actionMenu.cursor {
			cursor = "› "
		}
		label := fmt.Sprintf("%d %s", i+1, item.label)
		b.WriteString(cursor + label)
		if item.hint != "" {
			b.WriteString("\n  " + dim.Render(item.hint))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n" + dim.Render("enter run selected · ↑↓ choose · number run · esc close"))
	return strings.TrimRight(b.String(), "\n")
}

func (m Model) launchStatus() string {
	task := "(no task)"
	if m.launch != nil && len(m.launch.tasks) > 0 {
		task = m.launch.tasks[m.launch.taskIdx].Title
	}
	runner := "(no runner)"
	if m.launch != nil && len(m.profiles) > 0 {
		runner = m.profiles[m.launch.profileIdx].Name
	}
	return fmt.Sprintf("launch run · task: %s · runner: %s", task, runner)
}

// detailHeader renders the selected session's summary and a metadata line for
// the top of the detail pane.
func detailHeader(s model.Session) string {
	summary := s.Summary
	if summary == "" {
		summary = "(no summary)"
	}
	branch := s.Branch
	if branch == "" {
		branch = "(detached)"
	}
	task := ""
	if s.Task != "" {
		task = " · " + s.Task
	}
	state := "ended"
	if s.Live {
		state = string(s.Status)
	}
	meta := fmt.Sprintf("%s %s%s · %s · %d files +%d -%d · %s",
		state, s.Harness, task, branch,
		s.ChangeStat.Files, s.ChangeStat.Added, s.ChangeStat.Removed, relativeTime(s.LastActivity))
	return summary + "\n" + dim.Render(meta)
}

// worktreeHeader renders the selected worktree's path and a metadata line for
// the top of the detail pane in the worktree view.
func worktreeHeader(w model.Worktree) string {
	branch := w.Branch
	if branch == "" {
		branch = "(detached)"
	}
	kind := "linked"
	if w.IsPrimary {
		kind = "primary"
	}
	state := "clean"
	if w.Dirty {
		state = "dirty"
	}
	agent := "no agent"
	if len(w.Sessions) > 0 {
		agent = w.Sessions[0].Harness
		if len(w.Sessions) > 1 {
			agent = fmt.Sprintf("%s +%d", w.Sessions[0].Harness, len(w.Sessions)-1)
		}
	}
	meta := fmt.Sprintf("%s %s · %s · ↑%d↓%d · %d files +%d -%d · %s · %s",
		kind, branch, state, w.Ahead, w.Behind,
		w.ChangeStat.Files, w.ChangeStat.Added, w.ChangeStat.Removed, agent, relativeTime(w.LastCommit))
	return w.Path + "\n" + dim.Render(meta)
}

// sessionDetail is the full detail body for a session: header plus its report
// and artifacts.
func sessionDetail(s model.Session) string {
	return detailHeader(s) + sessionExtra(s)
}

func runDetail(r runs.Run) string {
	title := r.TaskTitle
	if title == "" {
		title = "Manual objective"
	}
	var b strings.Builder
	b.WriteString("Objective\n")
	b.WriteString(title + "\n")
	b.WriteString(dim.Render(fmt.Sprintf("State: %s · Agent: %s (%s) · Updated %s",
		readableRunStatus(r.Status), valueOr(r.Runner, "unknown"), valueOr(r.Tool, "unknown"), valueOr(relativeTime(r.UpdatedAt), "unknown"))))
	if r.Telemetry.Phase != "" || r.Telemetry.Summary != "" {
		phase := valueOr(r.Telemetry.Phase, "current")
		summary := valueOr(r.Telemetry.Summary, "No summary yet")
		b.WriteString("\n" + dim.Render("Now: ") + phase + " - " + summary)
	}
	if r.Branch != "" {
		b.WriteString("\n" + dim.Render("Branch: ") + r.Branch)
	}
	if r.TmuxTarget != "" {
		b.WriteString("\n" + dim.Render("Terminal: ") + r.TmuxTarget)
	}
	if r.WorktreePath != "" {
		b.WriteString("\n" + dim.Render("Worktree: ") + r.WorktreePath)
	}

	if r.Report != "" {
		b.WriteString("\n\nReport summary\n" + reportSummary(r.Report))
	} else {
		b.WriteString("\n\nReport summary\n" + dim.Render("No final report yet. Agents should write report.md or run gvardia run report."))
	}

	b.WriteString("\n\nEvidence\n")
	b.WriteString(runEvidenceBlock(r))

	b.WriteString("\n\nActivity\n")
	b.WriteString(runEventsBlock(r.Events))
	return b.String()
}

func reportSummary(report string) string {
	report = strings.TrimSpace(report)
	if report == "" {
		return dim.Render("No final report yet.")
	}
	lines := strings.Split(report, "\n")
	if section := markdownSection(lines, "tl;dr", "summary", "резюме", "итог"); len(section) > 0 {
		return strings.Join(limitNonEmpty(section, 5), "\n")
	}
	return strings.Join(limitNonEmpty(skipMarkdownTitle(lines), 6), "\n")
}

func markdownSection(lines []string, names ...string) []string {
	for i, line := range lines {
		title, ok := markdownHeading(line)
		if !ok || !matchesAny(title, names...) {
			continue
		}
		var out []string
		for _, next := range lines[i+1:] {
			if _, isHeading := markdownHeading(next); isHeading {
				break
			}
			out = append(out, next)
		}
		return out
	}
	return nil
}

func markdownHeading(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "#") {
		return "", false
	}
	line = strings.TrimLeft(line, "#")
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}
	return strings.ToLower(line), true
}

func matchesAny(value string, names ...string) bool {
	for _, name := range names {
		if strings.Contains(value, name) {
			return true
		}
	}
	return false
}

func skipMarkdownTitle(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	if _, ok := markdownHeading(lines[0]); ok {
		return lines[1:]
	}
	return lines
}

func limitNonEmpty(lines []string, limit int) []string {
	out := make([]string, 0, limit)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
		if len(out) >= limit {
			break
		}
	}
	if len(out) == 0 {
		return []string{dim.Render("Report is present but has no summary text.")}
	}
	return out
}

func readableRunStatus(status runs.Status) string {
	switch status {
	case runs.StatusRunning:
		return "Running"
	case runs.StatusReview:
		return "Needs review"
	case runs.StatusDone:
		return "Done"
	case runs.StatusFailed:
		return "Failed"
	case runs.StatusKilled:
		return "Killed"
	case runs.StatusPending:
		return "Queued"
	default:
		return "Unknown"
	}
}

func runEventsBlock(events []runs.Event) string {
	if len(events) == 0 {
		return dim.Render("No activity events yet.")
	}
	const cap = 6
	start := 0
	if len(events) > cap {
		start = len(events) - cap
	}
	var b strings.Builder
	for _, event := range events[start:] {
		kind := valueOr(event.Type, "event")
		msg := valueOr(event.Message, "(empty)")
		b.WriteString(fmt.Sprintf("- %s: %s\n", kind, msg))
	}
	return strings.TrimRight(b.String(), "\n")
}

func runEvidenceBlock(r runs.Run) string {
	var b strings.Builder
	if r.Report != "" {
		b.WriteString("- report: report.md\n")
	} else {
		b.WriteString("- report: not written yet\n")
	}
	for _, artifact := range r.RunArtifacts {
		if artifact.Type == "report" && artifact.Path == "report.md" {
			continue
		}
		b.WriteString(fmt.Sprintf("- %s: %s (%s)\n", valueOr(artifact.Type, "artifact"), valueOr(artifact.Title, artifact.Path), artifact.Path))
	}
	if len(r.Artifacts) > 0 {
		b.WriteString(fmt.Sprintf("- diff: %d files changed", len(r.Artifacts)))
		shown := len(r.Artifacts)
		if shown > 5 {
			shown = 5
		}
		for i := 0; i < shown; i++ {
			b.WriteString(fmt.Sprintf("\n  %s %s", r.Artifacts[i].Status, r.Artifacts[i].Path))
		}
		if len(r.Artifacts) > shown {
			b.WriteString(fmt.Sprintf("\n  ...and %d more", len(r.Artifacts)-shown))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func taskDetail(t model.Task) string {
	title := t.Title
	if title == "" {
		title = "(untitled task)"
	}
	meta := fmt.Sprintf("%s · %s · %s", valueOr(t.Status, "inbox"), valueOr(t.Project, "no project"), valueOr(t.ID, "no id"))
	var b strings.Builder
	b.WriteString(title + "\n" + dim.Render(meta))
	if t.Source != "" {
		b.WriteString("\n" + dim.Render("source ") + t.Source)
	}
	if t.Path != "" {
		b.WriteString("\n" + dim.Render("path ") + t.Path)
	}
	if t.Body != "" {
		b.WriteString("\n\n" + t.Body)
	}
	return b.String()
}

func toolDetail(tool runners.Tool) string {
	state := "missing"
	if tool.Installed {
		state = "installed"
	}
	kind := "custom"
	if tool.BuiltIn {
		kind = "built-in"
	}
	var b strings.Builder
	b.WriteString(tool.Name + "\n" + dim.Render(fmt.Sprintf("%s · %s · command %s", state, kind, valueOr(tool.Command, tool.Name))))
	if tool.Path != "" {
		b.WriteString("\n" + dim.Render("path ") + tool.Path)
	}
	b.WriteString("\n\n")
	if tool.Installed {
		b.WriteString("Ready for runner profiles that use this tool.")
	} else {
		b.WriteString("Install this CLI or override it with a custom tool in config.")
	}
	return b.String()
}

func valueOr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// sessionExtra is the report + artifacts block appended below a detail header.
func sessionExtra(s model.Session) string {
	var b strings.Builder
	if s.Report != "" {
		b.WriteString("\n\n" + dim.Render("report") + "\n" + s.Report)
	}
	if len(s.Artifacts) > 0 {
		b.WriteString("\n\n" + artifactsBlock(s.Artifacts))
	}
	return b.String()
}

// artifactsBlock renders a session's artifacts (changed files + report files),
// capped so a large diff can't flood the detail pane.
func artifactsBlock(arts []model.Artifact) string {
	const cap = 20
	var b strings.Builder
	b.WriteString(dim.Render(fmt.Sprintf("artifacts (%d)", len(arts))))
	shown := len(arts)
	if shown > cap {
		shown = cap
	}
	for i := 0; i < shown; i++ {
		b.WriteString(fmt.Sprintf("\n  %-6s %s", arts[i].Status, arts[i].Path))
	}
	if len(arts) > cap {
		b.WriteString(fmt.Sprintf("\n  …and %d more", len(arts)-cap))
	}
	return b.String()
}

// failureBanner summarizes skipped adapters for the status line.
func failureBanner(failures []adapters.Failure) string {
	if len(failures) == 0 {
		return ""
	}
	names := make([]string, 0, len(failures))
	for _, f := range failures {
		names = append(names, f.Adapter)
	}
	return "adapters skipped: " + strings.Join(names, ", ")
}

// truncate clips s to at most width display cells, appending an ellipsis.
func truncate(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "…")
}
