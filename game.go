package main


import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

//Constants

const (
	gW             = 70 // game field width in chars
	gH             = 30 // game field height in rows
	gLives         = 5
	gLevelKills    = 10  // kills to level up
	gMaxEnemies    = 5   // max simultaneous enemies
	gTickMs        = 150 // ms per game tick (increased from 120 for slower pace)
	gSpawnEvery    = 5   // ticks between spawns (increased from 4)
	gProjectileSpd = 3   // rows per tick projectile moves
)

// Data structures
type Enemy struct {
	id       int
	word     string
	progress int     // chars correctly typed
	x        int     // left edge of word, 0-based
	y        float64 // vertical position, floats for smooth movement
	isBoss   bool
	alive    bool
}

type Projectile struct {
	x, y   float64
	tx, ty float64 // target coords
	done   bool
}

type GameState struct {
	enemies     []Enemy
	projectiles []Projectile
	score       int
	lives       int
	level       int
	killed      int // total enemies killed this game
	tickN       int // tick counter for spawning
	locked      int // ID of locked enemy (-1 = none)
	nextID      int
	totalKeys   int
	errors      int
	startTime   time.Time
	over        bool
	finalWPM    float64
	finalAcc    float64
	speed       float64 // rows per tick enemies fall
}

// Game tick message

type gameTickMsg struct{}

func gameTickCmd() tea.Cmd {
	return tea.Tick(gTickMs*time.Millisecond, func(t time.Time) tea.Msg {
		return gameTickMsg{}
	})
}

// Init / reset

func newGameState() *GameState {
	gs := &GameState{
		lives:  gLives,
		level:  1,
		locked: -1,
		speed:  0.25, // reduced from 0.4 for slower initial speed
	}
	gs.startTime = time.Now()
	// Seed with 2 enemies so field isn't empty
	gs.spawnEnemy()
	gs.spawnEnemy()
	return gs
}

// Spawning
func (gs *GameState) spawnEnemy() {
	if len(gs.aliveEnemies()) >= gMaxEnemies {
		return
	}
	isBoss := gs.killed > 0 && gs.killed%15 == 0

	word := generateGameWord(isBoss)

	// Ensure first letter is unique among alive enemies so targeting is unambiguous
	alive := gs.aliveEnemies()
	for attempt := 0; attempt < 20; attempt++ {
		duplicate := false
		for _, e := range alive {
			if len(e.word) > 0 && len(word) > 0 && e.word[0] == word[0] {
				duplicate = true
				break
			}
		}
		if !duplicate {
			break
		}
		word = generateGameWord(isBoss)
	}

	// Random x position keeping word inside field
	maxX := gW - len(word) - 4
	if maxX < 2 {
		maxX = 2
	}
	x := rand.Intn(maxX) + 2

	gs.enemies = append(gs.enemies, Enemy{
		id:     gs.nextID,
		word:   word,
		x:      x,
		y:      0,
		isBoss: isBoss,
		alive:  true,
	})
	gs.nextID++
}

func (gs *GameState) aliveEnemies() []Enemy {
	out := make([]Enemy, 0, len(gs.enemies))
	for _, e := range gs.enemies {
		if e.alive {
			out = append(out, e)
		}
	}
	return out
}

// Tick update

func (gs *GameState) tick() {
	gs.tickN++

	// Move enemies down
	for i := range gs.enemies {
		if !gs.enemies[i].alive {
			continue
		}
		spd := gs.speed
		if gs.enemies[i].isBoss {
			spd *= 0.6
		}
		gs.enemies[i].y += spd

		// Enemy reached the bottom — lose a life
		if gs.enemies[i].y >= float64(gH-5) {
			gs.lives--
			gs.enemies[i].alive = false
			if gs.locked == gs.enemies[i].id {
				gs.locked = -1
			}
			if gs.lives <= 0 {
				gs.endGame()
				return
			}
		}
	}

	// Move projectiles toward their targets
	for i := range gs.projectiles {
		p := &gs.projectiles[i]
		if p.done {
			continue
		}
		dx := p.tx - p.x
		dy := p.ty - p.y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < 1.5 {
			p.done = true
			continue
		}
		step := float64(gProjectileSpd)
		p.x += dx / dist * step
		p.y += dy / dist * step
	}

	// Spawn
	if gs.tickN%gSpawnEvery == 0 {
		gs.spawnEnemy()
	}
}

// Input handling

