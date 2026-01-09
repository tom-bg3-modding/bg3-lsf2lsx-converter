WIP

# What is an LSF File?

At a very high level, an LSF file describes the tree like structure of Game Object.

## Structure of an LSF File

The binary file is split up into 5 main sections:

1. The Magic Header - Identifies the file as LSF
2. The Header - Contains just the engine version
3. File Metadata - Contains info about how the data has been compressed
4. Compressed Data - 
5. Data Structure - 

### Magic Header

The magic header identifies the file type and version.
We can see it by dumping the first 8 bytes of a LSF file:
```
bg3-diff-summary % hexdump -C -n 8 test_data/92c22339-0552-45ee-b613-6bf5e9a268fd.lsf
00000000  4c 53 4f 46 07 00 00 00  |LSOF....|
```
The first 4 bytes are ASCII text reading `LSOF`. This identifies the file as an LSF. All LSF files must start with this. The second 4 bytes is a 32 bit integer containing the LSF version. There are currently 7 different versions of the LSF file format:

1.  VerInitial - Initial version
2.  VerChunkedCompress - Added chunked compression for substreams
3.  VerExtendedNodes - Extended node descriptors
4.  VerBG3 - BG3 version (no change from v3 besides version numbering. BG3 doesn't actually use it in the released game)
5.  VerBG3ExtendedHeader - BG3 with updated header metadata
6.  VerBG3NodeKeys - BG3 with node key names
7.  VerBG3Patch3 - BG3 Patch 3 version

BG3 only uses versions 5 to 7. This is why the converter can't handle DOS or DOS2 files, it's only programmed to convert the file versions used by BG3.

You may also notice the version number is stored in the first byte: `07 00 00 00`. LSF files stores all numeric fields in [little endian format](#endianness) for performance reasons. Strings are sequentially stored as UTF-8 encoded bytes.

### File Header

This contains just the Engine Version, stored as a packed 64 bit integer. The version is split into major (7 bits), minor (8 bits), revision (16 bits), and build number (31 bits). The final 2 bits are unused.

For example, the bytes `c8 00 00 00 00 00 04 02` translate to this packed binary.

```
0000001000000100000000000000000000000000000000000000000011001000
  |_____||______||______________||_____________________________|
   Major  Minor      Revision                 Build
```

Recall numbers are little endian, so we count backwards:
- 63-62: 00 unused
- 61-55: 0000100 = Major: 4
- 54-47: 00001000 = Minor: 8
- 46-31: 0000000000000000 = Revision: 0
- 30-0:  0000000000000000000000011001000 = Build: 200

### Metadata

The metadata section is the next 48 bytes. 

- Strings
- Keys
- Nodes
- Attributes
- Values

### Compressed Data

# Glossary

## Endianness

Endianness is the order in which bytes are arranged within binary data.
It's a similar concept to how some human languages can be written right to left (e.g. Hebrew), or even top to bottom (e.g. Traditional Korean) 

The two main types of endianness are **big endian** and **little endian**.

1. **Big Endian:** Bytes are stored with the most significant byte (the "big end") first, at the lowest memory address. Imagine writing the number 123 - the biggest part of the number (the hundred) is written first, followed by the twenty and then the three.

2. **Little Endian:** Bytes are stored with the least significant byte (the "little end") first, at the lowest address. This is similar to writing numbers backward - for instance we'd write one hundred and twenty three as "321".

As a more direct example, consider the number 1,234,567,890 which is `0x499602D2` in hexadecimal.

- **Big Endian:** The bytes are stored as `49 96 02 D2`, with `49` at the lower memory address.
- **Little Endian:** The bytes are stored as `D2 02 96 49`, with `D2` at the lower memory address.

The two formats have advantages over the other depending on the situation - e.g. big endian is easier for a human to read, but little endian is often faster for a computer to do maths on.