/*
Package buffers provides a collection of types, constants and functions for manipulating data buffers,
inspired by WeiDU's set of functions.
*/
package buffers

import (
  "bytes"
  "compress/zlib"
  "encoding/binary"
  "io"
  "io/ioutil"

  "github.com/InfinityTools/go-ietools"
  "golang.org/x/text/encoding/charmap"
)

// Predefined argument lists for function GetOffsetArray()
var (
  ARE_V10_ACTORS                        = []int{0x54, 4, 0x58, 2, 0, 0, 0x110}
  ARE_V10_REGIONS                       = []int{0x5c, 4, 0x5a, 2, 0, 0, 0xc4}
  ARE_V10_SPAWN_POINTS                  = []int{0x60, 4, 0x64, 4, 0, 0, 0xc8}
  ARE_V10_ENTRANCES                     = []int{0x68, 4, 0x6c, 4, 0, 0, 0x68}
  ARE_V10_CONTAINERS                    = []int{0x70, 4, 0x74, 2, 0, 0, 0xc0}
  ARE_V10_AMBIENTS                      = []int{0x84, 4, 0x82, 2, 0, 0, 0xd4}
  ARE_V10_DOORS                         = []int{0xa8, 4, 0xa4, 4, 0, 0, 0xc8}
  ARE_V10_ANIMATIONS                    = []int{0xb0, 4, 0xac, 4, 0, 0, 0x4c}

  ARE_V91_ACTORS                        = []int{0x64, 4, 0x68, 2, 0, 0, 0x110}

  CRE_V10_KNOWN_SPELLS                  = []int{0x2a0, 4, 0x2a4, 4, 0, 0, 0xc}
  CRE_V10_SPELL_MEM_INFO                = []int{0x2a8, 4, 0x2ac, 4, 0, 0, 0x10}
  CRE_V10_EFFECTS                       = []int{0x2c4, 4, 0x2c8, 4, 0, 0, 0x108}
  CRE_V10_ITEMS                         = []int{0x2bc, 4, 0x2c0, 4, 0, 0, 0x14}

  ITM_V10_HEADERS                       = []int{0x64, 4, 0x68, 2, 0, 0, 0x38}
  ITM_V10_GEN_EFFECTS                   = []int{0x6a, 4, 0x70, 2, 0x6e, 2, 0x30}

  SPL_V10_HEADERS                       = []int{0x64, 4, 0x68, 2, 0, 0, 0x28}
  SPL_V10_GEN_EFFECTS                   = []int{0x6a, 4, 0x70, 2, 0x6e, 2, 0x30}

  STO_V10_ITEMS_PURCHASED               = []int{0x2c, 4, 0x30, 4, 0, 0, 0x4}
  STO_V10_ITEMS_SOLD                    = []int{0x34, 4, 0x38, 4, 0, 0, 0x1c}
  STO_V10_DRINKS                        = []int{0x4c, 4, 0x50, 4, 0, 0, 0x14}
  STO_V10_CURES                         = []int{0x70, 4, 0x74, 4, 0, 0, 0xc}

  WMP_AREAS                             = []int{0x34, 4, 0x30, 4, 0, 0, 0xf0}
  WMP_LINKS                             = []int{0x38, 4, 0x3c, 4, 0, 0, 0xd8}
)