// handleGameKey processes one keypress. Returns true if game ended.
func (gs *GameState) handleGameKey(ch rune) {
	if gs.over {
		return
	}
	gs.totalKeys++

	// If no locked target, find one starting with this char
	if gs.locked == -1 {
		for i := range gs.enemies {
			e := &gs.enemies[i]
			if !e.alive || len(e.word) == 0 {
				continue
			}
			if rune(e.word[e.progress]) == ch {
				gs.locked = e.id
				break
			}
		}
	}

	if gs.locked == -1 {
		gs.errors++
		return
	}

	// Find locked enemy
	idx := gs.lockedIdx()
	if idx < 0 {
		gs.locked = -1
		gs.errors++
		return
	}

	e := &gs.enemies[idx]
	expected := rune(e.word[e.progress])

	if ch != expected {
		gs.errors++
		return
	}

	// Correct character
	e.progress++

	// Fire a projectile
	// Player is centered at bottom; enemy text starts at e.x
	px := float64(gW / 2)
	py := float64(gH - 4)
	tx := float64(e.x) + float64(e.progress)
	ty := e.y + 1
	gs.projectiles = append(gs.projectiles, Projectile{
		x: px, y: py, tx: tx, ty: ty,
	})

	// Word complete?
	if e.progress >= len(e.word) {
		e.alive = false
		gs.locked = -1
		pts := len(e.word) * gs.level
		if e.isBoss {
			pts *= 3
		}
		gs.score += pts
		gs.killed++

		// Level up
		if gs.killed%gLevelKills == 0 {
			gs.level++
			gs.speed += 0.04 // reduced from 0.08 for slower progression
		}

		gs.spawnEnemy()
	}
}

func (gs *GameState) lockedIdx() int {
	for i := range gs.enemies {
		if gs.enemies[i].id == gs.locked {
			return i
		}
	}
	return -1
}

func (gs *GameState) endGame() {
	gs.over = true
	elapsed := time.Since(gs.startTime).Minutes()
	correct := gs.totalKeys - gs.errors
	if elapsed > 0 {
		gs.finalWPM = float64(correct) / 5.0 / elapsed
	}
	if gs.totalKeys > 0 {
		gs.finalAcc = float64(correct) / float64(gs.totalKeys) * 100
	}
}

// Rendering

