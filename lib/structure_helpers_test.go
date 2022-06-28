package lib_test

import (
	"testing"

	"git.sr.ht/~rjarry/aerc/lib"
	"git.sr.ht/~rjarry/aerc/models"
)

func TestLib_FindAllNonMultipart(t *testing.T) {

	testStructure := &models.BodyStructure{
		MIMEType: "multipart",
		Parts: []*models.BodyStructure{
			&models.BodyStructure{},
			&models.BodyStructure{
				MIMEType: "multipart",
				Parts: []*models.BodyStructure{
					&models.BodyStructure{},
					&models.BodyStructure{},
				},
			},
			&models.BodyStructure{},
		},
	}

	expected := [][]int{
		[]int{1},
		[]int{2, 1},
		[]int{2, 2},
		[]int{3},
	}

	parts := lib.FindAllNonMultipart(testStructure, nil, nil)

	if len(expected) != len(parts) {
		t.Errorf("incorrect dimensions; expected: %v, got: %v", expected, parts)
	}

	for i := 0; i < len(parts); i++ {
		if !lib.EqualParts(expected[i], parts[i]) {
			t.Errorf("incorrect values; expected: %v, got: %v", expected[i], parts[i])
		}
	}

}