// Predefined argument lists for function GetOffsetArray2(). The first argument must be specified manually.
var (
  ARE_V10_ITEMS                         = []int{0x78, 4, 0x44, 4, 0x40, 4, 0x14}
  ARE_V10_REGION_VERTICES               = []int{0x7c, 4, 0x2a, 2, 0x2c, 4, 0x4}
  ARE_V10_CONTAINER_VERTICES            = []int{0x7c, 4, 0x54, 2, 0x50, 4, 0x4}
  ARE_V10_DOOR_OPEN_OUTLINE_VERTICES    = []int{0x7c, 4, 0x30, 2, 0x2c, 4, 0x4}
  ARE_V10_DOOR_CLOSED_OUTLINE_VERTICES  = []int{0x7c, 4, 0x32, 2, 0x34, 4, 0x4}
  ARE_V10_DOOR_OPEN_CELL_VERTICES       = []int{0x7c, 4, 0x4c, 2, 0x48, 4, 0x4}
  ARE_V10_DOOR_CLOSED_CELL_VERTICES     = []int{0x7c, 4, 0x4e, 2, 0x50, 4, 0x4}

  CRE_V10_SPELL_MEM                     = []int{0x2b0, 4, 0xc, 4, 0x8, 4, 0xc}

  ITM_V10_HEAD_EFFECTS                  = []int{0x6a, 4, 0x1e, 2, 0x20, 2, 0x30}

  SPL_V10_HEAD_EFFECTS                  = []int{0x6a, 4, 0x1e, 2, 0x20, 2, 0x30}

  WMP_NORTH_LINKS                       = []int{0x38, 4, 0x54, 4, 0x50, 4, 0xd8}
  WMP_WEST_LINKS                        = []int{0x38, 4, 0x5c, 4, 0x58, 4, 0xd8}
  WMP_SOUTH_LINKS                       = []int{0x38, 4, 0x64, 4, 0x60, 4, 0xd8}
  WMP_EAST_LINKS                        = []int{0x38, 4, 0x6c, 4, 0x68, 4, 0xd8}
)

// Buffer contains the necessary information to provide read and write operations on buffer content.
type Buffer struct {
  buf []byte      // data buffer
  dirty bool      // true if content has been modified
  err error       // stores error state from last operation
}


// Create returns an empty Buffer object.
func Create() *Buffer {
  return Wrap(nil)
}

// Wrap attempts to wrap the given byte array into a Buffer object without the need of additional copy operations.
//
// As the Buffer takes ownership over the byte array, it is not advisable to make manual changes to the array afterwards.
// Wrap(nil) is functionally identical with Create().
func Wrap(buf []byte) *Buffer {
  if buf == nil { buf = make([]byte, 256) }
  buffer := Buffer { buf: buf, dirty: false, err: nil }
  return &buffer
}

// Load uses the given Reader to load data from the underlying buffer.
// The function returns a pointer to the Buffer object. Use function Error() to check if the function returned successfully.
func Load(r io.Reader) *Buffer {
  buffer := Buffer { nil, false, nil }

  buffer.buf, buffer.err = ioutil.ReadAll(r)
  return &buffer
}


// Save writes the current Buffer content to the specified Writer.
// Does nothing if the Buffer is in an invalid state (see Error() function).
func (b *Buffer) Save(w io.Writer) {
  if b.err != nil { return }

  _, b.err = w.Write(b.buf)
  if b.err == nil {
    b.dirty = false
  }
}

// Bytes returns the underlying buffer content.
// Returns an empty buffer if the Buffer object is in an invalid state (see Error() function).
func (b *Buffer) Bytes() []byte {
  if b.err != nil { return make([]byte, 0) }

  return b.buf
}

// Error returns the error state of the most recent operation on Buffer.
// Use ClearError() function to clear the current error state.
func (b *Buffer) Error() error {
  return b.err
}

// ClearError clears the error state from the last Buffer operation.
// Must be called for subsequent operations to work correctly.
func (b *Buffer) ClearError() {
  b.err = nil
}

// BufferLength returns the current length of the buffer in bytes.
func (b *Buffer) BufferLength() int {
  return len(b.buf)
}

// IsModified returns whether the current buffer content has been modified by a previous operation.
//
// The return value is only provided for informal purposes. None of the Buffer functions rely on it.
func (b *Buffer) IsModified() bool {
  return b.dirty
}

// ClearModified explicitly marks the Buffer object as unmodified.
func (b *Buffer) ClearModified() {
  b.dirty = false
}

