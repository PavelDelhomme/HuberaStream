package metadata

import (
	"testing"
)

func TestExtractTitleFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		expected []string
	}{
		{
			"RED_ONE.webm",
			[]string{"Red One"}, // Au moins "Red One" doit être présent
		},
		{
			"The_Matrix_1999.mp4",
			[]string{"The Matrix 1999", "The Matrix"},
		},
		{
			"spider-man-2.mkv",
			[]string{"Spider Man 2", "Spider-Man 2"},
		},
		{
			"test-video.avi",
			[]string{"Test Video"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			variants := ExtractTitleFromFilename(tt.filename)
			if len(variants) == 0 {
				t.Errorf("ExtractTitleFromFilename(%q) retourne un tableau vide", tt.filename)
			}

			// Vérifier qu'au moins un des variants attendus est présent
			found := false
			for _, expected := range tt.expected {
				for _, variant := range variants {
					if variant == expected {
						found = true
						break
					}
				}
				if found {
					break
				}
			}

			if !found && len(tt.expected) > 0 {
				t.Errorf("ExtractTitleFromFilename(%q) = %v, ne contient aucun des variants attendus: %v", tt.filename, variants, tt.expected)
			}
		})
	}
}

func TestExtractTitleFromFilename_Empty(t *testing.T) {
	result := ExtractTitleFromFilename("")
	if len(result) != 0 {
		t.Errorf("ExtractTitleFromFilename(\"\") devrait retourner un tableau vide, a retourné: %v", result)
	}
}

func TestExtractTitleFromFilename_OnlyExtension(t *testing.T) {
	result := ExtractTitleFromFilename(".mp4")
	if len(result) != 0 {
		t.Errorf("ExtractTitleFromFilename(\".mp4\") devrait retourner un tableau vide, a retourné: %v", result)
	}
}
