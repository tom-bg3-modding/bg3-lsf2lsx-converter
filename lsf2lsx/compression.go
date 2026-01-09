package main

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"

	"github.com/DataDog/zstd"
	"github.com/pierrec/lz4/v4"
)

// decompress decompresses data based on compression flags
func (r *LSFReader) decompress(reader *binaryReader, sizeOnDisk, uncompressedSize uint32, allowChunked bool) ([]byte, error) {
	meta := r.metadata

	if sizeOnDisk == 0 && uncompressedSize != 0 {
		// Data is not compressed
		buf := make([]byte, uncompressedSize)
		_, err := reader.Read(buf)
		return buf, err
	}

	if sizeOnDisk == 0 && uncompressedSize == 0 {
		// No data
		return []byte{}, nil
	}

	// BG3 always supports chunked compression (version >= 2)
	chunked := allowChunked
	isCompressed := meta.CompressionFlags.Method() != CompressionNone
	compressedSize := sizeOnDisk
	if !isCompressed {
		compressedSize = uncompressedSize
	}

	compressed := make([]byte, compressedSize)
	_, err := reader.Read(compressed)
	if err != nil {
		return nil, err
	}

	return decompressData(compressed, int(uncompressedSize), meta.CompressionFlags, chunked)
}

// decompressData decompresses data using the specified method
func decompressData(compressed []byte, decompressedSize int, flags CompressionFlags, chunked bool) ([]byte, error) {
	method := flags.Method()

	switch method {
	case CompressionNone:
		return compressed, nil

	case CompressionZlib:
		reader, err := zlib.NewReader(bytes.NewReader(compressed))
		if err != nil {
			return nil, err
		}
		defer reader.Close()

		decompressed := make([]byte, decompressedSize)
		_, err = io.ReadFull(reader, decompressed)
		if err != nil && err != io.EOF {
			return nil, err
		}
		return decompressed, nil

	case CompressionLZ4:
		if chunked {
			reader := lz4.NewReader(bytes.NewReader(compressed))
			decompressed := make([]byte, decompressedSize)
			_, err := io.ReadFull(reader, decompressed)
			if err != nil && err != io.EOF {
				return nil, err
			}
			return decompressed, nil
		} else {
			decompressed := make([]byte, decompressedSize)
			n, err := lz4.UncompressBlock(compressed, decompressed)
			if err != nil {
				return nil, err
			}
			if n != decompressedSize {
				return nil, fmt.Errorf("LZ4 decompression size mismatch: expected %d, got %d", decompressedSize, n)
			}
			return decompressed, nil
		}

	case CompressionZstd:
		decompressed, err := zstd.Decompress(nil, compressed)
		if err != nil {
			return nil, err
		}
		if len(decompressed) != decompressedSize {
			return nil, fmt.Errorf("Zstd decompression size mismatch: expected %d, got %d", decompressedSize, len(decompressed))
		}
		return decompressed, nil

	default:
		return nil, fmt.Errorf("unsupported compression method: %d", method)
	}
}

