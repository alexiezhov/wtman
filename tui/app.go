package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/hibobio/wtman/core"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type mode int

const (
	modeBranchList mode = iota
	modeRepoSelect
	modeBranchNamePrompt
	modeDeleteConfirm
	modeRenamePrompt
	modeSourceDirPrompt
	modeTargetDirPrompt
	modeDirtyConfirm
	modeSpinner
	modeHelp
)

type AppModel struct {
	cfg        core.Config
	cfgPath    string
	mode       mode
	prevMode   mode
	width      int
	height     int
	branchList BranchListModel
	repoSelect RepoSelectModel
	statusBar  StatusBarModel
	prompt     PromptModel
	spinner    spinner.Model
	spinnerMsg string
	errMsg     string
	watcher    *core.DirWatcher

	// state for multi-step flows
	pendingRepos  []core.RepoEntry
	pendingBranch string
	isNewFlow     bool
}

func NewApp(cfg core.Config, cfgPath string) AppModel {
	ApplyColors(cfg.Colors)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styleSpinner

	w := core.NewDirWatcher(cfg.SourceDir, cfg.TargetDir, 2*time.Second)

	return AppModel{
		cfg:        cfg,
		cfgPath:    cfgPath,
		mode:       modeBranchList,
		branchList: NewBranchList(),
		repoSelect: NewRepoSelect(),
		statusBar:  NewStatusBar(),
		prompt:     NewPrompt(),
		spinner:    s,
		watcher:    w,
	}
}

func (m AppModel) Init() tea.Cmd {
	m.watcher.Start()
	return tea.Batch(
		m.loadBranches,
		waitForWatchEvent(m.watcher.Events()),
	)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.branchList = m.branchList.SetSize(msg.Width, msg.Height)
		m.repoSelect = m.repoSelect.SetSize(msg.Width, msg.Height)
		m.statusBar = m.statusBar.SetWidth(msg.Width)
		m.prompt = m.prompt.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyCtrlD {
			m.watcher.Stop()
			return m, tea.Quit
		}
		// Block input during spinner
		if m.mode == modeSpinner {
			return m, nil
		}
		return m.handleKey(msg)

	case spinner.TickMsg:
		if m.mode == modeSpinner {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case branchesLoadedMsg:
		m.branchList = m.branchList.SetBranches(msg.branches)
		if msg.err != nil {
			return m, m.setError(msg.err.Error())
		}
		return m, nil

	case reposLoadedMsg:
		m.repoSelect = m.repoSelect.SetRepos(msg.repos)
		if msg.err != nil {
			return m, m.setError(msg.err.Error())
		}
		return m, nil

	case clearErrorMsg:
		m.errMsg = ""
		return m, nil

	case OperationDoneMsg:
		m.mode = modeBranchList
		if msg.Err != nil {
			return m, tea.Batch(m.loadBranches, m.setError(msg.Err.Error()))
		}
		m.errMsg = ""
		return m, m.loadBranches

	case CommandMsg:
		return m.handleCommand(msg)

	case BranchSelectedMsg:
		return m.enterUpdateMode(msg.Branch)

	case ReposConfirmedMsg:
		m.pendingRepos = msg.Repos
		if m.isNewFlow {
			m.mode = modeBranchNamePrompt
			m.prompt = m.prompt.ActivateText("Branch name:")
			return m, nil
		}
		return m.checkDirtyBeforeUpdate()

	case DirtyWorktreesMsg:
		if len(msg.DirtyRepos) == 0 {
			return m.runUpdate(false)
		}
		m.mode = modeDirtyConfirm
		label := fmt.Sprintf("Dirty worktrees: %s. Force remove? (uncommitted changes will be lost)", strings.Join(msg.DirtyRepos, ", "))
		m.prompt = m.prompt.ActivateConfirm(label)
		return m, nil

	case ReposCancelledMsg:
		m.mode = modeBranchList
		return m, nil

	case PromptResultMsg:
		return m.handlePromptResult(msg)

	case ConfirmResultMsg:
		return m.handleConfirmResult(msg)

	case WatchEventMsg:
		var cmd tea.Cmd
		switch msg.Event.Kind {
		case core.SourceChanged:
			if m.mode == modeRepoSelect {
				cmd = m.loadRepos
			}
		case core.TargetChanged:
			cmd = m.loadBranches
		}
		return m, tea.Batch(cmd, waitForWatchEvent(m.watcher.Events()))
	}

	return m, nil
}

