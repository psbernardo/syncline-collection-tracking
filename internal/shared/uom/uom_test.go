package uom

import "testing"

func TestNormalizeAndValidate(t *testing.T) {
	if got := Normalize(" box "); got != BOX {
		t.Fatalf("got %q", got)
	}
	for _, value := range Options() {
		if !Valid(value) {
			t.Fatalf("supported UOM was rejected: %q", value)
		}
	}
	if Valid(Code("SET")) || Valid(Code("KG")) {
		t.Fatal("unsupported UOM was accepted")
	}
}

func TestOptionsOrder(t *testing.T) {
	want := []Code{PC, CASE, PACK, BTL, REAM, ROLL, BOX}
	got := Options()
	if len(got) != len(want) {
		t.Fatalf("got %d options, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("option %d: got %q, want %q", index, got[index], want[index])
		}
	}
}
