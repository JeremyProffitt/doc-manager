package store

import (
	"testing"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

func TestFormStore(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, fs *MockFormStore)
	}{
		{
			name: "CreateForm and GetForm returns the form",
			run: func(t *testing.T, fs *MockFormStore) {
				form := &models.Form{
					ID:     "form-1",
					UserID: "user-1",
					Name:   "Test Form",
					Status: "uploaded",
					S3Key:  "forms/form-1/test.pdf",
				}
				if err := fs.CreateForm(form); err != nil {
					t.Fatalf("CreateForm failed: %v", err)
				}
				got, err := fs.GetForm("form-1")
				if err != nil {
					t.Fatalf("GetForm failed: %v", err)
				}
				if got == nil {
					t.Fatal("expected form, got nil")
				}
				if got.ID != "form-1" {
					t.Errorf("expected ID form-1, got %s", got.ID)
				}
				if got.Name != "Test Form" {
					t.Errorf("expected Name 'Test Form', got %s", got.Name)
				}
				if got.UserID != "user-1" {
					t.Errorf("expected UserID user-1, got %s", got.UserID)
				}
			},
		},
		{
			name: "ListForms returns only forms for the given userId",
			run: func(t *testing.T, fs *MockFormStore) {
				fs.CreateForm(&models.Form{ID: "f1", UserID: "user-a", Name: "Form A1"})
				fs.CreateForm(&models.Form{ID: "f2", UserID: "user-a", Name: "Form A2"})
				fs.CreateForm(&models.Form{ID: "f3", UserID: "user-b", Name: "Form B1"})

				forms, err := fs.ListForms("user-a")
				if err != nil {
					t.Fatalf("ListForms failed: %v", err)
				}
				if len(forms) != 2 {
					t.Fatalf("expected 2 forms for user-a, got %d", len(forms))
				}

				formsB, err := fs.ListForms("user-b")
				if err != nil {
					t.Fatalf("ListForms failed: %v", err)
				}
				if len(formsB) != 1 {
					t.Fatalf("expected 1 form for user-b, got %d", len(formsB))
				}
			},
		},
		{
			name: "UpdateForm changes form metadata",
			run: func(t *testing.T, fs *MockFormStore) {
				fs.CreateForm(&models.Form{ID: "f1", UserID: "u1", Name: "Old Name", Status: "uploading"})

				updated := &models.Form{ID: "f1", UserID: "u1", Name: "New Name", Status: "uploaded"}
				if err := fs.UpdateForm(updated); err != nil {
					t.Fatalf("UpdateForm failed: %v", err)
				}

				got, _ := fs.GetForm("f1")
				if got.Name != "New Name" {
					t.Errorf("expected name 'New Name', got %s", got.Name)
				}
				if got.Status != "uploaded" {
					t.Errorf("expected status 'uploaded', got %s", got.Status)
				}
			},
		},
		{
			name: "DeleteForm removes the form",
			run: func(t *testing.T, fs *MockFormStore) {
				fs.CreateForm(&models.Form{ID: "f1", UserID: "u1", Name: "To Delete"})

				if err := fs.DeleteForm("f1"); err != nil {
					t.Fatalf("DeleteForm failed: %v", err)
				}

				got, err := fs.GetForm("f1")
				if err != nil {
					t.Fatalf("GetForm after delete failed: %v", err)
				}
				if got != nil {
					t.Error("expected nil after delete, got form")
				}
			},
		},
		{
			name: "GetForm for non-existent ID returns nil",
			run: func(t *testing.T, fs *MockFormStore) {
				got, err := fs.GetForm("does-not-exist")
				if err != nil {
					t.Fatalf("GetForm failed: %v", err)
				}
				if got != nil {
					t.Error("expected nil for non-existent form")
				}
			},
		},
		{
			name: "ListForms returns empty slice when no forms exist",
			run: func(t *testing.T, fs *MockFormStore) {
				forms, err := fs.ListForms("user-x")
				if err != nil {
					t.Fatalf("ListForms failed: %v", err)
				}
				if forms == nil {
					t.Fatal("expected empty slice, got nil")
				}
				if len(forms) != 0 {
					t.Errorf("expected 0 forms, got %d", len(forms))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewMockFormStore()
			tt.run(t, fs)
		})
	}
}
