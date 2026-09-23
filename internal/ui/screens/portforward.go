package screens

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// pf form field indices.
const (
	fieldRemotePort = iota
	fieldLocalPort
	fieldHost
	pfFieldCount
)

// PortForwardForm collects the parameters for an SSM port-forward. When RemoteHost
// is enabled, the Host field is included; otherwise the tunnel targets a port on the
// instance itself.
type PortForwardForm struct {
	inputs     []textinput.Model
	focus      int
	RemoteHost bool
	Target     string // instance id (display only)
	TargetName string
}

// NewPortForwardForm builds the form for a given target instance.
func NewPortForwardForm(target, targetName string) PortForwardForm {
	inputs := make([]textinput.Model, pfFieldCount)

	inputs[fieldRemotePort] = textinput.New()
	inputs[fieldRemotePort].Placeholder = "5432"
	inputs[fieldRemotePort].CharLimit = 5
	inputs[fieldRemotePort].Width = 12
	inputs[fieldRemotePort].Focus()

	inputs[fieldLocalPort] = textinput.New()
	inputs[fieldLocalPort].Placeholder = "15432"
	inputs[fieldLocalPort].CharLimit = 5
	inputs[fieldLocalPort].Width = 12

	inputs[fieldHost] = textinput.New()
	inputs[fieldHost].Placeholder = "db.internal"
	inputs[fieldHost].CharLimit = 253
	inputs[fieldHost].Width = 30

	return PortForwardForm{
		inputs:     inputs,
		focus:      fieldRemotePort,
		Target:     target,
		TargetName: targetName,
	}
}

// ToggleRemoteHost switches between instance-port and remote-host modes.
func (f *PortForwardForm) ToggleRemoteHost() {
	f.RemoteHost = !f.RemoteHost
	if !f.RemoteHost && f.focus == fieldHost {
		f.focus = fieldRemotePort
		f.syncFocus()
	}
}

// fieldsInUse returns the field indices active in the current mode.
func (f *PortForwardForm) fieldsInUse() []int {
	if f.RemoteHost {
		return []int{fieldRemotePort, fieldLocalPort, fieldHost}
	}
	return []int{fieldRemotePort, fieldLocalPort}
}

// Next moves focus to the next field (wrapping).
func (f *PortForwardForm) Next() {
	fields := f.fieldsInUse()
	cur := indexOf(fields, f.focus)
	f.focus = fields[(cur+1)%len(fields)]
	f.syncFocus()
}

// Prev moves focus to the previous field (wrapping).
func (f *PortForwardForm) Prev() {
	fields := f.fieldsInUse()
	cur := indexOf(fields, f.focus)
	f.focus = fields[(cur-1+len(fields))%len(fields)]
	f.syncFocus()
}

func (f *PortForwardForm) syncFocus() {
	for i := range f.inputs {
		if i == f.focus {
			f.inputs[i].Focus()
		} else {
			f.inputs[i].Blur()
		}
	}
}

// Update forwards key events to the focused input.
func (f *PortForwardForm) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	f.inputs[f.focus], cmd = f.inputs[f.focus].Update(msg)
	return cmd
}

// Values returns the validated form values.
func (f *PortForwardForm) Values() (host, remotePort, localPort string, err error) {
	remotePort = strings.TrimSpace(f.inputs[fieldRemotePort].Value())
	localPort = strings.TrimSpace(f.inputs[fieldLocalPort].Value())
	host = strings.TrimSpace(f.inputs[fieldHost].Value())

	if !validPort(remotePort) {
		return "", "", "", fmt.Errorf("invalid remote port: %q", remotePort)
	}
	if !validPort(localPort) {
		return "", "", "", fmt.Errorf("invalid local port: %q", localPort)
	}
	if f.RemoteHost && host == "" {
		return "", "", "", fmt.Errorf("remote host required in remote host mode")
	}
	if !f.RemoteHost {
		host = ""
	}
	return host, remotePort, localPort, nil
}

// View renders the form.
func (f *PortForwardForm) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Remote port : %s\n", f.inputs[fieldRemotePort].View())
	fmt.Fprintf(&b, "Local port  : %s\n", f.inputs[fieldLocalPort].View())
	if f.RemoteHost {
		fmt.Fprintf(&b, "Remote host : %s\n", f.inputs[fieldHost].View())
	}
	return b.String()
}

func validPort(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n > 0 && n <= 65535
}

func indexOf(s []int, v int) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return 0
}
