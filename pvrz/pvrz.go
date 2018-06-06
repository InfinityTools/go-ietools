/*
Package pvrz provides functionality to deal with data of the PVR and PVRZ formats. 
It has been optimized for use in Enhanced Edition games based on the Infinity Engine.
*/
package pvrz

import (
  "errors"
  "fmt"
  "image"
  "image/color"
  "image/draw"
  "io"

  "github.com/InfinityTools/go-squish"
  "github.com/InfinityTools/go-ietools/buffers"
)

const (
  // Supported texture flags
  FLAGS_PREMULTIPLIED = 2   // Whether texture pixels are premultiplied by alpha

  // Supported compression type
  TYPE_BC1            = 7   // aka DXT1, full decoding and encoding support
  TYPE_BC2            = 9   // aka DXT3, full decoding and encoding support
  TYPE_BC3            = 11  // aka DXT5, full decoding and encoding support

  // Available color space constants
  SPACE_LRGB          = 0   // linear RGB space (default)
  SPACE_SRGB          = 1   // standard RGB space

  // Available channel types
  CHAN_UBN            = 0   // Unsigned Byte Normalized (default)
  CHAN_SBN            = 1   // Signed Byte Normalized
  CHAN_UB             = 2   // Unsigned Byte
  CHAN_SB             = 3   // Signed Byte
  // Constants below are not supported
  // CHAN_USN            = 4   // Unsigned Short Normalized
  // CHAN_SSN            = 5   // Signed Short Normalized
  // CHAN_US             = 6   // Unsigned Short
  // CHAN_SS             = 7   // Signed Short
  // CHAN_UIN            = 8   // Unsigned Int Normalized
  // CHAN_SIN            = 9   // Signed Int Normalized
  // CHAN_UI             = 10  // Unsigned Int
  // CHAN_SI             = 11  // Signed Int
  // CHAN_F              = 12  // Float

  // Available pixel encoding quality modes
  QUALITY_LOW         = 0   // Encode with lowest possible quality
  QUALITY_DEFAULT     = 1   // Encode with a sensible quality/speed ratio
  QUALITY_HIGH        = 2   // Encoed with highest possiblle quality

  versionSig          = 0x03525650  // Internally used: the PVR signature
)

var ErrIllegalArguments = errors.New("Illegal arguments specified")

// Stores parsed PVR header information
type pvrInfo struct {
  flags         int
  pixelType     int
  colorSpace    int
  channelType   int
  height        int
  width         int
  depth         int
  numSurfaces   int
  numFaces      int
  numMipMaps    int
  meta          []byte
}

// The main PVR structure.
type Pvr struct {
  info          pvrInfo
  img           draw.Image  // the uncompressed RGBA pixel data

  err           error
  quality       int         // encoding quality setting (see QUALITY_xxx constants)
  weightByAlpha bool        // whether source uses weighted alpha (improves alpha-blended images)
  useMetric     bool        // whether to apply color weights to improve percepted quality
}


// CreateNew initializes a new Pvr object with an empty pixel buffer of specified dimension. 
//
// pixelType defines the pixel compression type applied when using the Save() function.
//
// Note: It is strongly recommended to use a dimension that is compatible with the desired pixel compression.
func CreateNew(width, height, pixelType int) *Pvr {
  if width < 1 { width = 1 }
  if height < 1 { height = 1 }

  p := Pvr{}
  p.info.flags = 0
  p.info.pixelType, p.info.colorSpace, p.info.channelType = pixelType, SPACE_LRGB, CHAN_UBN
  p.info.height, p.info.width = height, width
  p.info.depth, p.info.numSurfaces, p.info.numFaces, p.info.numMipMaps = 1, 1, 1, 1
  p.info.meta = make([]byte, 0)

  p.img = image.NewRGBA(image.Rect(0, 0, width, height))

  p.quality = QUALITY_DEFAULT
  p.weightByAlpha = false
  p.useMetric = false

  return &p
}


// Load loads PVR or PVRZ data from the specified Reader.
func Load(r io.Reader) *Pvr {
  p := CreateNew(0, 0, TYPE_BC1)

  buf := make([]byte, 1024)
  totalRead, bytesRead := 0, 0
  var err error
  for {
    bytesRead, err = r.Read(buf[totalRead:])
    totalRead += bytesRead
    if totalRead >= len(buf) {
      buf = append(buf, make([]byte, len(buf))...)
    }
    if err != nil {
      break
    }
  }
  if err != nil && err != io.EOF { p.err = err; return p }
  if len(buf) > totalRead {
    buf = buf[:totalRead]
  }

  p.importPvr(buf)
  return p
}