// GetUint8 returns the unsigned byte value at the specified offset.
// Operation is skipped if error state is set.
func (b *Buffer) GetUint8(offset int) uint8 {
  if b.err != nil { return 0 }
  if offset < 0 || offset >= len(b.buf) { b.err = ietools.ErrOffsetOutOfRange; return 0 }
  return b.buf[offset]
}

// GetInt8 returns the signed byte value at the specified offset.
// Operation is skipped if error state is set.
func (b *Buffer) GetInt8(offset int) int8 {
  return int8(b.GetUint8(offset))
}

// GetUint16 returns the unsigned short value at the specified offset.
// Operation is skipped if error state is set.
func (b *Buffer) GetUint16(offset int) uint16 {
  if b.err != nil { return 0 }
  if offset < 0 || offset + 2 > len(b.buf) { b.err = ietools.ErrOffsetOutOfRange; return 0 }
  return binary.LittleEndian.Uint16(b.buf[offset:])
}

// GetInt16 returns the signed short value at the specified offset.
// Operation is skipped if error state is set.
func (b *Buffer) GetInt16(offset int) int16 {
  return int16(b.GetUint16(offset))
}

// GetUint32 returns the unsigned long value at the specified offset.
// Operation is skipped if error state is set.
func (b *Buffer) GetUint32(offset int) uint32 {
  if b.err != nil { return 0 }
  if offset < 0 || offset + 4 > len(b.buf) { b.err = ietools.ErrOffsetOutOfRange; return 0 }
  return binary.LittleEndian.Uint32(b.buf[offset:])
}

// GetInt32 returns the signed long value at the specified offset.
// Operation is skipped if error state is set.
func (b *Buffer) GetInt32(offset int) int32 {
  return int32(b.GetUint32(offset))
}

// GetUint returns the value at the specified offset in native uint type.
//
// bitsize specifies the size of the value to read in bits and supports 8, 16 and 32 to return an unsigned byte,
// short and long value respectively. Operation is skipped if error state is set.
func (b *Buffer) GetUint(offset, bitsize int) uint {
  switch {
    case bitsize <= 8:
      return uint(b.GetUint8(offset))
    case bitsize <= 16:
      return uint(b.GetUint16(offset))
    default:
      return uint(b.GetUint32(offset))
  }
}

// GetInt returns the value at the specified offset in native int type.
//
// bitsize specifies the size of the value to read in bits and supports 8, 16 and 32 to return a signed byte,
// short and long value respectively. Operation is skipped if error state is set.
func (b *Buffer) GetInt(offset, bitsize int) int {
  switch {
    case bitsize <= 8:
      return int(b.GetInt8(offset))
    case bitsize <= 16:
      return int(b.GetInt16(offset))
    default:
      return int(b.GetInt32(offset))
  }
}

// GetString returns a string of given size (in bytes) from the specified offset.
//
// If "null" is true, then string stops at the first null-character.
// Text encoding is assumed to be ANSI Windows-1252.
// Operation is skipped if error state is set.
func (b *Buffer) GetString(offset, size int, null bool) string {
  return b.GetStringEx(offset, size, null, charmap.Windows1252)
}

// GetStringEx returns a string of given size (in bytes) from the specified offset.
//
// If "null" is true, then string stops at the first null-character.
// Text encoding is specified by cmap. Specify a nil charmap to skip the ANSI decoding operation and read
// raw utf-8 data. Operation is skipped if error state is set.
func (b *Buffer) GetStringEx(offset, size int, null bool, cmap *charmap.Charmap) string {
  if b.err != nil { return "" }
  if size <= 0 { return "" }
  if offset < 0 || offset + size > len(b.buf) { b.err = ietools.ErrOffsetOutOfRange; return "" }

  buf := b.buf[offset:offset+size]
  if null {
    for idx := 0; idx < size; idx++ {
      if b.buf[offset+idx] == 0 {
        buf = b.buf[offset:offset+idx]
        break
      }
    }
  }
  var s string
  if cmap != nil {
    s, b.err = ietools.AnsiToUtf8(buf, cmap)
  } else {
    s = string(buf)
  }
  return s
}

