/*
Package ietools provides a collection of types, constants and functions inspired by WeiDU.

More specific functionality can be found in the respective sub-packages:
  - package buffers:  Functions and types for manipulating data buffers.
  - package pvrz:     Functions and types for handling pvr/pvrz data.
  - package tables:   Functions and types for table-related operations.
*/
package ietools

import (
  "errors"
  "path"
  "strings"

  "golang.org/x/text/encoding/charmap"
)

const (
  // Helps addressing single bits in a numeric value
  BIT0  = 0x00000001
  BIT1  = 0x00000002
  BIT2  = 0x00000004
  BIT3  = 0x00000008
  BIT4  = 0x00000010
  BIT5  = 0x00000020
  BIT6  = 0x00000040
  BIT7  = 0x00000080
  BIT8  = 0x00000100
  BIT9  = 0x00000200
  BIT10 = 0x00000400
  BIT11 = 0x00000800
  BIT12 = 0x00001000
  BIT13 = 0x00002000
  BIT14 = 0x00004000
  BIT15 = 0x00008000
  BIT16 = 0x00010000
  BIT17 = 0x00020000
  BIT18 = 0x00040000
  BIT19 = 0x00080000
  BIT20 = 0x00100000
  BIT21 = 0x00200000
  BIT22 = 0x00400000
  BIT23 = 0x00800000
  BIT24 = 0x01000000
  BIT25 = 0x02000000
  BIT26 = 0x04000000
  BIT27 = 0x08000000
  BIT28 = 0x10000000
  BIT29 = 0x20000000
  BIT30 = 0x40000000
  BIT31 = 0x80000000
)

// Potential errors in addition to default Go package errors.
var (
  ErrOffsetOutOfRange = errors.New("Offset out of range")
  ErrIllegalArguments = errors.New("Illegal arguments specified")
)


// AnsiToUtf8 converts an ANSI-encoded byte array into an UTF-8 string with the provided character map.
// Provide a nil charmap to assume Windows-1252 encoding.
func AnsiToUtf8(buffer []byte, cm *charmap.Charmap) (string, error) {
  if buffer == nil || len(buffer) == 0 { return "", nil }

  if cm == nil { cm = charmap.Windows1252 }
  decoder := cm.NewDecoder()
  out, err := decoder.Bytes(buffer)
  if err != nil { return "", err }
  return string(out), nil
}

// Utf8ToAnsi converts an UTF-8 string into a byte array of the specified ANSI encoding.
// Provide a nil charmap to convert to Windows-1252 encoding.
func Utf8ToAnsi(text string, cm *charmap.Charmap) ([]byte, error) {
  if cm == nil { cm = charmap.Windows1252 }
  encoder := cm.NewEncoder()
  out, err := encoder.Bytes([]byte(text))
  if err != nil { return nil, err }
  return out, nil
}

// SplitFilePath splits the given path string and returns file path, base name and extension as separate values.
//
// Trailing path separators are stripped from the directory string, except for the root directory. Empty directory is returned as ".".
// Name returns the last path element regardless of whether it points to a file or folder.
// Ext returns the name part after the last period (.). Ext may be empty. Period is not included in either name or ext.
func SplitFilePath(filepath string) (dir, name, ext string) {
  if PATH_SEPARATOR != "/" {
    filepath = strings.Replace(filepath, PATH_SEPARATOR, "/", -1)
  }
  dir = path.Dir(filepath)
  name = path.Base(filepath)
  ext = path.Ext(name)
  if len(ext) > 0 {
    name = name[:len(name) - len(ext)]
    ext = ext[1:]
  }
  return
}

// AssembleFilePath returns a fully qualified path string based on the given path elements.
func AssembleFilePath(dir, name, ext string) string {
  retVal := strings.TrimSpace(dir)
  if len(retVal) == 0 { retVal = "." }
  if retVal[len(retVal)-1:] != "/" && retVal[len(retVal)-1:] != "\\" { retVal += "/" }
  retVal += strings.TrimSpace(name)
  ext = strings.TrimSpace(ext)
  if len(ext) > 0 {
    retVal += "." + ext
  }

  return retVal
}
