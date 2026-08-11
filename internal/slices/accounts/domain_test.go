package accounts

import "testing"

func TestNewCompanyAccountTrimsAndValidates(t *testing.T) {
	account, err := NewCompanyAccount(CompanyAccount{
		CompanyName:     "  Acme Corp  ",
		ContactPerson:   "  Jane Doe ",
		TINNumber:       " 123 ",
		BillingAddress:  " Billing ",
		DeliveryAddress: " Delivery ",
		ContactNumber:   " 555-0100 ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if account.CompanyName != "Acme Corp" || account.ContactNumber != "555-0100" {
		t.Fatalf("account was not trimmed: %+v", account)
	}
}

func TestNewCompanyAccountReportsRequiredFields(t *testing.T) {
	_, err := NewCompanyAccount(CompanyAccount{})
	validation, ok := err.(ValidationErrors)
	if !ok {
		t.Fatalf("error type = %T, want ValidationErrors", err)
	}
	for _, field := range []string{"CompanyName", "ContactPerson", "TINNumber", "BillingAddress", "DeliveryAddress", "ContactNumber"} {
		if validation[field] == "" {
			t.Errorf("missing validation error for %s", field)
		}
	}
}

func TestNewCompanyAccountRejectsTooLongValue(t *testing.T) {
	input := CompanyAccount{
		CompanyName:     "Acme",
		ContactPerson:   "Jane",
		TINNumber:       "TIN",
		BillingAddress:  "Billing",
		DeliveryAddress: "Delivery",
		ContactNumber:   "555",
	}
	for index := 0; index < maxNameLength+1; index++ {
		input.CompanyName += "x"
	}
	_, err := NewCompanyAccount(input)
	validation, ok := err.(ValidationErrors)
	if !ok || validation["CompanyName"] == "" {
		t.Fatalf("expected company name length validation, got %v", err)
	}
}