// GetBuffer returns a copy of the specified content region.
// Operation is skipped if error state is set.
func (b *Buffer) GetBuffer(offset, size int) []byte {
  if b.err != nil { return make([]byte, 0) }
  if offset < 0 || offset + size > len(b.buf) { b.err = ietools.ErrOffsetOutOfRange; return make([]byte, 0) }

  retVal := make([]byte, size)
  copy(retVal, b.buf[offset:offset+size])
  return retVal
}

// PutUInt8 writes the given unsigned byte value at the specified offset and returns the previous value.
// Operation is skipped if error state is set.
func (b *Buffer) PutUint8(offset int, value uint8) uint8 {
  var retVal uint8 = 0
  if b.err != nil { return retVal }
  if offset < 0 || offset >= len(b.buf) { b.err = ietools.ErrOffsetOutOfRange; return retVal }

  retVal = uint8(b.buf[offset])
  if retVal != value {
    b.buf[offset] = byte(value)
    b.dirty = true
  }
  return retVal
}

// PutInt8 writes the given signed byte value at the specified offset and returns the previous value.
// Operation is skipped if error state is set.
func (b *Buffer) PutInt8(offset int, value int8) int8 {
  return int8(b.PutUint8(offset, uint8(value)))
}

// PutUInt16 writes the given unsigned short value at the specified offset and returns the previous value.
// Operation is skipped if error state is set.
func (b *Buffer) PutUint16(offset int, value uint16) uint16 {
  var retVal uint16 = 0
  if b.err != nil { return retVal }
  if offset < 0 || offset + 2 > len(b.buf) { b.err = ietools.ErrOffsetOutOfRange; return retVal }

  retVal = binary.LittleEndian.Uint16(b.buf[offset:])
  if retVal != value {
    binary.LittleEndian.PutUint16(b.buf[offset:], value)
    b.dirty = true
  }
  return retVal
}

// PutInt16 writes the given signed short value at the specified offset and returns the previous value.
// Operation is skipped if error state is set.
func (b *Buffer) PutInt16(offset int, value int16) int16 {
  return int16(b.PutUint16(offset, uint16(value)))
}

// PutInt32 writes the given unsigned long value at the specified offset and returns the previous value.
// Operation is skipped if error state is set.
func (b *Buffer) PutUint32(offset int, value uint32) uint32 {
  var retVal uint32 = 0
  if b.err != nil { return retVal }
  if offset < 0 || offset + 4 > len(b.buf) { b.err = ietools.ErrOffsetOutOfRange; return retVal }

  retVal = binary.LittleEndian.Uint32(b.buf[offset:])
  if retVal != value {
    binary.LittleEndian.PutUint32(b.buf[offset:], value)
    b.dirty = true
  }
  return retVal
}

// PutInt32 writes the given signed long value at the specified offset and returns the previous value.
// Operation is skipped if error state is set.
func (b *Buffer) PutInt32(offset int, value int32) int32 {
  return int32(b.PutUint32(offset, uint32(value)))
}

// PutString writes the given string at the specified offset.
//
// Only the specified number of charaters will be written. Remaining space in the buffer will be filled with 0.
// Text encoding of is assumed to be ANSI Windows-1252.
// Operation is skipped if error state is set.
func (b *Buffer) PutString(offset, size int, value string) {
  b.PutStringEx(offset, size, value, charmap.Windows1252)
}

// PutStringEx writes the given string at the specified offset.
//
// Only the specified number of charaters will be written. Remaining space in the buffer will be filled with 0.
// Text encoding is specified by cmap. Specify a nil charmap to skip the ANSI encoding operation and write
// raw utf-8 data data. Operation is skipped if error state is set.
func (b *Buffer) PutStringEx(offset, size int, value string, cmap *charmap.Charmap) {
  if b.err != nil { return }
  if size <= 0 { return }
  if offset < 0 || offset + size > len(b.buf) { b.err = ietools.ErrOffsetOutOfRange; return }

  var buf []byte
  if cmap != nil {
    buf, b.err = ietools.Utf8ToAnsi(value, cmap)
    if b.err != nil { return }
  } else {
    buf = []byte(value)
  }

  equal := true
  for idx := 0; equal && idx < len(buf); idx++ {
    equal = (buf[idx] == b.buf[offset+idx])
  }

  if !equal {
    copy(b.buf[offset:offset+size], buf)
    for idx := len(buf); idx < size; idx++ {
      b.buf[offset+idx] = 0
    }
    b.dirty = true
  }
}