func (m AppModel) View() string {
	var b strings.Builder

	// Title
	title := styleTitle.Render("  WTMAN - worktree manager")
	if m.mode == modeRepoSelect || m.mode == modeBranchNamePrompt {
		suffix := "new feature branch"
		if !m.isNewFlow {
			suffix = "update " + m.pendingBranch
		}
		title = styleTitle.Render("  WTMAN - worktree manager") + styleHeader.Render(" \u2500\u2500 "+suffix)
	}
	b.WriteString(title + "\n\n")

	// Main content
	switch m.mode {
	case modeBranchList, modeDeleteConfirm, modeDirtyConfirm, modeRenamePrompt, modeSourceDirPrompt, modeTargetDirPrompt:
		b.WriteString(m.branchList.View())
	case modeRepoSelect:
		b.WriteString(m.repoSelect.View())
	case modeBranchNamePrompt:
		names := repoNames(m.pendingRepos)
		b.WriteString("  Selected: " + strings.Join(names, ", ") + "\n\n")
		b.WriteString(m.prompt.View())
		return b.String()
	case modeSpinner:
		b.WriteString(m.branchList.View())
	case modeHelp:
		b.WriteString(m.helpView())
		return b.String()
	}

	// Error
	if m.errMsg != "" {
		b.WriteString("\n" + styleError.Render("  Error: "+m.errMsg) + "\n")
	}

	// Bottom area: prompts, status bar, or hints
	b.WriteString("\n")
	switch m.mode {
	case modeSpinner:
		b.WriteString("  " + m.spinner.View() + " " + m.spinnerMsg + "\n")
	case modeDeleteConfirm, modeDirtyConfirm, modeRenamePrompt, modeSourceDirPrompt, modeTargetDirPrompt:
		b.WriteString(m.prompt.View() + "\n")
	case modeRepoSelect:
		if fv := m.repoSelect.FilterView(); fv != "" {
			b.WriteString(fv + "\n")
		}
		b.WriteString(m.repoSelect.HintView() + "\n")
	case modeBranchList:
		if m.statusBar.IsActive() {
			b.WriteString(m.statusBar.View() + "\n")
		} else {
			b.WriteString(m.branchList.HintView() + "\n")
		}
	}

	return b.String()
}

// --- Key handling ---

func (m AppModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Status bar intercepts when active
	if m.statusBar.IsActive() {
		var cmd tea.Cmd
		m.statusBar, cmd = m.statusBar.Update(msg)
		return m, cmd
	}

	// Prompt intercepts when active
	if m.prompt.IsActive() {
		var cmd tea.Cmd
		m.prompt, cmd = m.prompt.Update(msg)
		return m, cmd
	}

	switch m.mode {
	case modeBranchList:
		if msg.Type == tea.KeyRunes && string(msg.Runes) == "/" {
			m.statusBar = m.statusBar.Activate()
			return m, nil
		}
		if msg.Type == tea.KeyRunes && string(msg.Runes) == "q" {
			m.watcher.Stop()
			return m, tea.Quit
		}
		var cmd tea.Cmd
		m.branchList, cmd = m.branchList.Update(msg)
		return m, cmd

	case modeHelp:
		if msg.Type == tea.KeyEsc || msg.Type == tea.KeyEnter ||
			(msg.Type == tea.KeyRunes && (string(msg.Runes) == "q" || string(msg.Runes) == "?")) {
			m.mode = modeBranchList
		}
		return m, nil

	case modeRepoSelect:
		var cmd tea.Cmd
		m.repoSelect, cmd = m.repoSelect.Update(msg)
		return m, cmd
	}

	return m, nil
}

