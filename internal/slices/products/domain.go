package products

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/psbernardo/syncline-collection-tracking/internal/shared/uom"
)

var (
	ErrNotFound  = errors.New("product not found")
	ErrConflict  = errors.New("product was changed by another request")
	ErrUOMLocked = errors.New("product UOM cannot change after transaction use")
)

type Product struct {
	ID                          int64
	SKU, Name, Description, UOM string
	IsActive                    bool
	CreatedAtUTC, UpdatedAtUTC  time.Time
	RowVersion                  []byte
}

type ValidationErrors map[string]string

func (e ValidationErrors) Error() string { return "product validation failed" }

func New(input Product) (Product, error) {
	p := input
	p.SKU = strings.ToUpper(strings.TrimSpace(p.SKU))
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.UOM = string(uom.Normalize(p.UOM))
	e := ValidationErrors{}
	required(e, "SKU", p.SKU, 50)
	required(e, "Name", p.Name, 255)
	required(e, "UOM", p.UOM, 20)
	if !uom.Valid(uom.Code(p.UOM)) {
		e["UOM"] = "Select a supported UOM."
	}
	if len(e) > 0 {
		return Product{}, e
	}
	return p, nil
}

func required(e ValidationErrors, field, value string, max int) {
	if value == "" {
		e[field] = "This field is required."
		return
	}
	if len([]rune(value)) > max {
		e[field] = fmt.Sprintf("Must be %d characters or fewer.", max)
	}
}