// PutBuffer writes the given byte slice at the specified offset.
// Operation is skipped if error state is set.
func (b *Buffer) PutBuffer(offset int, buf []byte) {
  if b.err != nil { return }
  if offset < 0 || offset + len(buf) > len(b.buf) { b.err = ietools.ErrOffsetOutOfRange; return }

  equal := true
  for idx := 0; equal && idx < len(buf); idx++ {
    equal = buf[idx] == b.buf[offset+idx]
  }

  if !equal {
    copy(b.buf[offset:offset+len(buf)], buf)
    b.dirty = true
  }
}

// ReplaceBuffer replaces the current byte array with the given array.
//
// The operation automatically resets any invalid state (see Error() function) and marks the
// Buffer object as modified. Specifying a nil array assigns an empty byte array.
func (b *Buffer) ReplaceBuffer(buf []byte) {
  if buf == nil { buf = make([]byte, 0) }
  b.buf = buf
  b.dirty = true
  b.err = nil
}

// InsertBytes inserts the given amount of bytes at the specified offset.
//
// Inserted bytes are zero by default. Operation is skipped if error state is set.
func (b *Buffer) InsertBytes(offset, size int) {
  if b.err != nil { return }
  if offset < 0 || offset > len(b.buf) { b.err = ietools.ErrOffsetOutOfRange; return }

  if size > 0 {
    // This approach will only allocate a new buffer if capacity is too small.
    l := len(b.buf) // original length
    b.buf = append(b.buf, make([]byte, size)...)
    copy(b.buf[offset+size:l+size], b.buf[offset:l])
    b.dirty = true
  }
}

// DeleteBytes removes the given amount of bytes from the buffer, starting at the specified offset.
// Operation is skipped if error state is set.
func (b *Buffer) DeleteBytes(offset, size int) {
  if b.err != nil { return }
  if offset < 0 || offset > len(b.buf) { b.err = ietools.ErrOffsetOutOfRange; return }

  if size > 0 {
    if offset == 0 {
      b.buf = b.buf[size:]
    } else {
      buf2 := b.buf[:len(b.buf)-size]
      copy(buf2[:offset], b.buf[:offset])
      if offset+size < len(buf2) {
        copy(buf2[offset:], b.buf[offset+size:len(b.buf)])
      }
      b.buf = buf2
    }
    b.dirty = true
  }
}

// DecompressInto attempts to decompress a zlib compressed block of the buffer and stores it in the specified buffer.
//
// Returns the target buffer to accomodate to size changes. Operation is skipped if error state is set.
func (b *Buffer) DecompressInto(offset, size int, buffer []byte) []byte {
  if b.err != nil { return buffer }
  if size <= 0 || offset < 0 || offset + size > len(b.buf) { b.err = ietools.ErrOffsetOutOfRange; return buffer }

  br := bytes.NewReader(b.buf[offset:offset+size])
  zr, err := zlib.NewReader(br)
  if err != nil { b.err = err; return buffer }
  defer zr.Close()

  if buffer == nil || len(buffer) == 0 {
    buffer = make([]byte, size)
  }

  totalBytes, bytesRead := 0, 0
  for {
    bytesRead, err = zr.Read(buffer[totalBytes:])
    totalBytes += bytesRead
    if totalBytes >= len(buffer) {
      buffer = append(buffer, make([]byte, len(buffer))...)
    }
    if err != nil { break }
  }

  if err != nil && err != io.EOF { b.err = err }

  if totalBytes < len(buffer) {
    buffer = buffer[:totalBytes]
  }

  return buffer
}