// Save sends PVR data to the specified Writer.
//
// Specify "compress" whether to write uncompressed PVR or compressed PVRZ data through the Writer.
// Note: Output texture dimension may be padded to meet pixel encoding requirements.
func (p *Pvr) Save(w io.Writer, compress bool) {
  if p.err != nil { return }

  data := p.exportPvr()
  if p.err != nil { return }

  buf := buffers.Wrap(data)
  if compress {
    pvrLen := buf.BufferLength()
    buf.CompressReplace(0, pvrLen, 9)
    buf.InsertBytes(0, 4)
    buf.PutInt32(0, int32(pvrLen))
  }
  w.Write(buf.Bytes())
}


// Error returns the error state of the most recent operation on Pvr.
// Use ClearError() function to clear the current error state.
func (p *Pvr) Error() error {
  return p.err
}


// ClearError clears the error state from the last Pvr operation.
// Must be called for subsequent operations to work correctly.
func (p *Pvr) ClearError() {
  p.err = nil
}


// SetImage replaces the current texture graphics with the specified image data.
//
// Note: It is strongly recommended to use images with dimensions supported by the desired pixel encoding type.
func (p *Pvr) SetImage(img image.Image) {
  if p.err != nil { return }
  if img == nil { p.err = ErrIllegalArguments; return }

  width, height := img.Bounds().Dx(), img.Bounds().Dy()
  imgOut := image.NewRGBA(image.Rect(0, 0, width, height))
  draw.Draw(imgOut, imgOut.Bounds(), img, img.Bounds().Min, draw.Src)
  p.info.width = width
  p.info.height = height
  p.img = imgOut
}


// GetImage returns a copy of the current texture graphics.
func (p *Pvr) GetImage() image.Image {
  if p.err != nil { return nil }

  imgOut := image.NewRGBA(image.Rect(0, 0, p.info.width, p.info.height))
  draw.Draw(imgOut, imgOut.Bounds(), p.img, p.img.Bounds().Min, draw.Src)
  return imgOut
}


// SetImageRect draws the content of "img" limited by the region "r" to the texture starting at position "dp".
func (p *Pvr) SetImageRect(img image.Image, r image.Rectangle, dp image.Point) {
  if p.err != nil { return }

  dr := image.Rectangle{dp, dp.Add(r.Size())}
  draw.Draw(p.img, dr, img, r.Min, draw.Src)
}


// GetImageRect returns the content of the texture in the specified region as a new Image object.
func (p *Pvr) GetImageRect(r image.Rectangle) image.Image {
  if p.err != nil { return nil }

  imgOut := image.NewRGBA(image.Rectangle{image.ZP, r.Size()})
  draw.Draw(imgOut, imgOut.Bounds(), p.img, r.Min, draw.Src)
  return imgOut
}


// FillImageRect fills the region "r" of the texture with the specified color.
func (p *Pvr) FillImageRect(r image.Rectangle, col color.Color) {
  if p.err != nil { return }

  draw.Draw(p.img, r, &image.Uniform{col}, image.ZP, draw.Src)
}


// GetWidth returns the width of the current pixel buffer in pixels.
func (p *Pvr) GetWidth() int {
  if p.err != nil { return 0 }
  return p.info.width
}


// GetHeight returns the height of the current pixel buffer in pixels.
func (p *Pvr) GetHeight() int {
  if p.err != nil { return 0 }
  return p.info.height
}


// SetDimension can be used to resize the current pixel buffer. Specify "preserve" to preserve as much of old content if possible.
func (p *Pvr) SetDimension(width, height int, preserve bool) {
  if p.err != nil { return }
  if width == p.info.width && height == p.info.height && preserve { return }
  if width < 1 { p.err = ErrIllegalArguments; return }
  if height < 1 { p.err = ErrIllegalArguments; return }

  imgNew := resizeCanvas(p.img, width, height, preserve)
  if imgNew == nil { p.err = ErrIllegalArguments; return }
  p.info.width = imgNew.Bounds().Dx()
  p.info.height = imgNew.Bounds().Dy()
  p.img = imgNew

}


// GetPixelType returns the currently assigned pixel compression type applied when using the Save() function.
func (p *Pvr) GetPixelType() int {
  return p.info.pixelType
}


