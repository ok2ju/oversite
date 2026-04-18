package demo

import (
	"bytes"
	"errors"
	"strings"
)

// MaxUploadSize is the maximum allowed demo file size (1 GB).
const MaxUploadSize int64 = 1 << 30

var (
	// MagicCS2 is the file magic for CS2 demo files.
	MagicCS2 = []byte("PBDEMS2\x00")
	// MagicCSGO is the file magic for CS:GO demo files.
	MagicCSGO = []byte("HL2DEMO\x00")
)

var (
	ErrInvalidExtension  = errors.New("invalid file extension: must be .dem or .dem.zst")
	ErrFileTooLarge      = errors.New("file exceeds maximum upload size")
	ErrInvalidMagicBytes = errors.New("invalid file format: not a valid demo file")
)

// ValidateExtension checks that the filename ends with .dem or .dem.zst (case-insensitive).
func ValidateExtension(filename string) error {
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".dem.zst") || strings.HasSuffix(lower, ".dem") {
		return nil
	}
	return ErrInvalidExtension
}

// IsCompressedDemo returns true if the filename ends with .dem.zst (case-insensitive).
func IsCompressedDemo(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".dem.zst")
}

// IsDemoFile returns true if the filename is a demo file (.dem or .dem.zst).
func IsDemoFile(filename string) bool {
	return ValidateExtension(filename) == nil
}

// ValidateSize checks that the file size does not exceed MaxUploadSize.
func ValidateSize(size int64) error {
	if size > MaxUploadSize {
		return ErrFileTooLarge
	}
	return nil
}

// ValidateMagicBytes checks the first 8 bytes of the file for CS2 or CS:GO demo magic.
func ValidateMagicBytes(header []byte) error {
	if len(header) < 8 {
		return ErrInvalidMagicBytes
	}
	prefix := header[:8]
	if bytes.Equal(prefix, MagicCS2) || bytes.Equal(prefix, MagicCSGO) {
		return nil
	}
	return ErrInvalidMagicBytes
}
