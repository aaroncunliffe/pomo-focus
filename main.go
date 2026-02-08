package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Testing
// const timeout = 5 * time.Second
// const shortBreak = 3 * time.Second
// const longBreak = 10 * time.Second

const timeout = 25 * time.Minute
const shortBreak = 5 * time.Minute
const longBreak = 15 * time.Minute

type SessionType int

const (
	SessionFocus SessionType = iota
	SessionShortBreak
	SessionLongBreak
)

type model struct {
	timer        timer.Model
	sessionType  SessionType
	sessionCount int
	keymap       keymap
	help         help.Model
	quitting     bool
	width        int
	height       int
}

type keymap struct {
	start      key.Binding
	stop       key.Binding
	next       key.Binding
	reset      key.Binding
	shortBreak key.Binding
	longBreak  key.Binding
	quit       key.Binding
}

func (m model) Init() tea.Cmd {
	return m.timer.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case timer.TickMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		return m, cmd

	case timer.StartStopMsg:
		var cmd tea.Cmd
		m.timer, cmd = m.timer.Update(msg)
		m.keymap.stop.SetEnabled(m.timer.Running())
		m.keymap.start.SetEnabled(!m.timer.Running())
		m.keymap.next.SetEnabled(false)
		return m, cmd

	case timer.TimeoutMsg:
		// Native terminal bell.
		fmt.Print("\a")
		m.keymap.stop.SetEnabled(false)
		m.keymap.start.SetEnabled(false)
		m.keymap.next.SetEnabled(true)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, m.keymap.shortBreak):
			// Short break doesn't reset session count.
			// TODO: currently advances the session when you leave this 'forced' break
			m.sessionType = SessionShortBreak
			m.timer.Timeout = shortBreak
			return m, m.timer.Start()
		case key.Matches(msg, m.keymap.longBreak):
			m.sessionCount = 0
			m.sessionType = SessionLongBreak
			m.timer.Timeout = longBreak
			return m, m.timer.Start()
		case key.Matches(msg, m.keymap.reset):
			m.sessionCount = 0
			m.timer.Timeout = timeout
			m.sessionType = SessionFocus
		case key.Matches(msg, m.keymap.start, m.keymap.stop):
			return m, m.timer.Toggle()
		case key.Matches(msg, m.keymap.next):
			switch m.sessionType {

			// Break Ending
			case SessionShortBreak, SessionLongBreak:
				m.sessionCount++
				m.sessionType = SessionFocus
				m.timer.Timeout = timeout

			// Focus ending
			case SessionFocus:
				if m.sessionCount == 4 {
					m.sessionCount = 0
					m.sessionType = SessionLongBreak
					m.timer.Timeout = longBreak
				} else {
					m.sessionType = SessionShortBreak
					m.timer.Timeout = shortBreak
				}
			}
			return m, m.timer.Start()
		}
	}

	return m, nil
}

func (m model) helpView() string {
	return m.help.ShortHelpView([]key.Binding{
		m.keymap.start,
		m.keymap.stop,
		m.keymap.next,
		m.keymap.reset,
		m.keymap.shortBreak,
		m.keymap.longBreak,
		m.keymap.quit,
	})
}

// Build the display lines for the terminal display
func (m model) View() string {
	timeout := m.timer.Timeout.Round(time.Second)
	s := ""

	// Colours
	timerColour := lipgloss.Color("#04B575")
	timerFinishedColour := lipgloss.Color("#FF0000")
	breakColour := lipgloss.Color("#9900ff")

	// ---------------------
	// Lipgloss Style blocks
	timerStyle := lipgloss.NewStyle().Foreground(timerColour).Bold(true)
	mainViewStyle := lipgloss.NewStyle().Padding(2, 4).Margin(4, 4, 0, 4).Border(lipgloss.NormalBorder(), true).Align(lipgloss.Center)
	helpViewStyle := lipgloss.NewStyle().Padding(2).Margin(0, 4, 3, 4)

	var backgroundStyles []lipgloss.WhitespaceOption
	backgroundStyles = append(backgroundStyles, lipgloss.WithWhitespaceChars("|-|"))

	// Timer not running styles
	if !m.timer.Running() {
		backgroundStyles = append(backgroundStyles,
			lipgloss.WithWhitespaceForeground(timerFinishedColour),
		)
	}
	// ---------------------

	switch m.sessionType {
	case SessionFocus:
		s += "Session Focus"
		if m.sessionCount == 4 {
			s += " - Long Break Next"
		}
		s += "\n"
		s += "Timer: " + timerStyle.Render(fmt.Sprintf("%s", timeout))
		s += "\n"
		s += fmt.Sprintf("Session Count: %d/4\n", m.sessionCount)

	case SessionShortBreak:
		s += "Session Short Break\n"
		s += "Timer: " + timerStyle.Render(fmt.Sprintf("%s", timeout))
		s += "\n"
		backgroundStyles = append(backgroundStyles, lipgloss.WithWhitespaceForeground(breakColour))

	case SessionLongBreak:
		s += "Session Long Break\n"
		s += "Timer: " + timerStyle.Render(fmt.Sprintf("%s", timeout))
		s += "\n"
		backgroundStyles = append(backgroundStyles, lipgloss.WithWhitespaceForeground(breakColour))
	}

	if m.timer.Timedout() {
		s += "Session Complete - Reset or Next"
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center,
			// Views to apply
			mainViewStyle.Render(s),
			helpViewStyle.Render(m.helpView()),
		),
		backgroundStyles...,
	)
}

func main() {
	timeoutValue := timeout

	// Manual timeout arg - First session only
	args := os.Args
	if len(args) > 1 && args[1] != "" {
		t, err := time.ParseDuration(args[1])
		if err != nil {
			log.Fatalf("timeout arg invalid: [%s]", err.Error())
		}
		timeoutValue = t
	}

	m := model{
		timer:        timer.NewWithInterval(timeoutValue, time.Millisecond),
		sessionType:  SessionFocus,
		sessionCount: 1,
		keymap: keymap{
			start: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "start"),
				key.WithDisabled(),
			),
			stop: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "stop"),
			),
			next: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "next"),
				key.WithDisabled(),
			),
			reset: key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "reset"),
			),
			shortBreak: key.NewBinding(
				key.WithKeys("k"),
				key.WithHelp("k", "short break"),
			),
			longBreak: key.NewBinding(
				key.WithKeys("l"),
				key.WithHelp("l", "long break"),
			),
			quit: key.NewBinding(
				key.WithKeys("q", "ctrl+c"),
				key.WithHelp("q", "quit"),
			),
		},
		help: help.New(),
	}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("error starting:", err)
		os.Exit(1)
	}
}