// --- Command handling ---

func (m AppModel) handleCommand(cmd CommandMsg) (tea.Model, tea.Cmd) {
	switch cmd.Name {
	case "/new":
		return m.enterNewMode()
	case "/delete":
		return m.enterDeleteMode()
	case "/rename":
		return m.enterRenameMode()
	case "/pull":
		return m.runPull()
	case "/open":
		return m.runOpen()
	case "/help":
		m.mode = modeHelp
		return m, nil
	case "/source-dir":
		m.mode = modeSourceDirPrompt
		m.prompt = m.prompt.ActivateText(fmt.Sprintf("Source dir (%s):", m.cfg.SourceDir))
		return m, nil
	case "/target-dir":
		m.mode = modeTargetDirPrompt
		m.prompt = m.prompt.ActivateText(fmt.Sprintf("Target dir (%s):", m.cfg.TargetDir))
		return m, nil
	case "/sort-by-name":
		m.branchList = m.branchList.SetSortMode(SortByName)
		return m, nil
	case "/sort-by-date":
		m.branchList = m.branchList.SetSortMode(SortByDate)
		return m, nil
	}
	return m, nil
}

func (m AppModel) enterNewMode() (tea.Model, tea.Cmd) {
	m.isNewFlow = true
	m.mode = modeRepoSelect
	repos, err := core.DiscoverRepos(m.cfg.SourceDir, m.cfg.ScanDepth)
	m.repoSelect = m.repoSelect.Activate(repos, nil, false, "new feature branch")
	if err != nil {
		return m, m.setError(err.Error())
	}
	return m, nil
}

func (m AppModel) enterUpdateMode(branch core.FeatureBranch) (tea.Model, tea.Cmd) {
	m.isNewFlow = false
	m.pendingBranch = branch.Name
	m.mode = modeRepoSelect
	repos, err := core.DiscoverRepos(m.cfg.SourceDir, m.cfg.ScanDepth)
	m.repoSelect = m.repoSelect.Activate(repos, branch.Repos, true, "update "+branch.Name)
	if err != nil {
		return m, m.setError(err.Error())
	}
	return m, nil
}

func (m AppModel) enterDeleteMode() (tea.Model, tea.Cmd) {
	br, ok := m.branchList.SelectedBranch()
	if !ok {
		return m, nil
	}
	m.pendingBranch = br.Name
	m.mode = modeDeleteConfirm
	label := fmt.Sprintf("Delete %q? Removes worktrees & branches.", br.Name)
	m.prompt = m.prompt.ActivateConfirm(label)
	return m, nil
}

func (m AppModel) enterRenameMode() (tea.Model, tea.Cmd) {
	br, ok := m.branchList.SelectedBranch()
	if !ok {
		return m, nil
	}
	m.pendingBranch = br.Name
	m.mode = modeRenamePrompt
	m.prompt = m.prompt.ActivateText(fmt.Sprintf("Rename %q to:", br.Name))
	return m, nil
}

// --- Prompt/confirm result handling ---

func (m AppModel) handlePromptResult(msg PromptResultMsg) (tea.Model, tea.Cmd) {
	if msg.Cancelled {
		if m.mode == modeBranchNamePrompt {
			m.mode = modeRepoSelect
			return m, nil
		}
		m.mode = modeBranchList
		return m, nil
	}

	switch m.mode {
	case modeBranchNamePrompt:
		m.pendingBranch = strings.TrimSpace(msg.Value)
		if m.pendingBranch == "" {
			m.errMsg = "branch name cannot be empty"
			return m, nil
		}
		return m.runCreate()

	case modeRenamePrompt:
		newName := strings.TrimSpace(msg.Value)
		if newName == "" {
			m.mode = modeBranchList
			return m, nil
		}
		return m.runRename(newName)

	case modeSourceDirPrompt:
		dir := strings.TrimSpace(msg.Value)
		if dir != "" {
			m.cfg.SourceDir = dir
			m.watcher.SetSourceDir(dir)
			if err := core.SaveConfig(m.cfgPath, m.cfg); err != nil {
				m.mode = modeBranchList
				return m, m.setError("save config: " + err.Error())
			}
		}
		m.mode = modeBranchList
		return m, nil

	case modeTargetDirPrompt:
		dir := strings.TrimSpace(msg.Value)
		if dir != "" {
			m.cfg.TargetDir = dir
			m.watcher.SetTargetDir(dir)
			if err := core.SaveConfig(m.cfgPath, m.cfg); err != nil {
				m.mode = modeBranchList
				return m, tea.Batch(m.loadBranches, m.setError("save config: "+err.Error()))
			}
		}
		m.mode = modeBranchList
		return m, m.loadBranches
	}

	m.mode = modeBranchList
	return m, nil
}

