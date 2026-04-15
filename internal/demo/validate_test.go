package demo

import (
	"errors"
	"testing"
)

func TestValidateExtension(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  error
	}{
		{
			name:     "valid .dem extension",
			filename: "match.dem",
			wantErr:  nil,
		},
		{
			name:     "valid .DEM uppercase",
			filename: "match.DEM",
			wantErr:  nil,
		},
		{
			name:     "valid .Dem mixed case",
			filename: "match.Dem",
			wantErr:  nil,
		},
		{
			name:     "invalid .txt extension",
			filename: "match.txt",
			wantErr:  ErrInvalidExtension,
		},
		{
			name:     "no extension",
			filename: "match",
			wantErr:  ErrInvalidExtension,
		},
		{
			name:     "empty string",
			filename: "",
			wantErr:  ErrInvalidExtension,
		},
		{
			name:     "full path with .dem",
			filename: "/home/user/demos/match.dem",
			wantErr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExtension(tt.filename)
			if !errors.Is(err, tt.wantErr) {
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
		{
			name:    "zero bytes",
			size:    0,
			wantErr: nil,
		},
		{
			name:    "small file",
			size:    1024,
			wantErr: nil,
		},
		{
			name:    "exactly max upload size",
			size:    MaxUploadSize,
			wantErr: nil,
		},
		{
			name:    "one byte over max",
			size:    MaxUploadSize + 1,
			wantErr: ErrFileTooLarge,
		},
		{
			name:    "way over max",
			size:    MaxUploadSize * 2,
			wantErr: ErrFileTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSize(tt.size)
			if !errors.Is(err, tt.wantErr) {
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
		{
			name:    "CS2 magic bytes",
			header:  []byte("PBDEMS2\x00"),
			wantErr: nil,
		},
		{
			name:    "CSGO magic bytes",
			header:  []byte("HL2DEMO\x00"),
			wantErr: nil,
		},
		{
			name:    "CS2 magic with extra trailing bytes",
			header:  append([]byte("PBDEMS2\x00"), 0xFF, 0xFF),
			wantErr: nil,
		},
		{
			name:    "random bytes",
			header:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			wantErr: ErrInvalidMagicBytes,
		},
		{
			name:    "too short header",
			header:  []byte{0x01, 0x02, 0x03},
			wantErr: ErrInvalidMagicBytes,
		},
		{
			name:    "empty header",
			header:  []byte{},
			wantErr: ErrInvalidMagicBytes,
		},
		{
			name:    "nil header",
			header:  nil,
			wantErr: ErrInvalidMagicBytes,
		},
		{
			name:    "almost CS2 magic (wrong last byte)",
			header:  []byte("PBDEMS2\x01"),
			wantErr: ErrInvalidMagicBytes,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMagicBytes(tt.header)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateMagicBytes(%v) = %v, want %v", tt.header, err, tt.wantErr)
			}
		})
	}
}

func TestMaxUploadSize(t *testing.T) {
	// Verify MaxUploadSize is 1 GB.
	const oneGB int64 = 1 << 30
	if MaxUploadSize != oneGB {
		t.Errorf("MaxUploadSize = %d, want %d (1 GB)", MaxUploadSize, oneGB)
	}
}
