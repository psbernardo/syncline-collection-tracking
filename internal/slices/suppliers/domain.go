package suppliers

import (
	"errors"
	"fmt"
	"github.com/psbernardo/syncline-collection-tracking/internal/shared/money"
	"strings"
	"time"
)

var (
	ErrSupplierNotFound         = errors.New("supplier not found")
	ErrProductNotFound          = errors.New("active product not found")
	ErrDuplicateSupplierProduct = errors.New("supplier product already exists")
	ErrSupplierConflict         = errors.New("supplier was changed by another request")
	ErrSupplierProductNotFound  = errors.New("supplier product not found")
)

type Supplier struct {
	ID                                                                                        int64
	Name, ContactPerson, ContactNumber, Email, BillingAddress, DeliveryAddress, TaxIdentifier string
	IsActive                                                                                  bool
	CreatedAtUTC, UpdatedAtUTC                                                                time.Time
	RowVersion                                                                                []byte
}
type SupplierProduct struct {
	ID, SupplierID, ProductID                          int64
	SupplierName, ProductSKU, ProductName, SupplierSKU string
	ReferenceCost                                      money.Amount
	IsActive                                           bool
	UpdatedAtUTC                                       time.Time
	RowVersion                                         []byte
}
type ValidationErrors map[string]string

func (e ValidationErrors) Error() string { return "supplier validation failed" }
func NewSupplier(in Supplier) (Supplier, error) {
	s := in
	s.Name = strings.TrimSpace(s.Name)
	s.ContactPerson = strings.TrimSpace(s.ContactPerson)
	s.ContactNumber = strings.TrimSpace(s.ContactNumber)
	s.Email = strings.TrimSpace(s.Email)
	s.BillingAddress = strings.TrimSpace(s.BillingAddress)
	s.DeliveryAddress = strings.TrimSpace(s.DeliveryAddress)
	s.TaxIdentifier = strings.TrimSpace(s.TaxIdentifier)
	e := ValidationErrors{}
	if s.Name == "" {
		e["Name"] = "This field is required."
	}
	if len([]rune(s.Name)) > 255 {
		e["Name"] = fmt.Sprintf("Must be %d characters or fewer.", 255)
	}
	if len(e) > 0 {
		return Supplier{}, e
	}
	return s, nil
}
func NewSupplierProduct(in SupplierProduct) (SupplierProduct, error) {
	if in.SupplierID < 1 {
		return SupplierProduct{}, ValidationErrors{"SupplierID": "Supplier is required."}
	}
	if in.ProductID < 1 {
		return SupplierProduct{}, ValidationErrors{"ProductID": "Product is required."}
	}
	if in.ReferenceCost < 0 {
		return SupplierProduct{}, ValidationErrors{"ReferenceCost": "Cost cannot be negative."}
	}
	in.SupplierSKU = strings.TrimSpace(in.SupplierSKU)
	return in, nil
}
