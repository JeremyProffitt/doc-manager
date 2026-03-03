package services

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"github.com/jung-kurt/gofpdf/v2"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

// PDFService generates populated PDF documents from form images and customer data.
type PDFService struct{}

// NewPDFService creates a new PDFService.
func NewPDFService() *PDFService {
	return &PDFService{}
}

// GeneratePopulatedPDF creates a PDF with customer data overlaid on the form image.
// formImageData: the original form file bytes (PNG or JPEG).
// contentType: MIME type of the form image (image/png, image/jpeg).
// placement: the field placement data with percentage-based coordinates. May be nil.
// customer: the customer data to fill in.
func (s *PDFService) GeneratePopulatedPDF(
	formImageData []byte,
	contentType string,
	placement *models.FieldPlacement,
	customer *models.Customer,
) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetAutoPageBreak(false, 0)

	if contentType != "application/pdf" {
		pdf.AddPage()

		// Determine image type for gofpdf registration.
		imgType := "PNG"
		if contentType == "image/jpeg" || contentType == "image/jpg" {
			imgType = "JPG"
		}

		// Decode image config for aspect ratio calculation.
		reader := bytes.NewReader(formImageData)
		imgConfig, _, err := image.DecodeConfig(reader)

		// Register the image from bytes.
		pdf.RegisterImageOptionsReader(
			"form",
			gofpdf.ImageOptions{ImageType: imgType, ReadDpi: true},
			bytes.NewReader(formImageData),
		)

		pageW, pageH := pdf.GetPageSize()

		// Place image to fill page while preserving aspect ratio.
		if err == nil && imgConfig.Width > 0 && imgConfig.Height > 0 {
			imgW := float64(imgConfig.Width)
			imgH := float64(imgConfig.Height)
			scale := pageW / imgW
			if imgH*scale > pageH {
				scale = pageH / imgH
			}
			pdf.ImageOptions("form", 0, 0, imgW*scale, imgH*scale, false, gofpdf.ImageOptions{}, 0, "")
		} else {
			// Fallback: stretch to fill page.
			pdf.ImageOptions("form", 0, 0, pageW, pageH, false, gofpdf.ImageOptions{}, 0, "")
		}

		// Overlay customer data at field positions.
		if placement != nil {
			for _, field := range placement.Fields {
				value := getCustomerFieldValue(customer, field.FieldName)
				if value == "" {
					continue
				}

				// Determine font settings: field-level overrides take priority.
				fontFamily := placement.FontFamily
				fontSize := placement.FontSize
				if field.FontFamily != nil && *field.FontFamily != "" {
					fontFamily = *field.FontFamily
				}
				if field.FontSize != nil && *field.FontSize > 0 {
					fontSize = *field.FontSize
				}

				// Convert percentage coordinates to page coordinates.
				x := field.X / 100.0 * pageW
				y := field.Y / 100.0 * pageH

				// Map font names to PDF standard fonts.
				pdfFont := mapFontFamily(fontFamily)
				pdf.SetFont(pdfFont, "", float64(fontSize))

				// Offset Y slightly for baseline alignment.
				pdf.Text(x, y+float64(fontSize)*0.35, value)
			}
		}
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("generating PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// getCustomerFieldValue maps a field name to the corresponding customer data value.
func getCustomerFieldValue(customer *models.Customer, fieldName string) string {
	switch fieldName {
	case "Name":
		return customer.Name
	case "Business":
		return customer.Business
	case "Address":
		return customer.Address
	case "City":
		return customer.City
	case "State":
		return customer.State
	case "Zip":
		return customer.Zip
	case "Phone Number", "Phone":
		return customer.Phone
	default:
		return ""
	}
}

// mapFontFamily maps user-facing font names to gofpdf built-in font names.
func mapFontFamily(name string) string {
	switch name {
	case "Courier":
		return "Courier"
	case "Times-Roman", "Times":
		return "Times"
	default:
		return "Helvetica"
	}
}