func (gs *GameState) render(width, height int) string {
	// Build a rune grid for the play field
	type cell struct {
		ch    rune
		style lipgloss.Style
	}

	dim := lipgloss.NewStyle().Foreground(activeTheme.surface1)
	blank := cell{' ', dim}

	grid := make([][]cell, gH)
	for i := range grid {
		grid[i] = make([]cell, gW)
		for j := range grid[i] {
			grid[i][j] = blank
		}
	}

	paint := func(y, x int, ch rune, s lipgloss.Style) {
		if y >= 0 && y < gH && x >= 0 && x < gW {
			grid[y][x] = cell{ch, s}
		}
	}
	paintStr := func(y, x int, s string, st lipgloss.Style) {
		for i, ch := range s {
			paint(y, x+i, ch, st)
		}
	}

	// Starfield (deterministic by tick for stability)
	starStyle := lipgloss.NewStyle().Foreground(activeTheme.surface0)
	stars := []struct{ x, y int }{
		{5, 2}, {12, 5}, {22, 1}, {35, 8}, {48, 3},
		{60, 6}, {3, 12}, {18, 15}, {40, 11}, {55, 14},
		{8, 18}, {30, 17}, {65, 9}, {45, 19},
	}
	for _, st := range stars {
		if st.y < gH-5 {
			ch := '·'
			if (st.x+st.y+gs.tickN/4)%7 == 0 {
				ch = '✦'
			}
			paint(st.y, st.x, ch, starStyle)
		}
	}

	// Projectiles
	projStyle := lipgloss.NewStyle().Foreground(activeTheme.yellow).Bold(true)
	for _, p := range gs.projectiles {
		if p.done {
			continue
		}
		py := int(p.y)
		px := int(p.x)
		paint(py, px, '•', projStyle)
	}

	//  enemies
	for i := range gs.enemies {
		e := &gs.enemies[i]
		if !e.alive {
			continue
		}

		ey := int(e.y)
		ex := e.x
		isLocked := gs.locked == e.id

		// Choose styles
		var shipStyle, typedStyle, pendingStyle, hullStyle lipgloss.Style
		if e.isBoss {
			shipStyle = lipgloss.NewStyle().Foreground(activeTheme.red).Bold(true)
			typedStyle = lipgloss.NewStyle().Foreground(activeTheme.surface2)
			pendingStyle = lipgloss.NewStyle().Foreground(activeTheme.red).Bold(true)
			hullStyle = lipgloss.NewStyle().Foreground(activeTheme.red)
		} else if isLocked {
			shipStyle = lipgloss.NewStyle().Foreground(activeTheme.yellow).Bold(true)
			typedStyle = lipgloss.NewStyle().Foreground(activeTheme.surface2)
			pendingStyle = lipgloss.NewStyle().Foreground(activeTheme.yellow).Bold(true)
			hullStyle = lipgloss.NewStyle().Foreground(activeTheme.yellow)
		} else {
			shipStyle = lipgloss.NewStyle().Foreground(activeTheme.mauve)
			typedStyle = lipgloss.NewStyle().Foreground(activeTheme.surface2)
			pendingStyle = lipgloss.NewStyle().Foreground(activeTheme.mauve)
			hullStyle = lipgloss.NewStyle().Foreground(activeTheme.surface1)
		}

		// Ship art above word
		wordW := len(e.word)
		if e.isBoss {
			// Boss: wider ship
			// Row -3: engine glow
			paintStr(ey-3, ex, strings.Repeat("≡", wordW), hullStyle)
			// Row -2: hull top
			wing := strings.Repeat("━", wordW/2)
			paintStr(ey-2, ex, "╔"+wing+"╗", shipStyle)
			// Row -1: cockpit
			mid := strings.Repeat("─", wordW-2)
			paintStr(ey-1, ex, "║"+mid+"║", shipStyle)
			// Row 0: cannon
			paint(ey, ex, '╚', shipStyle)
			paint(ey, ex+wordW, '╝', shipStyle)
			paintStr(ey, ex+1, strings.Repeat("═", wordW-1), hullStyle)
		} else {
			// Regular: compact 2-row ship
			mid := wordW / 2
			// Row -2: nose
			paint(ey-2, ex+mid, '▲', shipStyle)
			// Row -1: wings
			left := strings.Repeat("═", mid)
			right := strings.Repeat("═", wordW-mid-1)
			paintStr(ey-1, ex, left, hullStyle)
			paint(ey-1, ex+mid, '╪', shipStyle)
			paintStr(ey-1, ex+mid+1, right, hullStyle)
		}

		// Word row — typed chars dimmed, remaining chars bright
		for ci, ch := range e.word {
			st := pendingStyle
			if ci < e.progress {
				st = typedStyle
			} else if isLocked && ci == e.progress {
				// Next char to type — highlight it
				st = lipgloss.NewStyle().Foreground(activeTheme.green).Bold(true)
			}
			paint(ey+1, ex+ci, ch, st)
		}

		// Engine glow below word
		exhaust := strings.Repeat("▾", wordW)
		exhaustStyle := lipgloss.NewStyle().Foreground(activeTheme.peach)
		if isLocked || e.isBoss {
			exhaustStyle = lipgloss.NewStyle().Foreground(activeTheme.yellow)
		}
		paintStr(ey+2, ex, exhaust, exhaustStyle)
	}

	// Player ship at bottom
	playerX := gW/2 - 4
	playerY := gH - 4
	pStyle := lipgloss.NewStyle().Foreground(activeTheme.green).Bold(true)
	// paint(playerY-1, playerX+4,'▲', pStyle)
	paintStr(playerY, playerX, "╔╪╗", pStyle)
	paintStr(playerY+2, playerX, "╚▽╝", pStyle)

	// Cannon beam when locked and projectile exists
	if gs.locked >= 0 {
		for _, p := range gs.projectiles {
			if !p.done && int(p.y) < playerY-1 {
				beamY := int(p.y)
				beamX := int(p.x)
				bStyle := lipgloss.NewStyle().Foreground(activeTheme.green).Bold(true)
				paint(beamY, beamX, '*', bStyle)
			}
		}
	}

	// Floor line
	floorStyle := lipgloss.NewStyle().Foreground(activeTheme.surface0)
	paintStr(gH-5, 0, strings.Repeat("─", gW), floorStyle)

	// Render grid to string
	var sb strings.Builder
	for y, row := range grid {
		for _, c := range row {
			sb.WriteString(c.style.Render(string(c.ch)))
		}
		if y < gH-1 {
			sb.WriteRune('\n')
		}
	}

	// HUD: lives, score, level 
	hearts := ""
	for i := 0; i < gLives; i++ {
		if i < gs.lives {
			hearts += lipgloss.NewStyle().Foreground(activeTheme.red).Bold(true).Render("♥ ")
		} else {
			hearts += lipgloss.NewStyle().Foreground(activeTheme.surface1).Render("♡ ")
		}
	}
	scoreStr := lipgloss.NewStyle().Foreground(activeTheme.yellow).Bold(true).
		Render(fmt.Sprintf("score %d", gs.score))
	levelStr := lipgloss.NewStyle().Foreground(activeTheme.mauve).Bold(true).
		Render(fmt.Sprintf("level %d", gs.level))
	killStr := lipgloss.NewStyle().Foreground(activeTheme.subtext0).
		Render(fmt.Sprintf("killed %d", gs.killed))

	hud := lipgloss.JoinHorizontal(lipgloss.Top,
		hearts, "   ", scoreStr, "   ", levelStr, "   ", killStr,
	)

	hint := lipgloss.NewStyle().Foreground(activeTheme.surface1).
		Render("ctrl+g menu  esc quit")

	body := lipgloss.JoinVertical(lipgloss.Left,
		hud,
		sb.String(),
		hint,
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, body)
}