func (m AppModel) handleConfirmResult(msg ConfirmResultMsg) (tea.Model, tea.Cmd) {
	if !msg.Confirmed {
		m.mode = modeBranchList
		return m, nil
	}
	switch m.mode {
	case modeDeleteConfirm:
		return m.runDelete()
	case modeDirtyConfirm:
		return m.runUpdate(true)
	}
	m.mode = modeBranchList
	return m, nil
}

// --- Operations (run in background) ---

func (m AppModel) runCreate() (tea.Model, tea.Cmd) {
	m.mode = modeSpinner
	m.spinnerMsg = "Creating worktrees..."
	repos := m.pendingRepos
	branch := m.pendingBranch
	cfg := m.cfg
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			var errs []string
			if err := core.CreateWorktrees(cfg.SourceDir, repos, branch, cfg.TargetDir); err != nil {
				errs = append(errs, err.Error())
			}
			branchDir := cfg.TargetDir + "/" + core.BranchToDirName(branch)
			actual := core.ListReposOnDisk(branchDir)
			if err := core.CreateCursorWorkspace(branchDir, actual); err != nil {
				errs = append(errs, "workspace file: "+err.Error())
			}
			if cfg.PostCommand != "" {
				if err := core.RunPostCommand(cfg.PostCommand, branchDir); err != nil {
					errs = append(errs, "post_command: "+err.Error())
				}
			}
			return OperationDoneMsg{Err: joinErrors(errs)}
		},
	)
}

func (m AppModel) checkDirtyBeforeUpdate() (tea.Model, tea.Cmd) {
	repos := m.pendingRepos
	branch := m.pendingBranch
	targetDir := m.cfg.TargetDir
	return m, func() tea.Msg {
		dirty := core.DirtyRemovedWorktrees(repos, branch, targetDir)
		return DirtyWorktreesMsg{DirtyRepos: dirty}
	}
}

func (m AppModel) runUpdate(forceRemove bool) (tea.Model, tea.Cmd) {
	m.mode = modeSpinner
	m.spinnerMsg = "Updating worktrees..."
	repos := m.pendingRepos
	branch := m.pendingBranch
	cfg := m.cfg
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			var errs []string
			if err := core.UpdateFeatureBranch(cfg.SourceDir, repos, branch, cfg.TargetDir, forceRemove); err != nil {
				errs = append(errs, err.Error())
			}
			branchDir := cfg.TargetDir + "/" + core.BranchToDirName(branch)
			actual := core.ListReposOnDisk(branchDir)
			if err := core.CreateCursorWorkspace(branchDir, actual); err != nil {
				errs = append(errs, "workspace file: "+err.Error())
			}
			return OperationDoneMsg{Err: joinErrors(errs)}
		},
	)
}

func (m AppModel) runDelete() (tea.Model, tea.Cmd) {
	m.mode = modeSpinner
	m.spinnerMsg = "Deleting feature branch..."
	branch := m.pendingBranch
	targetDir := m.cfg.TargetDir
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			err := core.DeleteFeatureBranch(targetDir, branch, true)
			return OperationDoneMsg{Err: err}
		},
	)
}

