package msgview

import "testing"

func TestGetCollisionlessFilename(t *testing.T) {
	tests := []struct {
		originalFilename string
		expectedNewName  string
		existingFiles    map[string]struct{}
	}{
		{"test", "test", map[string]struct{}{}},
		{"test", "test", map[string]struct{}{"other-file": {}}},
		{"test.txt", "test.txt", map[string]struct{}{"test.log": {}}},
		{"test.txt", "test_1.txt", map[string]struct{}{"test.txt": {}}},
		{"test.txt", "test_2.txt", map[string]struct{}{"test.txt": {}, "test_1.txt": {}}},
		{"test.txt", "test_1.txt", map[string]struct{}{"test.txt": {}, "test_2.txt": {}}},
	}
	for _, tt := range tests {
		actual := getCollisionlessFilename(tt.originalFilename, tt.existingFiles)
		if actual != tt.expectedNewName {
			t.Errorf("expected %s, actual %s", tt.expectedNewName, actual)
		}
	}
}
