/*
Package tables provides functions for dealing with table-like structures, such as used by 2DA or IDS resource types.
*/
package tables

import (
  "bytes"
  "io"
  "strconv"
  "strings"
  "regexp"

  "github.com/InfinityTools/go-ietools"
  "golang.org/x/text/encoding/charmap"
)

// Table contains the necessary information to query or alter table data.
type Table struct {
  table [][]string        // a two-dimensional array[row][col] to store table data
  cmap  *charmap.Charmap  // the character map to be used for ANSI decoding or encoding
  dirty bool              // true if content has been modified
  err   error
}

// Load uses the given Reader to load table data from the underlying buffer. The function returns a pointer to the Table object.
//
// This function assumes that input data is encoded in ANSI Windows-1252.
// Use function Error to check if the Load function returned successfully.
func Load(r io.Reader) *Table {
  return LoadEx(r, charmap.Windows1252)
}

// LoadEx uses the given Reader to load table data from the underlying buffer, using the specified character map for ANSI decoding. 
//
// Specify a nil charmap to skip the decoding operation. The function returns a pointer to the Table object.
// Use function Error to check if the Load function returned successfully.
func LoadEx(r io.Reader, cmap *charmap.Charmap) *Table {
  table := Table { nil, nil, false, nil }

  buf := make([]byte, 1024)
  totalRead, bytesRead := 0, 0
  var err error
  for {
    bytesRead, err = r.Read(buf[totalRead:])
    totalRead += bytesRead
    if err != nil { break }
    if totalRead >= len(buf) {
      buf = append(buf, make([]byte, len(buf))...)
    }
  }
  if err != nil && err != io.EOF { table.err = err; return &table }
  if totalRead < len(buf) { buf = buf[:totalRead] }
  table.table = importTable(buf, cmap)
  table.cmap = cmap
  return &table
}

// Save writes the current table content to the specified Writer, encoding text as specified by the Load function.
//
// Does nothing if the Table is in an invalid state (see Error function).
// Set prettify to ensure that table data is properly aligned.
func (t *Table) Save(w io.Writer, prettify bool) {
  t.SaveEx(w, t.cmap, prettify)
}

// SaveEx writes the current table content to the specified Writer, using the specified character map for ANSI encoding. 
//
// Specify a nil charmap to skip the encoding operation. Does nothing if the Table is in an invalid state (see Error function).
// Set prettify to ensure that table data is properly aligned.
func (t *Table) SaveEx(w io.Writer, cmap *charmap.Charmap, prettify bool) {
  if t.err != nil { return }

  _, err := w.Write(t.exportTable(true, prettify, cmap))
  if err != nil { t.err = err; return }
  t.dirty = false
}


// Error returns the error state of the most recent operation on Table. Use ClearError function to clear the current error state.
func (t *Table) Error() error {
  return t.err
}

// ClearError clears the error state from the last Table operation. Must be called for subsequent operations to work correctly.
func (t *Table) ClearError() {
  t.err = nil
}

// IsModified returns whether the current table content has been modified by a previous operation.
// The return value is only provided for informal purposes. None of the Table functions rely on it.
func (t *Table) IsModified() bool {
  return t.dirty
}

// ClearModified explicitly marks the Table object as unmodified.
func (t *Table) ClearModified() {
  t.dirty = false
}


// Columns returns the maximum number of columns available for the table.
// Operation is skipped if error state is set.
func (t *Table) Columns() int {
  if t.err != nil { return 0 }

  numCols := 0
  for _, v := range t.table {
    if numCols < len(v) { numCols = len(v) }
  }
  return numCols
}

// RowColumns returns the number of columns of the specified row that has minCols or more items.
// Returns -1 if row doesn't exist.
// Operation is skipped if error state is set.
func (t *Table) RowColumns(row, minCols int) int {
  if t.err != nil { return 0 }

  r := t.absoluteRow(row, minCols)
  if r < 0 { return -1 }
  return len(t.table[r])
}

// Rows returns the number of rows containing at least "cols" number of columns.
// Operation is skipped if error state is set.
func (t *Table) Rows(cols int) int {
  if t.err != nil { return 0 }

  if cols < 0 { cols = 0 }
  numRows := 0
  if (cols > 0) {
    for _, v := range t.table {
      if len(v) >= cols {
        numRows++
      }
    }
  } else {
    numRows = len(t.table)
  }
  return numRows
}

