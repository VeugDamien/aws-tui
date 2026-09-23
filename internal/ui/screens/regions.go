package screens

import (
	"github.com/charmbracelet/bubbles/list"
)

// RegionItem adapts a region string to the list.DefaultItem interface.
type RegionItem struct {
	Region string
}

func (i RegionItem) Title() string       { return i.Region }
func (i RegionItem) Description() string { return "" }
func (i RegionItem) FilterValue() string { return i.Region }

// NewRegionList builds a list model for the given regions.
func NewRegionList(regions []string, width, height int) list.Model {
	items := make([]list.Item, 0, len(regions))
	for _, r := range regions {
		items = append(items, RegionItem{Region: r})
	}
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	l := list.New(items, delegate, width, height)
	l.Title = "Choose a region"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	return l
}
