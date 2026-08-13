package webtemplates

// MultiSelectOption describes one option in a reusable multi-select control.
type MultiSelectOption struct {
	Value    string
	Label    string
	Disabled bool
}

// MultiSelectViewModel contains the server-owned state needed to render a
// reusable multi-select control.
type MultiSelectViewModel struct {
	ID          string
	Name        string
	Label       string
	Placeholder string
	Options     []MultiSelectOption
	Selected    []string
}
