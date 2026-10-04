package suppliers

import "testing"

func TestNewHandlerParsesTemplates(t *testing.T) {
	if _, err := NewHandler(nil); err != nil {
		t.Fatal(err)
	}
}

func TestParseIDRejectsBlankAndNonPositiveValues(t *testing.T) {
	for _, value := range []string{"", "abc", "0", "-1"} {
		if id, err := parseID(value, "SupplierID"); id != 0 || err["SupplierID"] == "" {
			t.Fatalf("parseID(%q) = %d, %v", value, id, err)
		}
	}
}

func TestParseIDTrimsPositiveValue(t *testing.T) {
	id, err := parseID(" 42 ", "ProductID")
	if err != nil || id != 42 {
		t.Fatalf("parseID returned %d, %v", id, err)
	}
}

func TestProductSelectUsesHumanReadableLabel(t *testing.T) {
	selectModel := productSelect([]ProductOption{{ID: 7, SKU: "P-001", Name: "Bond paper", UOM: "BOX", Description: "Long product description"}}, 7)
	if selectModel.Options[0].Label != "P-001 - Bond paper (BOX)" || selectModel.Selected != "7" {
		t.Fatalf("unexpected select model: %+v", selectModel)
	}
	if selectModel.Options[0].Description != "Long product description" || !selectModel.ShowSelectedDetails {
		t.Fatalf("product description is not enabled for the selected summary: %+v", selectModel)
	}
}