// Is2DA returns true only if the current content conforms to the 2DA table format.
// Operation is skipped if error state is set.
func (t *Table) Is2DA() bool {
  if t.err != nil { return false }
  if t.Columns() < 2 { return false }
  if t.Rows(0) < 2 { return false }

  if strings.ToUpper(t.table[0][0]) != "2DA" ||
     strings.ToUpper(t.table[0][1]) != "V1.0" ||
     len(t.table[1]) != 1 {
     return false
   }
   return true
}

// IsIDS returns true only if the current content conforms to the IDS table format.
// Operation is skipped if error state is set.
func (t *Table) IsIDS() bool {
  if t.err != nil { return false }
  if t.Columns() != 2 { return false }
  if t.Rows(0) == 0 { return false }
  if t.Rows(0) > 1 {
    // first column may only contain numbers
    for row := 1; row < len(t.table); row++ {
      if len(t.table[row]) < 2 { return false }
      isHex := len(t.table[row][0]) > 2 && strings.ToLower(t.table[row][0][:2]) == "0x"
      var err error = nil
      if isHex {
        _, err = strconv.ParseInt(t.table[row][0][2:], 16, 32)
      } else {
        _, err = strconv.ParseInt(t.table[row][0], 10, 32)
      }
      if err != nil { return false }
    }
  }
  return true
}

// GetItem returns the item at [row, col] for rows containing minCols or more items.
//
// Sets t.err and returns empty item if [row, col] doesn't point to a valid location.
// Operation is skipped if error state is set.
func (t *Table) GetItem(row, col, minCols int) string {
  if t.err != nil { return "" }
  if row < 0 || col < 0 { t.err = ietools.ErrIllegalArguments; return "" }

  if minCols < 0 { minCols = 0 }
  row = t.absoluteRow(row, minCols)
  if row < 0 || col >= len(t.table[row]) { t.err = ietools.ErrIllegalArguments; return "" }
  return t.table[row][col]
}

// PutItem assigns item to the existing table location [row, col] for rows containing minCols or more items.
//
// Sets t.err if the specified location does not exist or item is empty.
// Operation is skipped if error state is set.
// Hint: Use DeleteItem to remove individual items. Use InsertItem to insert a new item.
func (t *Table) PutItem(row, col, minCols int, item string) {
  if t.err != nil { return }
  item = strings.TrimSpace(item)
  if row < 0 || col < 0 || len(item) == 0 { t.err = ietools.ErrIllegalArguments; return }

  if minCols < 0 { minCols = 0 }
  row = t.absoluteRow(row, minCols)
  if row < 0 || col >= len(t.table[row]) { t.err = ietools.ErrIllegalArguments; return }
  if t.table[row][col] != item {
    t.dirty = true
  }
  t.table[row][col] = item
}

// InsertItem inserts a new item with the specified string at [row, col] for rows containing minCols or more items.
//
// Sets t.err and skips insertion if location does not exist or item is empty.
// Operation is skipped if error state is set.
func (t *Table) InsertItem(row, col, minCols int, item string) {
  if t.err != nil { return }
  item = strings.TrimSpace(item)
  if row < 0 || col < 0 || len(item) == 0 { t.err = ietools.ErrIllegalArguments; return }

  if minCols < 0 { minCols = 0 }
  row = t.absoluteRow(row, minCols)
  if row < 0 || col > len(t.table[row]) { t.err = ietools.ErrIllegalArguments; return }

  t.table[row] = append(t.table[row], "")
  for c := len(t.table[row]) - 1; c > col; c-- {
    t.table[row][c] = t.table[row][c - 1]
  }
  t.table[row][col] = item
  t.dirty = true
}

// DeleteItem removes the table item at [row, col] for rows containing minCols or more items and returns the removed item.
//
// As a result the number of columns for this row will decrease by one. Sets t.err if the specified location does not exist.
// Operation is skipped if error state is set.
func (t *Table) DeleteItem(row, col, minCols int) string {
  if t.err != nil { return "" }
  if row < 0 || col < 0 { t.err = ietools.ErrIllegalArguments; return "" }

  if minCols < 0 { minCols = 0 }
  row = t.absoluteRow(row, minCols)
  if row < 0 || col >= len(t.table[row]) { t.err = ietools.ErrIllegalArguments; return "" }

  retVal := t.table[row][col]
  for c := col + 1; c < len(t.table[row]); c++ {
    t.table[row][c] = t.table[row][c - 1]
  }
  t.table[row] = t.table[row][:len(t.table[row]) - 1]
  t.dirty = true
  return retVal
}

