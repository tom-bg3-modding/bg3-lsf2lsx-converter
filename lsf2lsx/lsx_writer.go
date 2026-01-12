package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

func WriteLSX(filename string, resource *Resource) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return WriteLSXToWriter(file, resource)
}

func WriteLSXToWriter(w io.Writer, resource *Resource) error {
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "\t")

	_, err := w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>` + "\n"))
	if err != nil {
		return err
	}

	// Write root
	err = encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "save"}})
	if err != nil {
		return err
	}

	err = writeVersion(encoder, resource)
	if err != nil {
		return err
	}

	err = writeRegions(encoder, resource)
	if err != nil {
		return err
	}

	// Close root
	err = encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "save"}})
	if err != nil {
		return err
	}

	err = encoder.Flush()
	if err != nil {
		return err
	}

	return nil
}

func writeVersion(encoder *xml.Encoder, resource *Resource) error {
	attrs := []xml.Attr{
		{Name: xml.Name{Local: "major"}, Value: strconv.FormatUint(uint64(resource.Metadata.MajorVersion), 10)},
		{Name: xml.Name{Local: "minor"}, Value: strconv.FormatUint(uint64(resource.Metadata.MinorVersion), 10)},
		{Name: xml.Name{Local: "revision"}, Value: strconv.FormatUint(uint64(resource.Metadata.Revision), 10)},
		{Name: xml.Name{Local: "build"}, Value: strconv.FormatUint(uint64(resource.Metadata.BuildNumber), 10)},
	}

	err := encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "version"}, Attr: attrs})
	if err != nil {
		return err
	}

	err = encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "version"}})
	if err != nil {
		return err
	}

	return nil
}

func writeRegions(encoder *xml.Encoder, resource *Resource) error {
	// Sort region names for deterministic output
	regionNames := make([]string, 0, len(resource.Regions))
	for regionName := range resource.Regions {
		regionNames = append(regionNames, regionName)
	}
	sort.Strings(regionNames)

	for _, regionName := range regionNames {
		region := resource.Regions[regionName]
		attrs := []xml.Attr{
			{Name: xml.Name{Local: "id"}, Value: regionName},
		}

		err := encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "region"}, Attr: attrs})
		if err != nil {
			return err
		}

		// BG3 uses LSX V4 format (type names instead of IDs)
		err = writeNode(encoder, &region.Node)
		if err != nil {
			return err
		}

		err = encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "region"}})
		if err != nil {
			return err
		}
	}

	return nil
}

func writeNode(encoder *xml.Encoder, node *Node) error {
	attrs := []xml.Attr{
		{Name: xml.Name{Local: "id"}, Value: node.Name},
	}

	if node.KeyAttribute != "" {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "key"}, Value: node.KeyAttribute})
	}

	err := encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "node"}, Attr: attrs})
	if err != nil {
		return err
	}

	// We sort everything before writing to ensure the LSX is deterministic.

	//// Attributes ////
	attrNames := make([]string, 0, len(node.Attributes))
	for attrName := range node.Attributes {
		attrNames = append(attrNames, attrName)
	}
	sort.Strings(attrNames)
	for _, attrName := range attrNames {
		err = writeAttribute(encoder, attrName, node.Attributes[attrName])
		if err != nil {
			return err
		}
	}

	//// Children ////
	if len(node.Children) > 0 {
		err = encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "children"}})
		if err != nil {
			return err
		}

		// Sort child node names alphabetically first
		childNames := make([]string, 0, len(node.Children))
		for childName := range node.Children {
			childNames = append(childNames, childName)
		}
		sort.Strings(childNames)

		for _, childName := range childNames {
			children := node.Children[childName]
			// Multiple children with the same name - sort by their hash
			if len(children) > 1 {
				sort.Slice(children, func(i, j int) bool {
					return nodeHashString(children[i]) < nodeHashString(children[j])
				})
			}
			for _, child := range children {
				err = writeNode(encoder, child)
				if err != nil {
					return err
				}
			}
		}

		err = encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "children"}})
		if err != nil {
			return err
		}
	}

	err = encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "node"}})
	if err != nil {
		return err
	}

	return nil
}

func writeAttribute(encoder *xml.Encoder, attrName string, attr *NodeAttribute) error {
	attrs := []xml.Attr{
		{Name: xml.Name{Local: "id"}, Value: attrName},
	}

	// Type attribute (BG3 always uses V4 format with type names)
	typeStr := attributeTypeToString(attr.Type)
	attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "type"}, Value: typeStr})

	// Value attribute
	switch attr.Type {
	case AttrTranslatedString:
		ts := attr.Value.(*TranslatedString)
		if ts.Handle != "" {
			attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "handle"}, Value: ts.Handle})
		}
		if ts.Value != "" {
			attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "value"}, Value: ts.Value})
		} else {
			attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "version"}, Value: strconv.FormatUint(uint64(ts.Version), 10)})
		}

	case AttrTranslatedFSString:
		fs := attr.Value.(*TranslatedFSString)
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "value"}, Value: fs.Value})
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "handle"}, Value: fs.Handle})
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "arguments"}, Value: strconv.Itoa(len(fs.Arguments))})

	default:
		valueStr := attributeValueToString(attr)
		// Remove bogus 0x1F characters
		cleanValue := ""
		for _, r := range valueStr {
			if r != 0x1F {
				cleanValue += string(r)
			}
		}
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "value"}, Value: cleanValue})
	}

	err := encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "attribute"}, Attr: attrs})
	if err != nil {
		return err
	}

	// Handle TranslatedFSString arguments
	if attr.Type == AttrTranslatedFSString {
		fs := attr.Value.(*TranslatedFSString)
		if len(fs.Arguments) > 0 {
			err = encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "arguments"}})
			if err != nil {
				return err
			}

			for _, arg := range fs.Arguments {
				err = writeTranslatedFSStringArgument(encoder, arg)
				if err != nil {
					return err
				}
			}

			err = encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "arguments"}})
			if err != nil {
				return err
			}
		}
	}

	err = encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "attribute"}})
	if err != nil {
		return err
	}

	return nil
}

func writeTranslatedFSStringArgument(encoder *xml.Encoder, arg TranslatedFSStringArgument) error {
	attrs := []xml.Attr{
		{Name: xml.Name{Local: "key"}, Value: arg.Key},
		{Name: xml.Name{Local: "value"}, Value: arg.Value},
	}

	err := encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "argument"}, Attr: attrs})
	if err != nil {
		return err
	}

	// Write nested string
	err = writeTranslatedFSString(encoder, arg.String)
	if err != nil {
		return err
	}

	err = encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "argument"}})
	if err != nil {
		return err
	}

	return nil
}

func writeTranslatedFSString(encoder *xml.Encoder, fs TranslatedFSString) error {
	attrs := []xml.Attr{
		{Name: xml.Name{Local: "value"}, Value: fs.Value},
		{Name: xml.Name{Local: "handle"}, Value: fs.Handle},
		{Name: xml.Name{Local: "arguments"}, Value: strconv.Itoa(len(fs.Arguments))},
	}

	err := encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "string"}, Attr: attrs})
	if err != nil {
		return err
	}

	if len(fs.Arguments) > 0 {
		err = encoder.EncodeToken(xml.StartElement{Name: xml.Name{Local: "arguments"}})
		if err != nil {
			return err
		}

		for _, arg := range fs.Arguments {
			err = writeTranslatedFSStringArgument(encoder, arg)
			if err != nil {
				return err
			}
		}

		err = encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "arguments"}})
		if err != nil {
			return err
		}
	}

	err = encoder.EncodeToken(xml.EndElement{Name: xml.Name{Local: "string"}})
	if err != nil {
		return err
	}

	return nil
}

func attributeTypeToString(attrType AttributeType) string {
	typeMap := map[AttributeType]string{
		AttrByte:               "uint8",
		AttrShort:              "int16",
		AttrUShort:             "uint16",
		AttrInt:                "int32",
		AttrUInt:               "uint32",
		AttrFloat:              "float",
		AttrDouble:             "double",
		AttrIVec2:              "ivec2",
		AttrIVec3:              "ivec3",
		AttrIVec4:              "ivec4",
		AttrVec2:               "fvec2",
		AttrVec3:               "fvec3",
		AttrVec4:               "fvec4",
		AttrMat2:               "mat2x2",
		AttrMat3:               "mat3x3",
		AttrMat3x4:             "mat3x4",
		AttrMat4x3:             "mat4x3",
		AttrMat4:               "mat4x4",
		AttrBool:               "bool",
		AttrString:             "string",
		AttrPath:               "path",
		AttrFixedString:        "FixedString",
		AttrLSString:           "LSString",
		AttrULongLong:          "uint64",
		AttrScratchBuffer:      "ScratchBuffer",
		AttrLong:               "old_int64",
		AttrInt8:               "int8",
		AttrTranslatedString:   "TranslatedString",
		AttrWString:            "WString",
		AttrLSWString:          "LSWString",
		AttrUUID:               "guid",
		AttrInt64:              "int64",
		AttrTranslatedFSString: "TranslatedFSString",
	}

	if str, ok := typeMap[attrType]; ok {
		return str
	}
	return "None"
}

func byteSwapUUID(uuid []byte) []byte {
	if len(uuid) != 16 {
		return uuid
	}
	result := make([]byte, 16)
	copy(result, uuid)
	// Swap bytes 8-15 in pairs (indices 8-9, 10-11, 12-13, 14-15)
	for i := 8; i < 16; i += 2 {
		result[i], result[i+1] = result[i+1], result[i]
	}
	return result
}

func formatUUID(uuid []byte, byteSwap bool) string {
	if len(uuid) != 16 {
		return fmt.Sprintf("%x", uuid)
	}

	bytes := uuid
	if byteSwap {
		bytes = byteSwapUUID(uuid)
	}

	// Format as GUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		bytes[3], bytes[2], bytes[1], bytes[0], // First 4 bytes (little-endian)
		bytes[5], bytes[4], // Next 2 bytes
		bytes[7], bytes[6], // Next 2 bytes
		bytes[8], bytes[9], // Next 2 bytes
		bytes[10], bytes[11], bytes[12], bytes[13], bytes[14], bytes[15]) // Last 6 bytes
}

/*
Used to sort nodes deterministically.

We need this cos we wanna diff the LSXs, and nodes being in a different order will flag changes that
aren't actually changes. There's 2 ways nodes could get out of order in the data resource:
  - The maps we use to store the nodes are an unordered data structure
  - The actual binary gets out of order (The Divinity Engine might not write em deterministically)

We can't just sort alphabetically because sibling nodes can have the same name (i.e. Object)). The
simplest way to guarantee the same nodes are always in the same order is to hash the entire node
(attributes + children) and sort by that.
*/
func nodeHashString(node *Node) string {
	var result strings.Builder

	// Start with key attribute if present
	if node.KeyAttribute != "" {
		result.WriteString("key:")
		result.WriteString(node.KeyAttribute)
		result.WriteString("|")
	}

	// Add all attributes sorted by name
	attrNames := make([]string, 0, len(node.Attributes))
	for attrName := range node.Attributes {
		attrNames = append(attrNames, attrName)
	}
	sort.Strings(attrNames)

	for _, attrName := range attrNames {
		result.WriteString(attrName)
		result.WriteString(":")
		result.WriteString(attributeValueToString(node.Attributes[attrName]))
		result.WriteString("|")
	}

	// Add all children recursively, sorted by name then by their hash strings
	childNames := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		childNames = append(childNames, childName)
	}
	sort.Strings(childNames)

	for _, childName := range childNames {
		children := node.Children[childName]
		// Sort children with the same name by their hash strings
		if len(children) > 1 {
			childHashes := make([]struct {
				node *Node
				hash string
			}, len(children))
			for i, child := range children {
				childHashes[i] = struct {
					node *Node
					hash string
				}{child, nodeHashString(child)}
			}
			sort.Slice(childHashes, func(i, j int) bool {
				return childHashes[i].hash < childHashes[j].hash
			})
			for _, ch := range childHashes {
				result.WriteString(childName)
				result.WriteString(":")
				result.WriteString(ch.hash)
				result.WriteString("|")
			}
		} else {
			for _, child := range children {
				result.WriteString(childName)
				result.WriteString(":")
				result.WriteString(nodeHashString(child))
				result.WriteString("|")
			}
		}
	}
	fmt.Println(result.String())

	return result.String()
}

func attributeValueToString(attr *NodeAttribute) string {
	switch v := attr.Value.(type) {
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	case bool:
		if v {
			return "True"
		}
		return "False"
	case uint64:
		return strconv.FormatUint(v, 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case string:
		return v
	case []byte:
		// Check if this is a UUID (16 bytes) or ScratchBuffer
		if attr.Type == AttrUUID && len(v) == 16 {
			// BG3 always byte-swaps GUIDs
			return formatUUID(v, true)
		}
		// ScratchBuffer - format as hex
		return fmt.Sprintf("%x", v)
	case [2]int32:
		return fmt.Sprintf("%d %d", v[0], v[1])
	case [3]int32:
		return fmt.Sprintf("%d %d %d", v[0], v[1], v[2])
	case [4]int32:
		return fmt.Sprintf("%d %d %d %d", v[0], v[1], v[2], v[3])
	case [2]float32:
		return fmt.Sprintf("%g %g", v[0], v[1])
	case [3]float32:
		return fmt.Sprintf("%g %g %g", v[0], v[1], v[2])
	case [4]float32:
		return fmt.Sprintf("%g %g %g %g", v[0], v[1], v[2], v[3])
	case []float32:
		// For matrices
		result := ""
		for i, f := range v {
			if i > 0 {
				result += " "
			}
			result += strconv.FormatFloat(float64(f), 'g', -1, 32)
		}
		return result
	default:
		return fmt.Sprintf("%v", v)
	}
}
