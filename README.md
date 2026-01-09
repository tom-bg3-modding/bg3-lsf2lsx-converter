# LSF to LSX Converter for Baldur's Gate 3

A Go implementation of Norbyte's LSF to LSX file converter. Converts LSF (binary) files to human-readable LSX (XML) format. This tool is designed for use with BG3 only, LSFs from DOS2/DOS will not work.
The primary use of this tool is within BG3 mod repositories to implement text diffs of LSF binaries using git textconv.

## Installation

```bash
cd lsf2lsx
go mod download
go build
```

## Usage

Git textconv (writes to stdout):
```bash
./lsf2lsx <input.lsf>
```

For file output:
```bash
./lsf2lsx -input <input.lsf> -output <output.lsx>
```

If `-output` is not specified, the LSX is written to stdout. The input file can be provided either as a positional argument or via the `-input` flag.

## Requirements

- Go 1.21 or later
- Dependencies:
  - `github.com/DataDog/zstd` - Zstandard compression
  - `github.com/pierrec/lz4/v4` - LZ4 compression

## Implementation Details

The converter follows the same architecture as Norbyte's original C# implementation:

1. **LSF Reader** (`lsf_reader.go`): Reads binary LSF format
   - Parses file headers and metadata
   - Decompresses sections (strings, nodes, attributes, values)
   - Builds in-memory Resource structure

2. **Compression** (`compression.go`): Handles decompression
   - Supports LZ4, Zlib, and Zstandard
   - Handles chunked and non-chunked formats

3. **LSX Writer** (`lsx_writer.go`): Writes XML format
   - Converts Resource structure to XML
   - Handles special types (TranslatedString, TranslatedFSString)
   - Pretty-prints with indentation

4. **Data Structures** (`types.go`): Core data types
   - Resource, Region, Node, NodeAttribute
   - Attribute types and special string types

## File Format Support

- **LSF Versions**: 5-7 (BG3 Extended Header, Node Keys, Patch 3)
- **Compression**: None, LZ4, Zlib, Zstandard
- **Games**: Baldur's Gate 3
- **LSX Format**: Version 4 (uses type names instead of numeric type IDs)

See the [DOCS](DOCS.md) file for a more detailed breakdown of how the tool works.

