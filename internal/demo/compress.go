package demo

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// DecompressZstd decompresses a zstandard-compressed file. The decompressed
// output is written alongside the source file with the .zst suffix removed
// (e.g. "match.dem.zst" → "match.dem"). Returns the path to the decompressed file.
func DecompressZstd(srcPath string) (string, error) {
	outPath := strings.TrimSuffix(srcPath, ".zst")
	if outPath == srcPath {
		return "", fmt.Errorf("file does not have .zst suffix: %s", srcPath)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("opening zstd file: %w", err)
	}
	defer src.Close() //nolint:errcheck

	decoder, err := zstd.NewReader(src)
	if err != nil {
		return "", fmt.Errorf("creating zstd decoder: %w", err)
	}
	defer decoder.Close()

	dst, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("creating output file: %w", err)
	}

	if _, err := io.Copy(dst, decoder); err != nil {
		_ = dst.Close()
		_ = os.Remove(outPath)
		return "", fmt.Errorf("decompressing: %w", err)
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(outPath)
		return "", fmt.Errorf("closing output file: %w", err)
	}

	return outPath, nil
}

// decompressZstdToFile decompresses a zstandard-compressed file into the given
// target file path. Removes the partial output if decompression fails.
func decompressZstdToFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("opening zstd file: %w", err)
	}
	defer src.Close() //nolint:errcheck

	decoder, err := zstd.NewReader(src)
	if err != nil {
		return fmt.Errorf("creating zstd decoder: %w", err)
	}
	defer decoder.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}

	if _, err := io.Copy(dst, decoder); err != nil {
		_ = dst.Close()
		_ = os.Remove(dstPath)
		return fmt.Errorf("decompressing: %w", err)
	}
	if err := dst.Close(); err != nil {
		_ = os.Remove(dstPath)
		return fmt.Errorf("closing output file: %w", err)
	}
	return nil
}

// DecompressZstdTo decompresses a zstandard-compressed file into the given
// target directory with a .dem extension. Used by the download service where
// the source is a temp file without a meaningful name.
func DecompressZstdTo(srcPath, targetDir string) (string, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("opening zstd file: %w", err)
	}
	defer src.Close() //nolint:errcheck

	decoder, err := zstd.NewReader(src)
	if err != nil {
		return "", fmt.Errorf("creating zstd decoder: %w", err)
	}
	defer decoder.Close()

	outFile, err := os.CreateTemp(targetDir, "faceit-demo-*.dem")
	if err != nil {
		return "", fmt.Errorf("creating output file: %w", err)
	}
	outPath := outFile.Name()

	if _, err := io.Copy(outFile, decoder); err != nil {
		_ = outFile.Close()
		_ = os.Remove(outPath)
		return "", fmt.Errorf("decompressing: %w", err)
	}
	if err := outFile.Close(); err != nil {
		_ = os.Remove(outPath)
		return "", fmt.Errorf("closing output file: %w", err)
	}

	return outPath, nil
}