// Game over screen

func (gs *GameState) renderGameOver(width, height int) string {
	titleStyle := lipgloss.NewStyle().Foreground(activeTheme.red).Bold(true)
	numStyle := lipgloss.NewStyle().Foreground(activeTheme.yellow).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(activeTheme.subtext0)
	accStyle := lipgloss.NewStyle().Foreground(activeTheme.teal).Bold(true)
	wpmStyle := lipgloss.NewStyle().Foreground(activeTheme.wpm).Bold(true)

	col := func(val, label string) string {
		return lipgloss.NewStyle().Width(14).Render(
			lipgloss.JoinVertical(lipgloss.Left, val, labelStyle.Render(label)))
	}

	statsRow := lipgloss.JoinHorizontal(lipgloss.Bottom,
		col(wpmStyle.Render(fmt.Sprintf("%.0f", gs.finalWPM)), "wpm"),
		col(accStyle.Render(fmt.Sprintf("%.1f%%", gs.finalAcc)), "accuracy"),
		col(numStyle.Render(fmt.Sprintf("%d", gs.score)), "score"),
		col(numStyle.Render(fmt.Sprintf("%d", gs.level)), "level"),
		col(numStyle.Render(fmt.Sprintf("%d", gs.killed)), "destroyed"),
	)

	sep := lipgloss.NewStyle().Foreground(activeTheme.surface0).Render(strings.Repeat("─", 14*5))

	// ASCII game over art
	art := lipgloss.NewStyle().Foreground(activeTheme.red).Bold(true).Render(
		    "╔═══════════════════╗\n" +
			"║ G A M E  O V E R  ║\n" +
			"╚═══════════════════╝",
	)

	actions := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Foreground(activeTheme.green).Bold(true).Render("enter"),
		lipgloss.NewStyle().Foreground(activeTheme.subtext0).Render(" play again   "),
		lipgloss.NewStyle().Foreground(activeTheme.mauve).Bold(true).Render("m"),
		lipgloss.NewStyle().Foreground(activeTheme.subtext0).Render(" menu   "),
		lipgloss.NewStyle().Foreground(activeTheme.surface2).Render("esc"),
		lipgloss.NewStyle().Foreground(activeTheme.subtext0).Render(" quit"),
	)
	_ = titleStyle

	body := lipgloss.JoinVertical(lipgloss.Center,
		art, "", statsRow, "", sep, "", actions,
	)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, body)
}

// Model integration called from model.go

func (m Model) updateGameMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyCtrlG:
		next := m.goToMenu()
		return next, nil
	case tea.KeyCtrlT:
		m.toggleTheme()
		return m, nil
	case tea.KeyEsc:
		m.prevState = stateGame
		m.state = stateConfirmQuit
		return m, nil
	case tea.KeyRunes:
		if m.gameState != nil {
			for _, r := range msg.Runes {
				m.gameState.handleGameKey(r)
			}
			if m.gameState.over {
				m.state = stateGameOver
				return m, nil
			}
		}
	}
	return m, nil
}

func (m Model) updateGameOver(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyCtrlT:
		m.toggleTheme()
		return m, nil
	case tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyEnter:
		// Restart game
		m.gameState = newGameState()
		m.state = stateGame
		return m, gameTickCmd()
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "m", "M":
			next := m.goToMenu()
			return next, nil
		case "r", "R":
			m.gameState = newGameState()
			m.state = stateGame
			return m, gameTickCmd()
		}
	}
	return m, nil
}