// InsertRow inserts a new table row and fills it with the specified items. No row is inserted if items doesn't contain non-empty items.
//
// rowIndex must be in range 0 to Rows(0) inclusive.
// Operation is skipped if error state is set.
func (t *Table) InsertRow(rowIndex int, items []string) {
  if t.err != nil { return }
  if items == nil || len(items) == 0 { return }

  t.table = append(t.table, make([]string, 0))
  for row := len(t.table) - 1; row > rowIndex; row-- {
    t.table[row] = t.table[row - 1]
  }

  // add only non-empty items
  t.table[rowIndex] = make([]string, 0)
  for _, v := range items {
    v = strings.TrimSpace(v)
    if len(v) > 0 {
      t.table[rowIndex] = append(t.table[rowIndex], v)
    }
  }

  t.dirty = true
}

// InsertRowString inserts a new table row and fills it with the items extracted from the given string.
//
// No row is inserted if line doesn't contain non-empty items.
// rowIndex must be in range 0 to Rows(0) inclusive.
// Operation is skipped if error state is set.
func (t *Table) InsertRowString(rowIndex int, line string) {
  if t.err != nil { return }
  if len(line) == 0 { return }

  items, _ := importRow([]byte(line), 0, nil)
  t.InsertRow(rowIndex, items)
}

// InsertRowString inserts a new table row and fills it with the items extracted from the given byte array.
//
// No row is inserted if buffer doesn't contain non-empty items.
// rowIndex must be in range 0 to Rows(0) inclusive.
// Operation is skipped if error state is set.
func (t *Table) InsertRowBuffer(rowIndex int, buffer []byte) {
  if t.err != nil { return }
  if buffer == nil || len(buffer) == 0 { return }

  items, _ := importRow(buffer, 0, charmap.Windows1252)
  t.InsertRow(rowIndex, items)
}

// DeleteRow removes the specified row of data. rowIndex must be in range 0 to Rows(0) exclusive.
// Operation is skipped if error state is set.
func (t *Table) DeleteRow(rowIndex int) {
  if t.err != nil { return }
  if rowIndex < 0 || rowIndex >= len(t.table) { t.err = ietools.ErrIllegalArguments; return }

  for row := rowIndex + 1; row < len(t.table); row++ {
    t.table[row - 1] = t.table[row]
  }
  t.table = t.table[:len(t.table) - 1]
  t.dirty = true
}


// Used internally. Returns the absolute table row based on row and minCols. Returns -1 if desired row does not exist.
func (t *Table) absoluteRow(row, minCols int) int {
  if minCols < 0 { minCols = 0 }
  if row >= 0 && row < len(t.table) {
    for r, match := 0, 0; r < len(t.table); r++ {
      if len(t.table[r]) >= minCols {
        if match == row {
          return r
        }
        match++
      }
    }
  }
  return -1
}

// Used internally. Parses a raw stream of bytes into a two-dimensional string array of rows and columns.
// data contains the raw stream of text. cm is used to convert ANSI into UTF-8. Specify nil to skip conversion.
// Note: This parser will turn anything into a table representation.
func importTable(data []byte, cm *charmap.Charmap) [][]string {
  table := make([][]string, 0)
  if data == nil { return table }

  for pos := 0; pos < len(data); {
    var line []string
    line, pos = importRow(data, pos, cm)
    if len(line) > 0 {
      table = append(table, line)
    }
  }

  return table
}