// DecompressReplace attempts to decompress a zlib compressed block of the buffer and replaces it with the
// decompressed content.
//
// Buffer size will be adjusted if needed. Returns size of the decompressed block. Operation is skipped if error state is set.
func (b *Buffer) DecompressReplace(offset, size int) int {
  if b.err != nil { return 0 }
  if size < 0 { size = 0 }
  buffer := b.DecompressInto(offset, size, nil)
  if b.err != nil { return 0 }

  if len(buffer) > size {
    b.InsertBytes(offset + size, len(buffer) - size)
  } else if len(buffer) < size {
    b.DeleteBytes(offset + len(buffer), size - len(buffer))
  }
  if b.err != nil { return 0 }

  copy(b.buf[offset:offset+len(buffer)], buffer)
  b.dirty = true
  return len(buffer)
}

// CompressInto attempts to zlib compress the buffer region specified by offset and size using compression rate "level"
// (in range 0 - 9).
//
// Special compression levels -2 (deflate only) and -1 (default compression) are also accepted.
// The compressed data is stored in the specified buffer. Returns the target buffer to accomodate to size changes.
// Operation is skipped if error state is set.
func (b *Buffer) CompressInto(offset, size, level int, buffer []byte) []byte {
  if b.err != nil { return buffer }
  if size < 0 || offset < 0 || offset + size > len(b.buf) { b.err = ietools.ErrOffsetOutOfRange; return buffer }
  if level < -2 { level = -2 } else if level > 9 { level = 9 }  // -2: deflate only, -1: default compression

  if buffer == nil {
    buffer = make([]byte, 0)
  }
  bw := bytes.NewBuffer(buffer)
  zw, err := zlib.NewWriterLevel(bw, level)
  if err != nil { b.err = err; return buffer }
  defer zw.Close()

  bytesWritten, err := zw.Write(b.buf[offset:offset+size])
  if err != nil { b.err = err; return buffer }
  err = zw.Flush()
  if err != nil { b.err = err; return buffer }

  buffer = bw.Bytes()
  if bytesWritten < len(buffer) {
    buffer = buffer[:bytesWritten]
  }
  return buffer
}

// CompressReplace attempts to zlib compress the buffer region specified by offset and size using compression rate "level"
// which can be anything between 0 and 9.
//
// Special compression levels -2 (deflate only) and -1 (default compression) are also accepted.
// Buffer size will be adjusted if needed. Returns size of the compressed block. Operation is skipped if error state is set.
func (b *Buffer) CompressReplace(offset, size, level int) int {
  if b.err != nil { return 0 }
  if size < 0 { size = 0 }
  buffer := b.CompressInto(offset, size, level, nil)
  if b.err != nil { return 0 }

  if len(buffer) > size {
    b.InsertBytes(offset + size, len(buffer) - size)
  } else if len(buffer) < size {
    b.DeleteBytes(offset + len(buffer), size - len(buffer))
  }
  if b.err != nil { return 0 }

  copy(b.buf[offset:offset+len(buffer)], buffer)
  b.dirty = true
  return len(buffer)
}


