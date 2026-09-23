// Package clipboard copies text to the system clipboard using the native tool
// available on the host (pbcopy on macOS, wl-copy/xclip/xsel on Linux, clip on
// Windows). It shells out rather than pulling a cgo dependency, which keeps the
// build simple and works in every terminal (including Apple Terminal.app).
package clipboard

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// tool is a candidate clipboard command: the executable plus its arguments.
type tool struct {
	name string
	args []string
}

// candidates returns the ordered clipboard tools to try for the current OS.
func candidates() []tool {
	switch runtime.GOOS {
	case "darwin":
		return []tool{{name: "pbcopy"}}
	case "windows":
		return []tool{{name: "clip"}}
	default: // linux, *bsd
		return []tool{
			{name: "wl-copy"},                       // Wayland
			{name: "xclip", args: []string{"-selection", "clipboard"}}, // X11
			{name: "xsel", args: []string{"--clipboard", "--input"}},   // X11 alt
		}
	}
}

// Copy writes text to the system clipboard. It returns an explicit error listing
// the tools it looked for when none is available, so the UI can guide the user.
func Copy(text string) error {
	tools := candidates()

	var missing []string
	for _, t := range tools {
		path, err := exec.LookPath(t.name)
		if err != nil {
			missing = append(missing, t.name)
			continue
		}
		cmd := exec.Command(path, t.args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", t.name, err)
		}
		return nil
	}

	return fmt.Errorf("no clipboard tool found (tried: %s)", strings.Join(missing, ", "))
}
