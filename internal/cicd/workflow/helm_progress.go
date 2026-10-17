package workflow

import "strings"

// Keep lifecycle evidence without recording Chart contents, values, or Helm's
// debug output (which can contain Secret data). The worker persists callbacks
// immediately; callers without a callback receive the same evidence in LogText.
type helmTaskProgress struct {
	write func(string) error
	lines []string
}

func newHelmTaskProgress(write func(string) error) *helmTaskProgress {
	return &helmTaskProgress{write: write}
}

func (p *helmTaskProgress) report(message string) error {
	if p.write != nil {
		return p.write(message)
	}
	p.lines = append(p.lines, message)
	return nil
}

func (p *helmTaskProgress) text(final string) string {
	return strings.TrimSpace(strings.Join(append(p.lines, final), "\n"))
}
