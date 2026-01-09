// Unified handler for binary reading operations

package main

import (
	"bytes"
	"encoding/binary"
	"io"
)

type binaryReader struct {
	io.Reader
	io.Seeker
}

func newBinaryReader(r io.ReadSeeker) *binaryReader {
	return &binaryReader{Reader: r, Seeker: r}
}

func newBinaryReaderFromBytes(data []byte) *binaryReader {
	br := bytes.NewReader(data)
	return &binaryReader{
		Reader: br,
		Seeker: br,
	}
}

func (r *binaryReader) Len() int {
	if br, ok := r.Reader.(*bytes.Reader); ok {
		return br.Len()
	}
	return 0
}

func readUint8(reader io.Reader) (uint8, error) {
	var val uint8
	err := binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}

func readUint16(reader io.Reader) (uint16, error) {
	var val uint16
	err := binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}

func readUint32(reader io.Reader) (uint32, error) {
	var val uint32
	err := binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}

func readUint64(reader io.Reader) (uint64, error) {
	var val uint64
	err := binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}

func readInt8(reader io.Reader) (int8, error) {
	var val int8
	err := binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}

func readInt16(reader io.Reader) (int16, error) {
	var val int16
	err := binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}

func readInt32(reader io.Reader) (int32, error) {
	var val int32
	err := binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}

func readInt64(reader io.Reader) (int64, error) {
	var val int64
	err := binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}

func readFloat32(reader io.Reader) (float32, error) {
	var val float32
	err := binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}

func readFloat64(reader io.Reader) (float64, error) {
	var val float64
	err := binary.Read(reader, binary.LittleEndian, &val)
	return val, err
}

// reads a value based on attribute type
func readAttributeValue(attrType AttributeType, reader io.Reader) interface{} {
	switch attrType {
	case AttrByte:
		val, _ := readUint8(reader)
		return val
	case AttrShort:
		val, _ := readInt16(reader)
		return val
	case AttrUShort:
		val, _ := readUint16(reader)
		return val
	case AttrInt:
		val, _ := readInt32(reader)
		return val
	case AttrUInt:
		val, _ := readUint32(reader)
		return val
	case AttrFloat:
		val, _ := readFloat32(reader)
		return val
	case AttrDouble:
		val, _ := readFloat64(reader)
		return val
	case AttrBool:
		val, _ := readUint8(reader)
		return val != 0
	case AttrULongLong:
		val, _ := readUint64(reader)
		return val
	case AttrInt64:
		val, _ := readInt64(reader)
		return val
	case AttrInt8:
		val, _ := readInt8(reader)
		return val
	case AttrIVec2:
		x, _ := readInt32(reader)
		y, _ := readInt32(reader)
		return [2]int32{x, y}
	case AttrIVec3:
		x, _ := readInt32(reader)
		y, _ := readInt32(reader)
		z, _ := readInt32(reader)
		return [3]int32{x, y, z}
	case AttrIVec4:
		x, _ := readInt32(reader)
		y, _ := readInt32(reader)
		z, _ := readInt32(reader)
		w, _ := readInt32(reader)
		return [4]int32{x, y, z, w}
	case AttrVec2:
		x, _ := readFloat32(reader)
		y, _ := readFloat32(reader)
		return [2]float32{x, y}
	case AttrVec3:
		x, _ := readFloat32(reader)
		y, _ := readFloat32(reader)
		z, _ := readFloat32(reader)
		return [3]float32{x, y, z}
	case AttrVec4:
		x, _ := readFloat32(reader)
		y, _ := readFloat32(reader)
		z, _ := readFloat32(reader)
		w, _ := readFloat32(reader)
		return [4]float32{x, y, z, w}
	case AttrMat2:
		vals := make([]float32, 2*2)
		for i := range vals {
			vals[i], _ = readFloat32(reader)
		}
		return vals
	case AttrMat3:
		vals := make([]float32, 3*3)
		for i := range vals {
			vals[i], _ = readFloat32(reader)
		}
		return vals
	case AttrMat3x4:
		vals := make([]float32, 3*4)
		for i := range vals {
			vals[i], _ = readFloat32(reader)
		}
		return vals
	case AttrMat4x3:
		vals := make([]float32, 4*3)
		for i := range vals {
			vals[i], _ = readFloat32(reader)
		}
		return vals
	case AttrMat4:
		vals := make([]float32, 4*4)
		for i := range vals {
			vals[i], _ = readFloat32(reader)
		}
		return vals
	case AttrUUID:
		// UUID is 16 bytes
		uuid := make([]byte, 16)
		reader.Read(uuid)
		return uuid
	default:
		return nil
	}
}
