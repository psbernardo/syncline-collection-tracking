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

// TextDateInputViewModel contains the display and canonical values for a date field.
type TextDateInputViewModel struct {
	ID             string
	Name           string
	Label          string
	Value          string
	CanonicalValue string
	Placeholder    string
	Required       bool
	HelpID         string
	ErrorID        string
	Error          string
	Picker         bool
}

type OptionsProvider func(context.Context) ([]SearchableSelectOption, error)
