package main

import "io"

// Divinity Engine version
type PackedVersion struct {
	Major    uint32
	Minor    uint32
	Revision uint32
	Build    uint32
}

// LSMetadata contains version information
type LSMetadata struct {
	Timestamp    uint64
	MajorVersion uint32
	MinorVersion uint32
	Revision     uint32
	BuildNumber  uint32
}

// Resource is the root structure containing regions
type Resource struct {
	Metadata LSMetadata
	Regions  map[string]*Region
}

// Region is a top-level container (root node)
type Region struct {
	Node
	RegionName string
}

// Node represents a node in the resource tree
type Node struct {
	Name         string
	Parent       *Node
	Attributes   map[string]*NodeAttribute
	Children     map[string][]*Node
	KeyAttribute string
}

// AppendChild adds a child node
func (n *Node) AppendChild(child *Node) {
	if n.Children == nil {
		n.Children = make(map[string][]*Node)
	}
	if n.Children[child.Name] == nil {
		n.Children[child.Name] = make([]*Node, 0)
	}
	n.Children[child.Name] = append(n.Children[child.Name], child)
}

// NodeAttribute represents an attribute of a node
type NodeAttribute struct {
	Type  AttributeType
	Value interface{}
}

// AttributeType represents the type of an attribute
type AttributeType uint32

const (
	AttrNone AttributeType = iota
	AttrByte
	AttrShort
	AttrUShort
	AttrInt
	AttrUInt
	AttrFloat
	AttrDouble
	AttrIVec2
	AttrIVec3
	AttrIVec4
	AttrVec2
	AttrVec3
	AttrVec4
	AttrMat2
	AttrMat3
	AttrMat3x4
	AttrMat4x3
	AttrMat4
	AttrBool
	AttrString
	AttrPath
	AttrFixedString
	AttrLSString
	AttrULongLong
	AttrScratchBuffer
	AttrLong
	AttrInt8
	AttrTranslatedString
	AttrWString
	AttrLSWString
	AttrUUID
	AttrInt64
	AttrTranslatedFSString
	AttrMax = AttrTranslatedFSString
)

// TranslatedString represents a translated string
type TranslatedString struct {
	Version uint16
	Handle  string
	Value   string
}

// TranslatedFSString represents a translated FS string
type TranslatedFSString struct {
	Version   uint16
	Handle    string
	Value     string
	Arguments []TranslatedFSStringArgument
}

// TranslatedFSStringArgument represents an argument to a TranslatedFSString
type TranslatedFSStringArgument struct {
	Key    string
	Value  string
	String TranslatedFSString
}

// Metadata format (BG3 only uses LSFMetadataKeysAndAdjacency)
type LSFMetadataFormat uint32

const (
	LSFMetadataNone             LSFMetadataFormat = 0
	LSFMetadataKeysAndAdjacency LSFMetadataFormat = 1
	LSFMetadataNone2            LSFMetadataFormat = 2
)

// LSF format versions (BG3 only uses 5-7)
const (
	LSFVersionBG3ExtendedHeader = 5
	LSFVersionBG3NodeKeys       = 6
	LSFVersionBG3Patch3         = 7
	LSFVersionMinBG3            = 5
	LSFVersionMaxBG3            = 7
)

// LSFMagic represents the magic file header
type LSFMagic struct {
	Magic   uint32
	Version uint32
}

// LSFHeader represents the header (v5+ only)
type LSFHeader struct {
	EngineVersion int64
}

// LSFMetadataV6 represents BG3 metadata format (always uses V6 with Keys section)
type LSFMetadataV6 struct {
	StringsUncompressedSize    uint32
	StringsSizeOnDisk          uint32
	KeysUncompressedSize       uint32
	KeysSizeOnDisk             uint32
	NodesUncompressedSize      uint32
	NodesSizeOnDisk            uint32
	AttributesUncompressedSize uint32
	AttributesSizeOnDisk       uint32
	ValuesUncompressedSize     uint32
	ValuesSizeOnDisk           uint32
	CompressionFlags           CompressionFlags
	Unknown2                   uint8
	Unknown3                   uint16
	MetadataFormat             LSFMetadataFormat
}

// LSFNodeEntryV3 represents BG3 node format (always uses V3 with extended nodes)
type LSFNodeEntryV3 struct {
	NameHashTableIndex  uint32
	ParentIndex         int32
	NextSiblingIndex    int32
	FirstAttributeIndex int32
}

// LSFAttributeEntryV3 represents BG3 attribute format (always uses V3 with adjacency data)
type LSFAttributeEntryV3 struct {
	NameHashTableIndex uint32
	TypeAndLength      uint32
	NextAttributeIndex int32
	Offset             uint32
}

// LSFKeyEntry represents a key attribute entry
type LSFKeyEntry struct {
	NodeIndex uint32
	KeyName   uint32
}

// LSFNodeInfo holds processed node information
type LSFNodeInfo struct {
	ParentIndex         int
	NameIndex           int
	NameOffset          int
	FirstAttributeIndex int
	KeyAttribute        string
}

// LSFAttributeInfo holds processed attribute information
type LSFAttributeInfo struct {
	NameIndex          int
	NameOffset         int
	TypeId             uint32
	Length             uint32
	DataOffset         uint32
	NextAttributeIndex int
}

// LSFReader reads LSF files (BG3-only)
type LSFReader struct {
	stream        io.ReadSeeker
	version       uint32
	gameVersion   PackedVersion
	metadata      *LSFMetadataV6 // BG3 always uses V6
	names         [][]string
	nodes         []*LSFNodeInfo
	attributes    []*LSFAttributeInfo
	nodeInstances []*Node
	values        []byte
}

// CompressionMethod represents the compression method
type CompressionMethod uint8

const (
	CompressionNone CompressionMethod = iota
	CompressionZlib
	CompressionLZ4
	CompressionZstd
)

// CompressionFlags is a bitfield for compression settings
type CompressionFlags uint8

func (f CompressionFlags) Method() CompressionMethod {
	return CompressionMethod(f & 0x0f)
}

func (f CompressionFlags) Level() uint8 {
	return uint8((f >> 4) & 0x0f)
}
