package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// LSOF file signature
var LSFMagicSignature = []byte{'L', 'S', 'O', 'F'}

// Wrapper for Read to handle file opening
func ReadLSF(filename string) (*Resource, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := &LSFReader{
		stream: file,
	}

	return reader.Read()
}

func (r *LSFReader) Read() (*Resource, error) {
	reader := newBinaryReader(r.stream)

	magic, err := r.readMagic(reader)
	if err != nil {
		return nil, err
	}

	r.version = magic.Version

	err = r.readHeader(reader)
	if err != nil {
		return nil, err
	}

	err = r.readMetadata(reader)
	if err != nil {
		return nil, err
	}

	err = r.readSections(reader)
	if err != nil {
		return nil, err
	}

	resource := r.buildResource()
	return resource, nil
}

///////////////////////////////
//// File Section Handlers ////
///////////////////////////////

func (r *LSFReader) readMagic(reader *binaryReader) (*LSFMagic, error) {
	magic := &LSFMagic{}
	err := binary.Read(reader, binary.LittleEndian, magic)
	if err != nil {
		return nil, err
	}

	expectedMagic := binary.LittleEndian.Uint32(LSFMagicSignature)
	if magic.Magic != expectedMagic {
		return nil, fmt.Errorf("invalid LSF signature; expected %08X, got %08X", expectedMagic, magic.Magic)
	}

	if magic.Version < LSFVersionMinBG3 || magic.Version > LSFVersionMaxBG3 {
		return nil, fmt.Errorf("LSF version %d is not supported (BG3 requires version 5-7, got %d)", magic.Version, magic.Version)
	}

	return magic, nil
}

func (r *LSFReader) readHeader(reader *binaryReader) error {
	header := &LSFHeader{}
	err := binary.Read(reader, binary.LittleEndian, header)
	if err != nil {
		return err
	}
	r.gameVersion = unpackVersion64(header.EngineVersion)

	// Duped lslib's logic for LSF files with missing engine version
	if r.gameVersion.Major == 0 {
		r.gameVersion.Major = 4
		r.gameVersion.Minor = 0
		r.gameVersion.Revision = 9
		r.gameVersion.Build = 0
	}

	return nil
}

// BG3 always uses V6 metadata format (with Keys section)
func (r *LSFReader) readMetadata(reader *binaryReader) error {
	meta := &LSFMetadataV6{}
	err := binary.Read(reader, binary.LittleEndian, meta)
	if err != nil {
		return err
	}
	r.metadata = meta
	return nil
}

func (r *LSFReader) readSections(reader *binaryReader) error {
	meta := r.metadata

	// Read names
	namesData, err := r.decompress(reader, meta.StringsSizeOnDisk, meta.StringsUncompressedSize, false)
	if err != nil {
		return err
	}
	r.readNames(namesData)

	// Read nodes - BG3 always uses V3 format (extended)
	nodesData, err := r.decompress(reader, meta.NodesSizeOnDisk, meta.NodesUncompressedSize, true)
	if err != nil {
		return err
	}
	r.readNodes(nodesData)

	attrsData, err := r.decompress(reader, meta.AttributesSizeOnDisk, meta.AttributesUncompressedSize, true)
	if err != nil {
		return err
	}
	r.readAttributesV3(attrsData)

	valuesData, err := r.decompress(reader, meta.ValuesSizeOnDisk, meta.ValuesUncompressedSize, true)
	if err != nil {
		return err
	}
	r.values = valuesData

	if meta.MetadataFormat == LSFMetadataKeysAndAdjacency && meta.KeysSizeOnDisk > 0 {
		keysData, err := r.decompress(reader, meta.KeysSizeOnDisk, meta.KeysUncompressedSize, true)
		if err != nil {
			return err
		}
		r.readKeys(keysData)
	}

	return nil
}

