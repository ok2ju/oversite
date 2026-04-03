package demo

import (
	"testing"
)

func TestValidateExtension(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  error
	}{
		{"valid .dem", "match.dem", nil},
		{"valid uppercase .DEM", "match.DEM", nil},
		{"invalid .txt", "match.txt", ErrInvalidExtension},
		{"invalid .dem.bak", "match.dem.bak", ErrInvalidExtension},
		{"empty string", "", ErrInvalidExtension},
		{"no extension", "match", ErrInvalidExtension},
		{"dot only", ".dem", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExtension(tt.filename)
			if err != tt.wantErr {
				t.Errorf("ValidateExtension(%q) = %v, want %v", tt.filename, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		wantErr error
	}{
		{"zero bytes", 0, nil},
		{"1 byte", 1, nil},
		{"499 MB", 499 << 20, nil},
		{"exactly 500 MB", MaxUploadSize, nil},
		{"500 MB + 1", MaxUploadSize + 1, ErrFileTooLarge},
		{"1 GB", 1 << 30, ErrFileTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSize(tt.size)
			if err != tt.wantErr {
				t.Errorf("ValidateSize(%d) = %v, want %v", tt.size, err, tt.wantErr)
			}
		})
	}
}

func TestValidateMagicBytes(t *testing.T) {
	tests := []struct {
		name    string
		header  []byte
		wantErr error
	}{
		{"CS2 magic bytes", MagicCS2, nil},
		{"CS:GO magic bytes", MagicCSGO, nil},
		{"CS2 with trailing data", append([]byte("PBDEMS2\x00"), 0xFF, 0xAB), nil},
		{"random bytes", []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}, ErrInvalidMagicBytes},
		{"too short", []byte{0x01, 0x02}, ErrInvalidMagicBytes},
		{"empty", []byte{}, ErrInvalidMagicBytes},
		{"nil", nil, ErrInvalidMagicBytes},
		{"almost CS2 magic", []byte("PBDEMS2\x01"), ErrInvalidMagicBytes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMagicBytes(tt.header)
			if err != tt.wantErr {
				t.Errorf("ValidateMagicBytes(%v) = %v, want %v", tt.header, err, tt.wantErr)
			}
		})
	}
}