// GetOffsetArray is a specialized method for retrieving offsets to all available substructures of a type specified by
// the arguments.
//
// It is useful to quickly determine offsets to all available ability or effect structures in an item or spell resource.
// Seven parameters are required. The order of parameters is as follows:
//  ofs, ofsSize        The start offset and length of offset field to the list of substructures.
//  count, countSize    The number of substructures and length of the count field.
//  index, indexSize    An optional start index and length of index field for the substructures.
//                      Set to 0 to ignore.
//  structSize          The size of a substructure in bytes. Must be non-zero.
// Returns an array of offsets for each individual substructure found in the current buffer content.
// The package provides a number of predefined configurations for compatible structures.
// Operation is skipped if error state is set.
func (b *Buffer) GetOffsetArray(sevenValues ...int) []int {
  if b.err != nil { return make([]int, 0) }
  if sevenValues == nil || len(sevenValues) < 7 { b.err = ietools.ErrIllegalArguments; return make([]int, 0) }
  if sevenValues[0] <= 0 || sevenValues[2] <= 0 { b.err = ietools.ErrIllegalArguments; return make([]int, 0) }
  if sevenValues[1] != 2 && sevenValues[1] != 4 { b.err = ietools.ErrIllegalArguments; return make([]int, 0) }
  if sevenValues[3] < 1 || sevenValues[3] > 4 || sevenValues[3] == 3 { b.err = ietools.ErrIllegalArguments; return make([]int, 0) }
  if sevenValues[5] < 0 || sevenValues[5] > 4 || sevenValues[3] == 3 { b.err = ietools.ErrIllegalArguments; return make([]int, 0) }
  if sevenValues[6] <= 0 { b.err = ietools.ErrIllegalArguments; return make([]int, 0) }

  var ofs, cnt, idx int = 0, 0, 0
  switch sevenValues[1] {
    case 2: ofs = int(b.GetInt16(sevenValues[0]))
    case 4: ofs = int(b.GetInt32(sevenValues[0]))
  }
  switch sevenValues[3] {
    case 1: cnt = int(b.GetInt8(sevenValues[2]))
    case 2: cnt = int(b.GetInt16(sevenValues[2]))
    case 4: cnt = int(b.GetInt32(sevenValues[2]))
  }
  if sevenValues[4] > 0 {
    switch sevenValues[5] {
      case 1: idx = int(b.GetInt8(sevenValues[4]))
      case 2: idx = int(b.GetInt16(sevenValues[4]))
      case 4: idx = int(b.GetInt32(sevenValues[4]))
    }
  }

  var retVal []int = nil
  if ofs > 0 && cnt > 0 && cnt >= idx {
    size := sevenValues[6]
    retVal = make([]int, cnt - idx)
    for i := idx; i < cnt; i++ {
      retVal[i - idx] = ofs + i*size
    }
  }

  return retVal
}

// GetOffsetArray2 is a specialized method for retrieving offsets to all available substructures of a type specified
// by the arguments.
//
// It is most commonly used in conjunction with GetOffsetArray() to retrieve an extra parameter required for the
// function to work.
// Eight parameters are required. The order of parameters is as followed:
//  offset2             This is an offset to a specific substructure. It can be received by the
//                      GetOffsetArray() function.
//  ofs, ofsSize        The start offset and length of offset field to the list of substructures.
//                      Same as for GetOffsetArray().
//  count, countSize    The number of substructures and length of the count field. Same as for
//                      GetOffsetArray(), except that it's relative to ofs2.
//  index, indexSize    An optional start index and length of index field for the substructures.
//                      Set to 0 to ignore. Same as for GetOffsetArray(), except that it's relative
//                      to ofs2.
//  structSize          The size of a substructure in bytes. Must be non-zero. Same as for
//                      GetOffsetArray().
// Returns an array of offsets for each individual substructure found in the current buffer content.
// The package provides a number of predefined configurations for compatible structures.
// Operation is skipped if error state is set.
func (b *Buffer) GetOffsetArray2(offset2 int, sevenValues ...int) []int {
  if b.err != nil { return make([]int, 0) }
  if sevenValues == nil || len(sevenValues) < 7 { b.err = ietools.ErrIllegalArguments; return make([]int, 0) }
  if offset2 <= 0 { b.err = ietools.ErrIllegalArguments; return make([]int, 0) }

  var ofs, cnt int = sevenValues[0], offset2 + sevenValues[2]
  var idx int = 0
  if sevenValues[4] > 0 && sevenValues[5] > 0 { idx = offset2 + sevenValues[4] }
  return b.GetOffsetArray(ofs, sevenValues[1],
                          cnt, sevenValues[3],
                          idx, sevenValues[5],
                          sevenValues[6])
}