// SetPixelType sets the pixel compression type that is applied when using the Save() function.
func (p *Pvr) SetPixelType(pixelType int) {
  if p.err != nil { return }
  if !pixelTypeSupported(pixelType) { p.err = ErrIllegalArguments; return }

  p.info.pixelType = pixelType
}


// GetChannelType returns the size of individual pixel elements (r, g, b, a).
// Currently only byte-sized channel types are supported (see CHAN_xxx constants).
func (p *Pvr) GetChannelType() int {
  if p.err != nil { return 0 }
  return p.info.channelType
}


// SetChannelType defines the size of individual pixel elements (r, g, b, a).
// Currently only byte-sized channel types are supported (see CHAN_xxx constants).
func (p *Pvr) SetChannelType(channelType int) {
  if p.err != nil { return }
  if channelType < CHAN_UBN || channelType > CHAN_SB { p.err = ErrIllegalArguments; return }

  p.info.channelType = channelType
}


// GetColorSpace returns the color space used to represent pixel data. (see SPACE_xxx constants).
func (p *Pvr) GetColorSpace() int {
  if p.err != nil { return 0 }
  return p.info.colorSpace
}


// SetColorSpace defines the the color space used to represent pixel data. (see SPACE_xxx constants).
func (p *Pvr) SetColorSpace(colorSpace int) {
  if p.err != nil { return }
  if colorSpace != SPACE_LRGB && colorSpace != SPACE_SRGB { p.err = ErrIllegalArguments; return }

  p.info.colorSpace = colorSpace
}


// GetQuality returns the quality mode applied to pixel compression (see QUALITY_xxx constants).
func (p *Pvr) GetQuality() int {
  if p.err != nil { return 0 }
  return p.quality
}


// SetQuality defines the quality mode applied to pixel compression. Use one of the QUALITY_xxx constants.
func (p *Pvr) SetQuality(q int) {
  if p.err != nil { return }
  if q < QUALITY_LOW { q = QUALITY_LOW }
  if q > QUALITY_HIGH { q = QUALITY_HIGH }
  p.quality = q
}


// GetWeightByAlpha indicates whether pixel values are weighted by their alpha component when performing compression.
func (p *Pvr) GetWeightByAlpha() bool {
  if p.err != nil { return false }
  return p.weightByAlpha
}


// SetWeightByAlpha defines whether pixel values are weighted by their alpha component when performing compression.
func (p *Pvr) SetWeightByAlpha(set bool) {
  if p.err != nil { return }
  p.weightByAlpha = set
}


// IsPerceptiveMetric returns whether a perceptive metric is applied to pixel compression.
func (p *Pvr) IsPerceptiveMetric() bool {
  if p.err != nil { return false }
  return p.useMetric
}


// SetPerceptiveMetric defines whether a perceptive metric is applied to pixel compression.
func (p *Pvr) SetPerceptiveMetric(set bool) {
  if p.err != nil { return }
  p.useMetric = set
}


// Used internally. Returns whether the specified pixel format is supported by this package.
func pixelTypeSupported(value int) bool {
  switch value {
  case TYPE_BC1, TYPE_BC2, TYPE_BC3:
    return true
  default:
    return false
  }
}


