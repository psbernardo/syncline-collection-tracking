package products

import "testing"

func TestNewProductNormalizesAndValidates(t *testing.T) {
	p, err := New(Product{SKU: " lap-001 ", Name: " Laptop ", UOM: "pc"})
	if err != nil {
		t.Fatal(err)
	}
	if p.SKU != "LAP-001" || p.Name != "Laptop" || p.UOM != "PC" {
		t.Fatalf("unexpected product: %+v", p)
	}
}

func TestNewProductRejectsUnknownUOM(t *testing.T) {
	_, err := New(Product{SKU: "A", Name: "Item", UOM: "KG"})
	v, ok := err.(ValidationErrors)
	if !ok || v["UOM"] == "" {
		t.Fatalf("expected UOM validation, got %v", err)
	}
}
