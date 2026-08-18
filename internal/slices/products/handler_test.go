package products

import "testing"

func TestNewHandlerParsesTemplates(t *testing.T) {
	if _, err := NewHandler(nil); err != nil {
		t.Fatal(err)
	}
}