// Used internally. Imports PVR or PVRZ data from the specified byte array. The function attempts to determine right format automatically.
func (p *Pvr) importPvr(data []byte) {
  if data == nil { p.err = errors.New("No input buffer specified"); return }

  buf := buffers.Wrap(data)
  if buf.Error() != nil { p.err = buf.Error(); return }
  if buf.BufferLength() < 4 { p.err = errors.New("Input buffer too small"); return }

  sig := int(buf.GetInt32(0))
  if sig != versionSig {
    // simply consistency check
    if sig < 0x34 || sig > (1 << 25) { p.err = fmt.Errorf("PVR target size outside of accepted limits: %d", sig); return }
    // try decompressing PVRZ
    buf.DecompressReplace(4, buf.BufferLength() - 4)
    if buf.Error() != nil { p.err = buf.Error(); return }

    buf.DeleteBytes(0, 4)
    if buf.BufferLength() < sig { p.err = fmt.Errorf("PVRZ data size mismatch: %d != %d", buf.BufferLength(), sig); return }
    if buf.BufferLength() > sig {
      buf.DeleteBytes(sig, buf.BufferLength() - sig)
    }
    sig = int(buf.GetInt32(0))
  }

  // parsing PVR header
  if sig != versionSig { p.err = fmt.Errorf("Invalid PVR header signature: %08x", sig); return }
  if buf.BufferLength() < 0x34 { p.err = fmt.Errorf("PVR input buffer too small"); return }
  flags := int(buf.GetInt32(0x04))
  pf := int(buf.GetInt32(0x0c))
  if pf != 0 { p.err = fmt.Errorf("Extended pixel format not supported"); return }
  pixelType := int(buf.GetInt32(0x08))
  if !pixelTypeSupported(pixelType) { p.err = fmt.Errorf("Unsupported pixel format: %d", pixelType); return }
  colorSpace := int(buf.GetInt32(0x10))
  if colorSpace < 0 || colorSpace > 1 { p.err = fmt.Errorf("Unsupported color space: %d", colorSpace); return }
  channelType := int(buf.GetInt32(0x14))
  if channelType < CHAN_UBN || channelType > CHAN_SB { p.err = fmt.Errorf("Unsupported channel type: %d", channelType); return }
  height := int(buf.GetInt32(0x18))
  if height < 0 || height > 4096 { p.err = fmt.Errorf("Unsupported texture height: %d", height); return }
  if (height & 3) != 0 { p.err = errors.New("Texture height must be a multiple of 4"); return }
  width := int(buf.GetInt32(0x1c))
  if width < 0 || width > 4096 { p.err = fmt.Errorf("Unsupported texture width: %d", width); return }
  if (width & 3) != 0 { p.err = errors.New("Texture width must be a multiple of 4"); return }
  depth := int(buf.GetInt32(0x20))
  if depth != 1 { p.err = fmt.Errorf("Unsupported texture depth: %d", depth); return }
  numSurfaces := int(buf.GetInt32(0x24))
  if numSurfaces != 1 { p.err = fmt.Errorf("Unsupported number of texture surfaces: %d", numSurfaces); return }
  numFaces := int(buf.GetInt32(0x28))
  if numFaces != 1 { p.err = fmt.Errorf("Unsupported number of texture faces: %d", numFaces); return }
  numMipMaps := int(buf.GetInt32(0x2c))
  if numMipMaps != 1 { p.err = fmt.Errorf("Unsupported number of texture mip maps: %d", numMipMaps); return }
  metaLen := int(buf.GetInt32(0x30))
  if metaLen < 0 { metaLen = 0 }
  if buf.BufferLength() < 0x34 + metaLen { p.err = errors.New("Metadata size mismatch"); return }
  var meta []byte
  if metaLen > 0 {
    meta = buf.GetBuffer(0x34, metaLen)
  } else {
    meta = make([]byte, 0)
  }

  // importing texture data
  ofsData := 0x34 + metaLen
  dxtFlags := 0
  switch pixelType {
    case TYPE_BC1: dxtFlags = squish.FLAGS_DXT1
    case TYPE_BC2: dxtFlags = squish.FLAGS_DXT3
    case TYPE_BC3: dxtFlags = squish.FLAGS_DXT5
  }
  texSize := squish.GetStorageRequirements(width, height, dxtFlags)
  if buf.BufferLength() - ofsData < texSize { p.err = fmt.Errorf("PVR input buffer too small"); return }
  img := decodeTexture(buf.Bytes()[ofsData:], width, height, pixelType)
  if img == nil { p.err = errors.New("Error while decoding texture data"); return }

  p.info.flags = flags
  p.info.pixelType = pixelType
  p.info.colorSpace = colorSpace
  p.info.channelType = channelType
  p.info.height, p.info.width, p.info.depth = height, width, depth
  p.info.numSurfaces, p.info.numFaces, p.info.numMipMaps = numSurfaces, numFaces, numMipMaps
  p.info.meta = meta
  p.img = img
}

// Used internally. Creates a bye buffer containing PVR data.
func (p *Pvr) exportPvr() []byte {
  hdr := p.prepareHeader()
  out := encodeTexture(p.img, p.info.pixelType, p.quality, p.weightByAlpha, p.useMetric)
  if out == nil { p.err = errors.New("Unable to encode texture data"); return nil }
  buf := make([]byte, len(hdr) + len(out))
  copy(buf[:len(hdr)], hdr)
  copy(buf[len(hdr):], out)

  return buf
}


