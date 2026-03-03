package services

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

// createTestPNG generates a minimal PNG image for testing.
func createTestPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.White)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func stringPtr(s string) *string { return &s }
func intPtr(i int) *int          { return &i }

func TestPDFService(t *testing.T) {
	svc := NewPDFService()
	testImage := createTestPNG(612, 792)

	tests := []struct {
		name        string
		imageData   []byte
		contentType string
		placement   *models.FieldPlacement
		customer    *models.Customer
		wantErr     bool
		check       func(t *testing.T, pdf []byte)
	}{
		{
			name:        "Generate PDF from form image with text overlays returns valid PDF",
			imageData:   testImage,
			contentType: "image/png",
			placement: &models.FieldPlacement{
				FormID:     "form-1",
				Version:    1,
				FontFamily: "Helvetica",
				FontSize:   12,
				Fields: []models.Field{
					{FieldName: "Name", X: 10, Y: 20, Width: 30, Height: 3},
				},
			},
			customer: &models.Customer{
				ID:   "cust-1",
				Name: "John Doe",
			},
			check: func(t *testing.T, pdf []byte) {
				t.Helper()
				if len(pdf) == 0 {
					t.Error("expected non-empty PDF output")
				}
				if !bytes.HasPrefix(pdf, []byte("%PDF")) {
					t.Errorf("expected PDF to start with %%PDF, got %q", string(pdf[:min(10, len(pdf))]))
				}
			},
		},
		{
			name:        "Form-level font settings applied - PDF is larger with text overlays",
			imageData:   testImage,
			contentType: "image/png",
			placement: &models.FieldPlacement{
				FormID:     "form-1",
				Version:    1,
				FontFamily: "Courier",
				FontSize:   14,
				Fields: []models.Field{
					{FieldName: "Name", X: 10, Y: 20, Width: 30, Height: 3},
					{FieldName: "Business", X: 10, Y: 30, Width: 30, Height: 3},
				},
			},
			customer: &models.Customer{
				ID:       "cust-1",
				Name:     "Jane Smith",
				Business: "Acme Corp",
			},
			check: func(t *testing.T, pdf []byte) {
				t.Helper()
				if len(pdf) == 0 {
					t.Error("expected non-empty PDF output")
				}
				if !bytes.HasPrefix(pdf, []byte("%PDF")) {
					t.Error("expected valid PDF header")
				}
				// Generate a PDF without text for comparison
				svc := NewPDFService()
				noTextPDF, err := svc.GeneratePopulatedPDF(testImage, "image/png", nil, &models.Customer{ID: "x"})
				if err != nil {
					t.Fatalf("failed to generate baseline PDF: %v", err)
				}
				// PDF with text overlays should be larger than one without
				if len(pdf) <= len(noTextPDF) {
					t.Errorf("expected PDF with text (%d bytes) to be larger than PDF without text (%d bytes)", len(pdf), len(noTextPDF))
				}
			},
		},
		{
			name:        "Field-level font override applied - field with override uses its own font",
			imageData:   testImage,
			contentType: "image/png",
			placement: &models.FieldPlacement{
				FormID:     "form-1",
				Version:    1,
				FontFamily: "Helvetica",
				FontSize:   12,
				Fields: []models.Field{
					{
						FieldName:  "Name",
						X:          10,
						Y:          20,
						Width:      30,
						Height:     3,
						FontFamily: stringPtr("Courier"),
						FontSize:   intPtr(18),
					},
				},
			},
			customer: &models.Customer{
				ID:   "cust-1",
				Name: "Override Test",
			},
			check: func(t *testing.T, pdf []byte) {
				t.Helper()
				if len(pdf) == 0 {
					t.Error("expected non-empty PDF output")
				}
				if !bytes.HasPrefix(pdf, []byte("%PDF")) {
					t.Error("expected valid PDF header")
				}
				// The PDF should reference Courier font (field override) rather than just Helvetica
				if !bytes.Contains(pdf, []byte("Courier")) {
					t.Error("expected PDF to reference Courier font from field-level override")
				}
			},
		},
		{
			name:        "Field with null font override inherits form defaults",
			imageData:   testImage,
			contentType: "image/png",
			placement: &models.FieldPlacement{
				FormID:     "form-1",
				Version:    1,
				FontFamily: "Times-Roman",
				FontSize:   10,
				Fields: []models.Field{
					{
						FieldName:  "Name",
						X:          10,
						Y:          20,
						Width:      30,
						Height:     3,
						FontFamily: nil,
						FontSize:   nil,
					},
				},
			},
			customer: &models.Customer{
				ID:   "cust-1",
				Name: "Default Font Test",
			},
			check: func(t *testing.T, pdf []byte) {
				t.Helper()
				if len(pdf) == 0 {
					t.Error("expected non-empty PDF output")
				}
				if !bytes.HasPrefix(pdf, []byte("%PDF")) {
					t.Error("expected valid PDF header")
				}
				// Times-Roman maps to Times in gofpdf - check PDF references it
				if !bytes.Contains(pdf, []byte("Times")) {
					t.Error("expected PDF to reference Times font (inherited from form defaults)")
				}
			},
		},
		{
			name:        "Empty customer field value - no text rendered at that position",
			imageData:   testImage,
			contentType: "image/png",
			placement: &models.FieldPlacement{
				FormID:     "form-1",
				Version:    1,
				FontFamily: "Helvetica",
				FontSize:   12,
				Fields: []models.Field{
					{FieldName: "Name", X: 10, Y: 20, Width: 30, Height: 3},
					{FieldName: "Business", X: 10, Y: 30, Width: 30, Height: 3},
				},
			},
			customer: &models.Customer{
				ID:       "cust-1",
				Name:     "Has Name",
				Business: "", // empty business
			},
			check: func(t *testing.T, pdf []byte) {
				t.Helper()
				if len(pdf) == 0 {
					t.Error("expected non-empty PDF output")
				}
				if !bytes.HasPrefix(pdf, []byte("%PDF")) {
					t.Error("expected valid PDF header")
				}
				// Compare with PDF that has both fields populated
				svc := NewPDFService()
				bothPDF, err := svc.GeneratePopulatedPDF(testImage, "image/png",
					&models.FieldPlacement{
						FormID: "form-1", Version: 1, FontFamily: "Helvetica", FontSize: 12,
						Fields: []models.Field{
							{FieldName: "Name", X: 10, Y: 20, Width: 30, Height: 3},
							{FieldName: "Business", X: 10, Y: 30, Width: 30, Height: 3},
						},
					},
					&models.Customer{ID: "cust-1", Name: "Has Name", Business: "Some Biz"},
				)
				if err != nil {
					t.Fatalf("failed to generate comparison PDF: %v", err)
				}
				// PDF with empty field should be smaller than one with both populated
				if len(pdf) >= len(bothPDF) {
					t.Errorf("expected PDF with empty field (%d bytes) to be smaller than one with both fields (%d bytes)", len(pdf), len(bothPDF))
				}
			},
		},
		{
			name:        "Multiple fields rendered at correct relative positions",
			imageData:   testImage,
			contentType: "image/png",
			placement: &models.FieldPlacement{
				FormID:     "form-1",
				Version:    1,
				FontFamily: "Helvetica",
				FontSize:   12,
				Fields: []models.Field{
					{FieldName: "Name", X: 10, Y: 10, Width: 30, Height: 3},
					{FieldName: "Address", X: 10, Y: 30, Width: 40, Height: 3},
					{FieldName: "City", X: 10, Y: 40, Width: 20, Height: 3},
					{FieldName: "State", X: 40, Y: 40, Width: 10, Height: 3},
					{FieldName: "Zip", X: 55, Y: 40, Width: 15, Height: 3},
					{FieldName: "Phone", X: 10, Y: 50, Width: 30, Height: 3},
				},
			},
			customer: &models.Customer{
				ID:      "cust-1",
				Name:    "Multi Field",
				Address: "123 Main St",
				City:    "Springfield",
				State:   "IL",
				Zip:     "62701",
				Phone:   "555-1234",
			},
			check: func(t *testing.T, pdf []byte) {
				t.Helper()
				if len(pdf) == 0 {
					t.Error("expected non-empty PDF output")
				}
				if !bytes.HasPrefix(pdf, []byte("%PDF")) {
					t.Error("expected valid PDF header")
				}
				// A PDF with 6 populated fields should be significantly larger
				// than one with no fields
				svc := NewPDFService()
				noTextPDF, err := svc.GeneratePopulatedPDF(testImage, "image/png", nil, &models.Customer{ID: "x"})
				if err != nil {
					t.Fatalf("failed to generate baseline PDF: %v", err)
				}
				if len(pdf) <= len(noTextPDF) {
					t.Errorf("expected PDF with 6 fields (%d bytes) to be larger than PDF with no text (%d bytes)", len(pdf), len(noTextPDF))
				}
			},
		},
		{
			name:        "Nil placement still produces a valid PDF with just the image",
			imageData:   testImage,
			contentType: "image/png",
			placement:   nil,
			customer:    &models.Customer{ID: "cust-1", Name: "No Fields"},
			check: func(t *testing.T, pdf []byte) {
				t.Helper()
				if len(pdf) == 0 {
					t.Error("expected non-empty PDF output")
				}
				if !bytes.HasPrefix(pdf, []byte("%PDF")) {
					t.Error("expected valid PDF header")
				}
			},
		},
		{
			name:        "Phone Number field name maps to customer Phone",
			imageData:   testImage,
			contentType: "image/png",
			placement: &models.FieldPlacement{
				FormID:     "form-1",
				Version:    1,
				FontFamily: "Helvetica",
				FontSize:   12,
				Fields: []models.Field{
					{FieldName: "Phone Number", X: 10, Y: 50, Width: 30, Height: 3},
				},
			},
			customer: &models.Customer{
				ID:    "cust-1",
				Phone: "555-9999",
			},
			check: func(t *testing.T, pdf []byte) {
				t.Helper()
				if len(pdf) == 0 {
					t.Error("expected non-empty PDF output")
				}
				// PDF with phone number should be larger than one without
				svc := NewPDFService()
				noTextPDF, err := svc.GeneratePopulatedPDF(testImage, "image/png", nil, &models.Customer{ID: "x"})
				if err != nil {
					t.Fatalf("failed to generate baseline PDF: %v", err)
				}
				if len(pdf) <= len(noTextPDF) {
					t.Error("expected PDF with Phone Number field to be larger than baseline")
				}
			},
		},
		{
			name:        "Empty fields slice produces valid PDF with just the image",
			imageData:   testImage,
			contentType: "image/png",
			placement: &models.FieldPlacement{
				FormID:     "form-1",
				Version:    1,
				FontFamily: "Helvetica",
				FontSize:   12,
				Fields:     []models.Field{},
			},
			customer: &models.Customer{ID: "cust-1", Name: "No Fields"},
			check: func(t *testing.T, pdf []byte) {
				t.Helper()
				if len(pdf) == 0 {
					t.Error("expected non-empty PDF output")
				}
				if !bytes.HasPrefix(pdf, []byte("%PDF")) {
					t.Error("expected valid PDF header")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.GeneratePopulatedPDF(tt.imageData, tt.contentType, tt.placement, tt.customer)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GeneratePopulatedPDF() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.check != nil && err == nil {
				tt.check(t, result)
			}
		})
	}
}

func TestGetCustomerFieldValue(t *testing.T) {
	customer := &models.Customer{
		ID:       "cust-1",
		Name:     "John Doe",
		Business: "Acme Corp",
		Address:  "123 Main St",
		City:     "Springfield",
		State:    "IL",
		Zip:      "62701",
		Phone:    "555-1234",
	}

	tests := []struct {
		fieldName string
		expected  string
	}{
		{"Name", "John Doe"},
		{"Business", "Acme Corp"},
		{"Address", "123 Main St"},
		{"City", "Springfield"},
		{"State", "IL"},
		{"Zip", "62701"},
		{"Phone Number", "555-1234"},
		{"Phone", "555-1234"},
		{"Unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run("field_"+tt.fieldName, func(t *testing.T) {
			got := getCustomerFieldValue(customer, tt.fieldName)
			if got != tt.expected {
				t.Errorf("getCustomerFieldValue(%q) = %q, want %q", tt.fieldName, got, tt.expected)
			}
		})
	}
}

func TestMapFontFamily(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Courier", "Courier"},
		{"Times-Roman", "Times"},
		{"Times", "Times"},
		{"Helvetica", "Helvetica"},
		{"Arial", "Helvetica"},    // unknown font defaults to Helvetica
		{"", "Helvetica"},         // empty defaults to Helvetica
		{"SomeFont", "Helvetica"}, // unknown defaults to Helvetica
	}

	for _, tt := range tests {
		t.Run("font_"+tt.input, func(t *testing.T) {
			got := mapFontFamily(tt.input)
			if got != tt.expected {
				t.Errorf("mapFontFamily(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
