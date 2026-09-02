package pdf

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/signintech/gopdf"
)

const (
	Width        = 595.0
	Height       = 842.0
	Margin       = 24.0
	PageBottom   = 818.0
	Purple       = "purple"
	LightPurple  = "light-purple"
	Gray         = "gray"
	Font         = "Document"
	FontBold     = "DocumentBold"
	FontItalic   = "DocumentItalic"
	FontBoldItal = "DocumentBoldItalic"
)

type SellerProfile struct {
	Name, Address, Email, Phone, Logo string
}

type Metadata struct{ Label, Value string }

func DefaultSeller() SellerProfile {
	return SellerProfile{
		Name:    envOr("SELLER_NAME", "Syncline Consumer Goods Trading"),
		Address: envOr("SELLER_ADDRESS", "402-E Marigold Street Lakeview Homes 1, Putatan Muntinlupa City"),
		Email:   envOr("SELLER_EMAIL", "syncline.mae@gmail.com"),
		Phone:   envOr("SELLER_PHONE", "63 927 670 7281"),
		Logo:    findLogo(),
	}
}

func FindFont() string {
	if value := strings.TrimSpace(os.Getenv("PDF_FONT_PATH")); value != "" {
		return value
	}
	candidates := []string{
		"/usr/share/fonts/truetype/montserrat/Montserrat-Regular.ttf",
		"/usr/share/fonts/truetype/msttcorefonts/Verdana.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/liberation2/LiberationSans-Regular.ttf",
	}
	if windir := os.Getenv("WINDIR"); windir != "" {
		candidates = append([]string{filepath.Join(windir, "Fonts", "verdana.ttf")}, candidates...)
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func Configure(document *gopdf.GoPdf, font string) error {
	document.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	if font == "" {
		return fmt.Errorf("PDF font is not configured")
	}
	for _, family := range []string{Font, FontBold, FontItalic, FontBoldItal} {
		if err := document.AddTTFFont(family, font); err != nil {
			return err
		}
	}
	document.SetMargins(Margin, Margin, Margin, Margin)
	document.SetCompressLevel(6)
	document.AddPage()
	return nil
}

func Family(style string) string {
	switch style {
	case "B":
		return FontBold
	case "I":
		return FontItalic
	case "BI", "IB":
		return FontBoldItal
	default:
		return Font
	}
}

func Text(document *gopdf.GoPdf, x, y, size float64, value string, width float64, align int, style, color string) {
	document.SetXY(x, y)
	document.SetTextColor(RGB(color))
	_ = document.SetFont(Family(style), "", size)
	_ = document.CellWithOption(&gopdf.Rect{W: width, H: size + 4}, value, gopdf.CellOption{Align: align})
}

func MultiText(document *gopdf.GoPdf, x, y, size float64, value string, width, height float64, align int, style, color string) {
	document.SetXY(x, y)
	document.SetTextColor(RGB(color))
	_ = document.SetFont(Family(style), "", size)
	_ = document.MultiCellWithOption(&gopdf.Rect{W: width, H: height}, value, gopdf.CellOption{Align: align})
}

func Wrap(document *gopdf.GoPdf, value string, width float64) []string {
	lines, err := document.SplitText(value, width)
	if err != nil || len(lines) == 0 {
		return []string{value}
	}
	return lines
}

func RGB(name string) (uint8, uint8, uint8) {
	switch name {
	case Purple:
		return 55, 20, 115
	case LightPurple:
		return 190, 170, 220
	case Gray:
		return 217, 217, 217
	case "white":
		return 255, 255, 255
	default:
		return 0, 0, 0
	}
}

func Line(document *gopdf.GoPdf, x1, y1, x2, y2 float64, color string) {
	document.SetStrokeColor(RGB(color))
	document.SetLineWidth(0.5)
	document.Line(x1, y1, x2, y2)
}

func FormatAmount(value interface{ FormatPHP() string }) string {
	return strings.ReplaceAll(value.FormatPHP(), "₱", "PHP ")
}

func FormatUnitPrice(value interface{ FormatPHP() string }) string {
	return strings.TrimPrefix(FormatAmount(value), "PHP ")
}

func SafeFilename(value, fallback string) string {
	var builder strings.Builder
	for _, r := range value {
		if strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_", r) {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return fallback + ".pdf"
	}
	return builder.String() + ".pdf"
}

func Write(w io.Writer, document *gopdf.GoPdf, kind string) error {
	bytes, err := document.GetBytesPdfReturnErr()
	if err != nil {
		return fmt.Errorf("build %s PDF: %w", kind, err)
	}
	if _, err := w.Write(bytes); err != nil {
		return fmt.Errorf("write %s PDF: %w", kind, err)
	}
	return nil
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func findLogo() string {
	if value := strings.TrimSpace(os.Getenv("SELLER_LOGO_PATH")); value != "" {
		if _, err := os.Stat(value); err == nil {
			return value
		}
	}
	for _, candidate := range []string{"/files/syncline-logo.png", "files/syncline-logo.png"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}
