package suppliers

import (
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"testing"
)

func TestNewSupplierTrimsRequiredName(t *testing.T) {
	s, err := NewSupplier(Supplier{Name: "  Acme Supply  "})
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Acme Supply" {
		t.Fatalf("name was not trimmed: %q", s.Name)
	}
}

func TestNewSupplierProductRejectsNegativeCost(t *testing.T) {
	_, err := NewSupplierProduct(SupplierProduct{SupplierID: 1, ProductID: 2, ReferenceCost: money.Amount(-1)})
	v, ok := err.(ValidationErrors)
	if !ok || v["ReferenceCost"] == "" {
		t.Fatalf("expected cost validation, got %v", err)
	}
}

func TestNormalizeProductIDsRemovesInvalidAndDuplicateValues(t *testing.T) {
	got := normalizeProductIDs([]int64{0, 4, 4, -2, 9, 4})
	if len(got) != 2 || got[0] != 4 || got[1] != 9 {
		t.Fatalf("unexpected normalized IDs: %v", got)
	}
}
