package main

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/timer"
	tea "github.com/charmbracelet/bubbletea"
)

const timeout = 25 * time.Minute
const shortBreak = 5 * time.Minute
const longBreak = 15 * time.Minute

type model struct {
	timer    timer.Model
	keymap   keymap
	help     help.Model
	quitting bool
}

type keymap struct {
	start      key.Binding
	stop       key.Binding
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
		return m, cmd

	case timer.TimeoutMsg:
		// Native terminal bell.
		fmt.Print("\a")

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, m.keymap.shortBreak):
			m.timer.Timeout = shortBreak
			return m, m.timer.Start()
		case key.Matches(msg, m.keymap.longBreak):
			m.timer.Timeout = longBreak
			return m, m.timer.Start()
		case key.Matches(msg, m.keymap.reset):
			m.timer.Timeout = timeout
		case key.Matches(msg, m.keymap.start, m.keymap.stop):
			return m, m.timer.Toggle()
		}
	}

	return m, nil
}

func (m model) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.start,
		m.keymap.stop,
		m.keymap.reset,
		m.keymap.shortBreak,
		m.keymap.longBreak,
		m.keymap.quit,
	})
}

func (m model) View() string {

	timeout := m.timer.Timeout
	s := fmt.Sprintf("timer: %s", timeout.Round(time.Second))

	if m.timer.Timedout() {
		s = "Session Complete - Reset or Break"
	}
	s += "\n"
	if !m.quitting {
		s += m.helpView()
	}
	return s
}

func main() {
	m := model{
		timer: timer.NewWithInterval(timeout, time.Millisecond),
		keymap: keymap{
			start: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "start"),
			),
			stop: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "stop"),
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

	m.keymap.start.SetEnabled(false)

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("error starting:", err)
		os.Exit(1)
	}
}