func (r *LSFReader) readNames(data []byte) error {
	reader := newBinaryReaderFromBytes(data)
	numHashEntries, err := readUint32(reader)
	if err != nil {
		return err
	}

	r.names = make([][]string, numHashEntries)
	for i := uint32(0); i < numHashEntries; i++ {
		numStrings, err := readUint16(reader)
		if err != nil {
			return err
		}

		hash := make([]string, 0, numStrings)
		for j := uint16(0); j < numStrings; j++ {
			nameLen, err := readUint16(reader)
			if err != nil {
				return err
			}
			nameBytes := make([]byte, nameLen)
			_, err = reader.Read(nameBytes)
			if err != nil {
				return err
			}
			hash = append(hash, string(nameBytes))
		}
		r.names[i] = hash
	}

	return nil
}

func (r *LSFReader) readNodes(data []byte) error {
	reader := newBinaryReaderFromBytes(data)
	r.nodes = make([]*LSFNodeInfo, 0)

	for reader.Len() > 0 {
		entry := &LSFNodeEntryV3{}
		err := binary.Read(reader, binary.LittleEndian, entry)
		if err != nil {
			return err
		}

		nodeInfo := &LSFNodeInfo{
			ParentIndex:         int(entry.ParentIndex),
			NameIndex:           int(entry.NameHashTableIndex >> 16),
			NameOffset:          int(entry.NameHashTableIndex & 0xffff),
			FirstAttributeIndex: int(entry.FirstAttributeIndex),
		}

		r.nodes = append(r.nodes, nodeInfo)
	}

	return nil
}

func (r *LSFReader) readAttributesV3(data []byte) error {
	reader := newBinaryReaderFromBytes(data)
	r.attributes = make([]*LSFAttributeInfo, 0)

	for reader.Len() > 0 {
		entry := &LSFAttributeEntryV3{}
		err := binary.Read(reader, binary.LittleEndian, entry)
		if err != nil {
			return err
		}

		attrInfo := &LSFAttributeInfo{
			NameIndex:          int(entry.NameHashTableIndex >> 16),
			NameOffset:         int(entry.NameHashTableIndex & 0xffff),
			TypeId:             entry.TypeAndLength & 0x3f,
			Length:             entry.TypeAndLength >> 6,
			DataOffset:         entry.Offset,
			NextAttributeIndex: int(entry.NextAttributeIndex),
		}

		r.attributes = append(r.attributes, attrInfo)
	}

	return nil
}

func (r *LSFReader) readKeys(data []byte) error {
	reader := newBinaryReaderFromBytes(data)

	for reader.Len() > 0 {
		entry := &LSFKeyEntry{}
		err := binary.Read(reader, binary.LittleEndian, entry)
		if err != nil {
			return err
		}

		keyNameIndex := int(entry.KeyName >> 16)
		keyNameOffset := int(entry.KeyName & 0xffff)
		keyAttribute := r.names[keyNameIndex][keyNameOffset]

		nodeIdx := int(entry.NodeIndex)
		if nodeIdx < len(r.nodes) {
			r.nodes[nodeIdx].KeyAttribute = keyAttribute
		}
	}

	return nil
}

