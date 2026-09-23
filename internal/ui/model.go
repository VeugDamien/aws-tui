// Package ui implements the Bubble Tea TUI: model, update, view and the async
// commands that bridge to the AWS-facing packages.
package ui

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/VeugDamien/aws-tui/internal/auth"
	"github.com/VeugDamien/aws-tui/internal/awsclient"
	"github.com/VeugDamien/aws-tui/internal/awsconfig"
	"github.com/VeugDamien/aws-tui/internal/ui/screens"
)

// Screen identifies the active view.
type Screen int

const (
	ScreenProfiles Screen = iota
	ScreenActions
	ScreenRegions
	ScreenEC2
	ScreenEC2Detail
	ScreenPortForwardForm
	ScreenTunnels
	ScreenASG
	ScreenASGDetail
	ScreenELB
	ScreenELBDetail
)

const defaultRegion = "eu-west-3"

// AppModel is the root Bubble Tea model.
type AppModel struct {
	screen Screen
	// returnScreen is where config overlays (profiles/regions) return to.
	returnScreen Screen

	// actionCursor is the highlighted entry in the main actions menu.
	actionCursor int

	profiles      []awsconfig.Profile
	activeProfile string
	activeRegion  string
	identity      *auth.CallerIdentity
	accountAlias  string
	awsCfg        aws.Config

	profileList screens.ProfileList
	regionList  list.Model
	ec2         screens.EC2Table
	ec2Detail   screens.EC2Detail
	pfForm      screens.PortForwardForm

	asg       screens.ASGTable
	asgDetail screens.ASGDetail
	elb       screens.ELBTable
	elbDetail screens.ELBDetail

	spinner spinner.Model
	help    help.Model
	keys    keyMap

	instances []awsclient.Instance
	asgs      []awsclient.AutoScalingGroup
	lbs       []awsclient.LoadBalancer

	// background tunnels
	tunnels      *tunnelManager
	tunnelCursor int
	// tunnelReturn is the screen to return to when leaving the tunnels view.
	tunnelReturn Screen
	// pfReturn is the screen to return to after the port-forward form (EC2 list or
	// instance detail), depending on where it was opened from.
	pfReturn Screen

	loading    bool
	loadingMsg string
	statusMsg  string
	statusSeq  int // bumped each time statusMsg is set; used to auto-clear
	err        error

	// startupDone guards the one-time restore of the last used profile.
	startupDone bool

	width, height int
}

// New constructs the initial model.
func New() AppModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	h := help.New()

	return AppModel{
		screen:       ScreenProfiles,
		activeRegion: defaultRegion,
		spinner:      sp,
		help:         h,
		keys:         defaultKeyMap(),
		tunnels:      newTunnelManager(),
		loading:      true,
		loadingMsg:   "Chargement des profils…",
	}
}

// Init kicks off the initial profile load and spinner.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(loadProfilesCmd(), m.spinner.Tick)
}

// currentProfile returns the awsconfig.Profile matching activeProfile.
func (m AppModel) currentProfile() awsconfig.Profile {
	for _, p := range m.profiles {
		if p.Name == m.activeProfile {
			return p
		}
	}
	return awsconfig.Profile{Name: m.activeProfile, AuthKind: awsconfig.AuthUnknown}
}

// contentHeight returns the height available for the main content area (the body),
// after reserving space for the surrounding chrome.
func (m AppModel) contentHeight() int {
	// Reserve: top/bottom margins (2) + header (1) + config bar (2: rule+line)
	// + blank separator (1) + status bar (2) + footer (1) + slack (1).
	const chrome = 10
	h := m.height - chrome
	if h < 5 {
		h = 5
	}
	return h
}

func (m AppModel) contentWidth() int {
	w := m.width - 4
	if w < 60 {
		w = 60
	}
	return w
}

// ec2ContentHeight is the height available for the EC2 table itself, reserving the
// extra lines viewEC2 draws above it (title line + filter line) and the scroll hint.
func (m AppModel) ec2ContentHeight() int {
	h := m.contentHeight() - 3
	if h < 3 {
		h = 3
	}
	return h
}
