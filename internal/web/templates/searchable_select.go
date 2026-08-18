package webtemplates

import "context"

// SearchableSelectOption is one selectable human-readable option.
type SearchableSelectOption struct {
	Value       string
	Label       string
	Description string
	Search      string
	UOM         string
	Disabled    bool
}

// SearchableSelectViewModel contains the server-owned state for a single select.
type SearchableSelectViewModel struct {
	ID          string
	Name        string
	Label       string
	Placeholder string
	Options     []SearchableSelectOption
	Selected    string
}

type OptionsProvider func(context.Context) ([]SearchableSelectOption, error)