func (r *LSFReader) buildResource() *Resource {
	resource := &Resource{
		Metadata: LSMetadata{
			MajorVersion: r.gameVersion.Major,
			MinorVersion: r.gameVersion.Minor,
			Revision:     r.gameVersion.Revision,
			BuildNumber:  r.gameVersion.Build,
		},
		Regions: make(map[string]*Region),
	}

	meta := r.metadata
	resource.MetadataFormat = meta.MetadataFormat

	// Build nodes
	r.nodeInstances = make([]*Node, len(r.nodes))
	valueReader := newBinaryReaderFromBytes(r.values)

	for i, nodeInfo := range r.nodes {
		var node *Node
		if nodeInfo.ParentIndex == -1 {
			// Root region
			region := &Region{
				Node: Node{
					Name:       r.names[nodeInfo.NameIndex][nodeInfo.NameOffset],
					Attributes: make(map[string]*NodeAttribute),
					Children:   make(map[string][]*Node),
				},
			}
			region.RegionName = region.Name
			region.KeyAttribute = nodeInfo.KeyAttribute
			node = &region.Node
			r.nodeInstances[i] = node
			resource.Regions[region.RegionName] = region
		} else { // Child node
			node = &Node{
				Name:       r.names[nodeInfo.NameIndex][nodeInfo.NameOffset],
				Parent:     r.nodeInstances[nodeInfo.ParentIndex],
				Attributes: make(map[string]*NodeAttribute),
				Children:   make(map[string][]*Node),
			}
			node.KeyAttribute = nodeInfo.KeyAttribute
			r.nodeInstances[i] = node
			r.nodeInstances[nodeInfo.ParentIndex].AppendChild(node)
		}

		// Read attributes
		if nodeInfo.FirstAttributeIndex != -1 {
			attrIdx := nodeInfo.FirstAttributeIndex
			for attrIdx != -1 {
				attrInfo := r.attributes[attrIdx]
				attrName := r.names[attrInfo.NameIndex][attrInfo.NameOffset]

				// Seek to attribute data
				valueReader.Seek(int64(attrInfo.DataOffset), 0)
				attrValue := r.readAttribute(AttributeType(attrInfo.TypeId), valueReader, attrInfo.Length)

				node.Attributes[attrName] = attrValue

				attrIdx = attrInfo.NextAttributeIndex
			}
		}
	}

	return resource
}

func (r *LSFReader) readAttribute(attrType AttributeType, reader *binaryReader, length uint32) *NodeAttribute {
	attr := &NodeAttribute{Type: attrType}

	switch attrType {
	case AttrString, AttrPath, AttrFixedString, AttrLSString, AttrWString, AttrLSWString:
		value := r.readString(reader, int(length))
		attr.Value = value

	case AttrTranslatedString:
		// BG3 always uses the new format (version field, no value field)
		ts := &TranslatedString{}
		ts.Version, _ = readUint16(reader)
		handleLen, _ := readInt32(reader)
		ts.Handle = r.readString(reader, int(handleLen))
		attr.Value = ts

	case AttrTranslatedFSString:
		fs := r.readTranslatedFSString(reader)
		attr.Value = fs

	case AttrScratchBuffer:
		buf := make([]byte, length)
		reader.Read(buf)
		attr.Value = buf

	default:
		// Use BinUtils equivalent
		value := readAttributeValue(attrType, reader)
		attr.Value = value
	}

	return attr
}

func (r *LSFReader) readTranslatedFSString(reader *binaryReader) *TranslatedFSString {
	// BG3 always uses the new format (version field, no value field)
	fs := &TranslatedFSString{}
	fs.Version, _ = readUint16(reader)

	handleLen, _ := readInt32(reader)
	fs.Handle = r.readString(reader, int(handleLen))

	argCount, _ := readInt32(reader)
	fs.Arguments = make([]TranslatedFSStringArgument, argCount)

	for i := int32(0); i < argCount; i++ {
		arg := TranslatedFSStringArgument{}
		argKeyLen, _ := readInt32(reader)
		arg.Key = r.readString(reader, int(argKeyLen))

		arg.String = *r.readTranslatedFSString(reader)

		argValueLen, _ := readInt32(reader)
		arg.Value = r.readString(reader, int(argValueLen))

		fs.Arguments[i] = arg
	}

	return fs
}

func (r *LSFReader) readString(reader *binaryReader, length int) string {
	if length == 0 {
		return ""
	}

	bytes := make([]byte, length-1)
	reader.Read(bytes)

	// Remove trailing nulls
	lastNull := len(bytes)
	for lastNull > 0 && bytes[lastNull-1] == 0 {
		lastNull--
	}

	nullTerm := make([]byte, 1)
	reader.Read(nullTerm)
	if nullTerm[0] != 0 {
		// Not strictly null-terminated, but continue anyway
	}

	return string(bytes[:lastNull])
}

// unpack the 64bit Divinity Engine version
func unpackVersion64(packed int64) PackedVersion {
	return PackedVersion{
		Major:    uint32((packed >> 55) & 0x7f),
		Minor:    uint32((packed >> 47) & 0xff),
		Revision: uint32((packed >> 31) & 0xffff),
		Build:    uint32(packed & 0x7fffffff),
	}
}