func (m AppModel) runRename(newName string) (tea.Model, tea.Cmd) {
	m.mode = modeSpinner
	m.spinnerMsg = "Renaming feature branch..."
	oldName := m.pendingBranch
	targetDir := m.cfg.TargetDir
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			err := core.RenameFeatureBranch(targetDir, oldName, newName)
			return OperationDoneMsg{Err: err}
		},
	)
}

func (m AppModel) runOpen() (tea.Model, tea.Cmd) {
	br, ok := m.branchList.SelectedBranch()
	if !ok {
		return m, nil
	}
	if m.cfg.PostCommand == "" {
		return m, m.setError("post_command is not configured")
	}
	branchDir := m.cfg.TargetDir + "/" + core.BranchToDirName(br.Name)
	cfg := m.cfg
	return m, func() tea.Msg {
		if err := core.RunPostCommand(cfg.PostCommand, branchDir); err != nil {
			return OperationDoneMsg{Err: fmt.Errorf("post_command: %w", err)}
		}
		return OperationDoneMsg{}
	}
}

func (m AppModel) runPull() (tea.Model, tea.Cmd) {
	m.mode = modeSpinner
	m.spinnerMsg = "Pulling source repos..."
	sourceDir := m.cfg.SourceDir
	scanDepth := m.cfg.ScanDepth
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			err := core.PullSourceRepos(sourceDir, scanDepth)
			return OperationDoneMsg{Err: err}
		},
	)
}

func (m AppModel) helpView() string {
	type entry struct{ key, desc string }
	shortcuts := []entry{
		{"j / ↓", "move down"},
		{"k / ↑", "move up"},
		{"ENTER", "update repos in selected branch"},
		{"o", "run post_command for selected branch"},
		{"d / DEL", "delete selected branch"},
		{"/ + command", "command palette"},
	}
	commands := []entry{
		{"/new", "create a new feature branch"},
		{"/delete", "delete selected branch"},
		{"/rename", "rename selected branch"},
		{"/pull", "git pull --no-tags all repos in source-dir"},
		{"/sort-by-date", "sort branches by creation date"},
		{"/sort-by-name", "sort branches alphabetically"},
		{"/source-dir", "change source repos directory"},
		{"/target-dir", "change target (branches) directory"},
	}

	keyW := 16
	var b strings.Builder
	b.WriteString(styleHeader.Render("  Shortcuts") + "\n\n")
	for _, e := range shortcuts {
		b.WriteString("  " + styleCheckOn.Render(padRight(e.key, keyW)) + "  " + e.desc + "\n")
	}
	b.WriteString("\n" + styleHeader.Render("  Commands") + "\n\n")
	for _, e := range commands {
		b.WriteString("  " + styleCheckOn.Render(padRight(e.key, keyW)) + "  " + e.desc + "\n")
	}
	b.WriteString("\n" + styleHint.Render("  press any key to close") + "\n")
	return b.String()
}

// --- Data loading commands ---

type branchesLoadedMsg struct {
	branches []core.FeatureBranch
	err      error
}

type reposLoadedMsg struct {
	repos []core.RepoEntry
	err   error
}

func (m AppModel) loadBranches() tea.Msg {
	branches, err := core.ListFeatureBranches(m.cfg.TargetDir)
	return branchesLoadedMsg{branches: branches, err: err}
}

func (m AppModel) loadRepos() tea.Msg {
	repos, err := core.DiscoverRepos(m.cfg.SourceDir, m.cfg.ScanDepth)
	return reposLoadedMsg{repos: repos, err: err}
}

func waitForWatchEvent(ch <-chan core.WatchEvent) tea.Cmd {
	return func() tea.Msg {
		ev := <-ch
		return WatchEventMsg{Event: ev}
	}
}

func repoNames(repos []core.RepoEntry) []string {
	names := make([]string, len(repos))
	for i, r := range repos {
		names[i] = r.Name
	}
	return names
}

func (m *AppModel) setError(msg string) tea.Cmd {
	m.errMsg = msg
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func joinErrors(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(errs, "; "))
}
