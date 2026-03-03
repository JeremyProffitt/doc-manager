package store

import (
	"testing"

	"github.com/JeremyProffitt/doc-manager/internal/models"
)

func TestFieldStore(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, fs *MockFieldStore)
	}{
		{
			name: "SaveFieldPlacement returns version 1 for first save",
			run: func(t *testing.T, fs *MockFieldStore) {
				placement := &models.FieldPlacement{
					FormID:    "form-1",
					Source:    "ai_analysis",
					CreatedAt: "2024-01-01T00:00:00Z",
					Fields: []models.Field{
						{FieldName: "Name", Page: 1, X: 100, Y: 200},
					},
				}
				version, err := fs.SaveFieldPlacement(placement)
				if err != nil {
					t.Fatalf("SaveFieldPlacement failed: %v", err)
				}
				if version != 1 {
					t.Errorf("expected version 1, got %d", version)
				}
			},
		},
		{
			name: "SaveFieldPlacement auto-increments version",
			run: func(t *testing.T, fs *MockFieldStore) {
				p1 := &models.FieldPlacement{FormID: "form-1", Source: "ai_analysis"}
				v1, _ := fs.SaveFieldPlacement(p1)

				p2 := &models.FieldPlacement{FormID: "form-1", Source: "manual_edit"}
				v2, _ := fs.SaveFieldPlacement(p2)

				if v1 != 1 {
					t.Errorf("expected first version 1, got %d", v1)
				}
				if v2 != 2 {
					t.Errorf("expected second version 2, got %d", v2)
				}
			},
		},
		{
			name: "GetLatestFieldPlacement returns highest version",
			run: func(t *testing.T, fs *MockFieldStore) {
				fs.SaveFieldPlacement(&models.FieldPlacement{FormID: "form-1", Source: "v1"})
				fs.SaveFieldPlacement(&models.FieldPlacement{FormID: "form-1", Source: "v2"})
				fs.SaveFieldPlacement(&models.FieldPlacement{FormID: "form-1", Source: "v3"})

				latest, err := fs.GetLatestFieldPlacement("form-1")
				if err != nil {
					t.Fatalf("GetLatestFieldPlacement failed: %v", err)
				}
				if latest == nil {
					t.Fatal("expected placement, got nil")
				}
				if latest.Version != 3 {
					t.Errorf("expected version 3, got %d", latest.Version)
				}
				if latest.Source != "v3" {
					t.Errorf("expected source 'v3', got %s", latest.Source)
				}
			},
		},
		{
			name: "GetFieldPlacement with specific version returns that version",
			run: func(t *testing.T, fs *MockFieldStore) {
				fs.SaveFieldPlacement(&models.FieldPlacement{FormID: "form-1", Source: "first"})
				fs.SaveFieldPlacement(&models.FieldPlacement{FormID: "form-1", Source: "second"})

				got, err := fs.GetFieldPlacement("form-1", 1)
				if err != nil {
					t.Fatalf("GetFieldPlacement failed: %v", err)
				}
				if got == nil {
					t.Fatal("expected placement, got nil")
				}
				if got.Version != 1 {
					t.Errorf("expected version 1, got %d", got.Version)
				}
				if got.Source != "first" {
					t.Errorf("expected source 'first', got %s", got.Source)
				}
			},
		},
		{
			name: "GetFieldPlacement for non-existent version returns nil",
			run: func(t *testing.T, fs *MockFieldStore) {
				got, err := fs.GetFieldPlacement("form-1", 99)
				if err != nil {
					t.Fatalf("GetFieldPlacement failed: %v", err)
				}
				if got != nil {
					t.Error("expected nil for non-existent version")
				}
			},
		},
		{
			name: "ListVersions returns metadata sorted by version desc",
			run: func(t *testing.T, fs *MockFieldStore) {
				fs.SaveFieldPlacement(&models.FieldPlacement{
					FormID: "form-1",
					Source: "v1",
					Fields: []models.Field{{FieldName: "Name"}},
				})
				fs.SaveFieldPlacement(&models.FieldPlacement{
					FormID: "form-1",
					Source: "v2",
					Fields: []models.Field{{FieldName: "Address"}},
				})

				versions, err := fs.ListVersions("form-1")
				if err != nil {
					t.Fatalf("ListVersions failed: %v", err)
				}
				if len(versions) != 2 {
					t.Fatalf("expected 2 versions, got %d", len(versions))
				}

				// Should be sorted desc by version
				if versions[0].Version != 2 {
					t.Errorf("expected first version to be 2, got %d", versions[0].Version)
				}
				if versions[1].Version != 1 {
					t.Errorf("expected second version to be 1, got %d", versions[1].Version)
				}

				// Fields should be stripped (metadata only)
				for _, v := range versions {
					if len(v.Fields) != 0 {
						t.Errorf("expected Fields to be empty for metadata, got %d fields", len(v.Fields))
					}
				}
			},
		},
		{
			name: "DeleteAllVersions removes all versions",
			run: func(t *testing.T, fs *MockFieldStore) {
				fs.SaveFieldPlacement(&models.FieldPlacement{FormID: "form-1", Source: "v1"})
				fs.SaveFieldPlacement(&models.FieldPlacement{FormID: "form-1", Source: "v2"})

				if err := fs.DeleteAllVersions("form-1"); err != nil {
					t.Fatalf("DeleteAllVersions failed: %v", err)
				}

				got, _ := fs.GetLatestFieldPlacement("form-1")
				if got != nil {
					t.Error("expected nil after DeleteAllVersions")
				}

				versions, _ := fs.ListVersions("form-1")
				if len(versions) != 0 {
					t.Errorf("expected 0 versions after delete, got %d", len(versions))
				}
			},
		},
		{
			name: "GetLatestFieldPlacement returns nil when no versions exist",
			run: func(t *testing.T, fs *MockFieldStore) {
				got, err := fs.GetLatestFieldPlacement("no-such-form")
				if err != nil {
					t.Fatalf("GetLatestFieldPlacement failed: %v", err)
				}
				if got != nil {
					t.Error("expected nil for form with no versions")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := NewMockFieldStore()
			tt.run(t, fs)
		})
	}
}