// Used internally. Parses a single row of table data and returns it as a string array.
// data contains the raw stream of text. cm is used to convert ANSI into UTF-8. Specify nil to skip conversion.
func importRow(data []byte, startPos int, cm *charmap.Charmap) (line []string, newPos int) {
  line = make([]string, 0)
  newPos = startPos
  if data == nil { return }

  // available parser modes
  const MODE_EMPTY = 0  // indicates end of line
  const MODE_SPACE = 1  // parsing whitespace between/after row items
  const MODE_TOKEN = 2  // parsing non-whitespace data

  // initializing matchers
  regNewline := regexp.MustCompile("[\f\n\r\v]")        // separator for rows
  regSpace := regexp.MustCompile("[\a\b\t ]")           // separator for columns
  regToken := regexp.MustCompile("[^\f\n\r\v\a\b\t ]")  // textual content includes everything not covered by regNewline and regSpace

  mode := MODE_SPACE
  curCol, posToken := 0, -1
  for pos := startPos; pos < len(data) && mode != MODE_EMPTY; pos++ {
    switch mode {
    case MODE_SPACE:
      if regToken.Match(data[pos:pos+1]) {
        posToken = pos
        mode = MODE_TOKEN
      } else if regNewline.Match(data[pos:pos+1]) {
        newPos = pos
        mode = MODE_EMPTY
      }
    case MODE_TOKEN:
      if regSpace.Match(data[pos:pos+1]) {
        var s string
        var err error = nil
        if cm != nil {
          s, err = ietools.AnsiToUtf8(data[posToken:pos], cm)
        }
        if cm == nil || err != nil {
          s = string(data[posToken:pos])
        }
        if curCol >= len(line) {
          line = append(line, s)
        }
        posToken = -1
        curCol++
        mode = MODE_SPACE
      } else if regNewline.Match(data[pos:pos+1]) {
        newPos = pos
        mode = MODE_EMPTY
      }
    }
  }

  if mode != MODE_EMPTY {
    newPos = len(data)
  }

  // dealing with pending data
  if posToken >= 0 {
    var s string
    var err error = nil
    if cm != nil {
      s, err = ietools.AnsiToUtf8(data[posToken:newPos], cm)
    }
    if cm == nil || err != nil {
      s = string(data[posToken:newPos])
    }
    if curCol >= len(line) {
      line = append(line, s)
    }
  }

  // prevent deadlocks in parent function
  if mode == MODE_EMPTY {
    newPos++
  }

  return
}

// Used internally. Converts the current table content into a raw stream of bytes.
// UseWinBreak indicates whether to use Windows-style line breaks (\r\n) or Unix-style line breaks (\n).
// cm is used to convert UTF-8 into ANSI. Specify nil to skip conversion.
func (t *Table) exportTable(useWinBreak, prettify bool, cm *charmap.Charmap) []byte {
 var nl []byte
  if useWinBreak { nl = []byte{0x0d, 0x0a} } else { nl = []byte{0x0a} }

  // calculating minimum column widths
  is2DA := t.Is2DA()
  colWidths := make([]int, t.Columns())
  maxWidth := 0
  for col := 0; col < t.Columns(); col++ {
    minW := 0
    if prettify {
      for row := 0; row < t.Rows(0); row++ {
        if col < len(t.table[row]) {
          shift := 0  // special: headers are shifted right by one in 2DA tables
          if is2DA && row == 2 {
            if col == 0 { continue }
            shift = 1
          }
          if len(t.table[row][col-shift]) > minW {
            minW = len(t.table[row][col-shift])
          }
        }
      }
      // align to even position
      minW = (minW + 3) & ^1
    }
    colWidths[col] = minW
    if minW > maxWidth { maxWidth = minW }
  }
  spaces := make([]byte, maxWidth + 1)
  for i := 0; i < len(spaces); i++ { spaces[i] = 0x20 }

  var buf bytes.Buffer
  if len(t.table) > 0 {
    for row := 0; row < len(t.table); row++ {
      shift := 0  // special: headers are shifted right by one in prettified 2DA tables
      if row == 2 && is2DA && prettify {
        buf.Write(spaces[:colWidths[0]])
        shift = 1
      }

      for col := 0; col < len(t.table[row]); col++ {
        var item []byte
        var err error = nil
        if cm != nil {
          item, err = ietools.Utf8ToAnsi(t.table[row][col], cm)
        }
        if cm == nil || err != nil {
          item = []byte(t.table[row][col])
        }
        if len(item) > 0 {
          buf.Write(item)
          if col + 1 < len(t.table[row]) {
            width := len(item) + 1
            if colWidths[col + shift] > width { width = colWidths[col + shift] }
            buf.Write(spaces[:width - len(item)])
          }
        }
      }
      buf.Write(nl)
    }
  } else {
    buf.Write(nl)
  }

  retVal := make([]byte, buf.Len())
  copy(retVal, buf.Bytes())
  return retVal
}