// Used internally. Returns current PVR header as byte array.
func (p *Pvr) prepareHeader() []byte {
  buf := buffers.Create()
  buf.InsertBytes(0, 0x34)
  buf.PutInt32(0x00, versionSig)
  buf.PutInt32(0x04, int32(p.info.flags))
  buf.PutInt32(0x08, int32(p.info.pixelType))
  buf.PutInt32(0x0c, 0)   // extended pixel format bits not used
  buf.PutInt32(0x10, int32(p.info.colorSpace))
  buf.PutInt32(0x14, int32(p.info.channelType))
  buf.PutInt32(0x18, int32(p.info.height))
  buf.PutInt32(0x1c, int32(p.info.width))
  buf.PutInt32(0x20, int32(p.info.depth))
  buf.PutInt32(0x24, int32(p.info.numSurfaces))
  buf.PutInt32(0x28, int32(p.info.numFaces))
  buf.PutInt32(0x2c, int32(p.info.numMipMaps))
  buf.PutInt32(0x30, int32(len(p.info.meta)))
  if len(p.info.meta) > 0 {
    buf.InsertBytes(0x34, len(p.info.meta))
    buf.PutBuffer(0x34, p.info.meta)
  }
  return buf.Bytes()
}

// Used internally. Decodes raw PVR texture data into 32-bt ARGB pixels.
func decodeTexture(data []byte, width, height, pixelType int) draw.Image {
  if width <= 0 || height <= 0 || data == nil { return nil }

  flags := squish.FLAGS_SOURCE_BGRA
  switch pixelType {
    case TYPE_BC1: flags |= squish.FLAGS_DXT1
    case TYPE_BC2: flags |= squish.FLAGS_DXT3
    case TYPE_BC3: flags |= squish.FLAGS_DXT5
    default: return nil
  }
  img := squish.DecompressImage(width, height, data, flags)
  imgOut, ok := img.(draw.Image)
  if !ok {
    imgOut = image.NewRGBA(image.Rectangle{image.ZP, img.Bounds().Size()})
    draw.Draw(imgOut, imgOut.Bounds(), img, img.Bounds().Min, draw.Src)
  }
  return imgOut
}


// Used internally. Encodes 32-bit ARGB pixel data into the specified texture compression format.
// Set "quality" to the desired quality setting.
// Set "useMetric" to use perceptive color weights which may improve visual quality.
func encodeTexture(img image.Image, pixelType, quality int, weightByAlpha, useMetric bool) []byte {
  width, height := img.Bounds().Dx(), img.Bounds().Dy()
  if width < 1 || width & 3 != 0 || height < 1 || height & 3 != 0 { return nil }
  if quality < QUALITY_LOW { quality = QUALITY_LOW }
  if quality > QUALITY_HIGH { quality = QUALITY_HIGH }

  newWidth, newHeight := width, height
  flags := 0
  switch pixelType {
    case TYPE_BC1:
      flags |= squish.FLAGS_DXT1
      newWidth = (newWidth + 3) & ^3
      newHeight = (newHeight + 3) & ^3
    case TYPE_BC2:
      flags |= squish.FLAGS_DXT3
      newWidth = (newWidth + 3) & ^3
      newHeight = (newHeight + 3) & ^3
    case TYPE_BC3:
      flags |= squish.FLAGS_DXT5
      newWidth = (newWidth + 3) & ^3
      newHeight = (newHeight + 3) & ^3
    default: return nil
  }

  if newWidth != width || newHeight != height {
    img = resizeCanvas(img, newWidth, newHeight, true)
    if img == nil { return nil }
    width = newWidth
    height = newHeight
  }

  switch quality {
    case QUALITY_LOW:
      flags |= squish.FLAGS_RANGE_FIT
    case QUALITY_HIGH:
      flags |= squish.FLAGS_ITERATIVE_CLUSTER_FIT
    default:
      flags |= squish.FLAGS_CLUSTER_FIT
  }

  if weightByAlpha {
    flags |= squish.FLAGS_WEIGHT_BY_ALPHA
  }

  var metric []float32 = nil
  if useMetric {
    metric = squish.METRIC_PERCEPTUAL
  } else {
    metric = squish.METRIC_UNIFORM
  }

  imgOut := image.NewNRGBA(img.Bounds())
  draw.Draw(imgOut, imgOut.Bounds(), img, img.Bounds().Min, draw.Src)
  data := squish.CompressImage(imgOut, flags, metric)

  return data
}

// Used internally. Resizes the given image object. Optionally preserve as much of old content as possible.
func resizeCanvas(img image.Image, width, height int, preserve bool) draw.Image {
  imgNew := image.NewRGBA(image.Rect(0, 0, width, height))
  if preserve {
    draw.Draw(imgNew, imgNew.Bounds(), img, img.Bounds().Min, draw.Src)
  }
  return imgNew
}
