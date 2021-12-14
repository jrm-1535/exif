// support for EXIF metadata parsing, removing and serializing
package exif

import (
    "fmt"
    "bytes"
    "encoding/binary"
    "io"
)

/*
    EXIF metadata layout:

    Exif header:
      "Exif\x00\x00"            Fixed 6-byte header

    TIFF header:    Note this is the origin of following offsets
      "II" | "MM"               2-byte endianess (Intel LE/Motorola BE)
                                All following multi-byte values depend on endianess
      0x002a                    2-byte Magic Number
      0x00000008                4-byte offset of immediately following primary IFD

    IFD0:           Primary Image Data
    IFD1:           Thumbnail Image Data (optional)

    An IFD has the following layout
      <n>                       2-byte (_UnsignedShort) count of following entries 
      { IFD entry } * n         12-byte entry
      <offset next IFD>         4-byte offset of Thumbnail IFD
      <IFD data>                variable length data area for values pointed to
                                by entries, for embedded IFDs (EXIF and/or GPS),
                                or for an embedded JPEG thumbnail image.

    Each IFD entry is:
        entry Tag               2-byte unique tag (tTag)
        entry type              2-byte TIFF tag (tType)
        value count             4-byte count of values:
                                    number of items in an array, or number of
                                    bytes in an ASCII string.
        value or value offset   4-byte data:
                                    if the value fits in 4 bytes, the value is
                                    directly in the entry, otherwise the entry
                                    value contains the offset in the IFD data
                                    where the value is located.
*/

const (
    _originOffset = 6           // TIFF header offset in EXIF file
    _headerSize   = 8           // TIFF header size
    _valOffSize   = 4           // value fits if < 4 bytes, otherwise offset
    _IfdEntrySize = ( _ShortSize + _LongSize) * 2
)

type Compression uint

const (
    Undefined Compression = iota
    NotCompressed
    CCITT_1D
    CCITT_Group3
    CCITT_Group4
    LZW
    JPEG
    JPEG_Technote2
    Deflate
    RFC_2301_BW_JBIG
    RFC_2301_Color_JBIG
    PackBits
)

type Control struct {
    Print bool
}

type Desc struct {
    data    []byte          // starts at TIFF header (origin after exif header)
    endian  binary.ByteOrder // endianess as defined in binary

    tType   Compression     // Thumbnail information if any
    tOffset uint32          // Thumbnail JPEG SOI offset, if jpeg thumbnail
    tLen    uint32          // Thumbnail after JPEG EOD, if jpeg thumbnail

            control         // what to do

    root    *ifdd           // storage for rewriting exif metadata
}

type control struct {
            Control         // to keep Desc fully opaque
}

func (d *Desc) readTIFFData( offset uint32, dest interface{} ) {
    b := bytes.NewBuffer( d.data[offset:] )
    binary.Read( b, d.endian, dest )
    return
}

// (d *Desc)get<tType>s(offset, count) functions read the requested count of
// typed data from an offset anywhere in the data slice, using the endianess
// and data slice from d. The result is a slice of the corresponding go type.

func (d *Desc) getASCIIString( offset, count uint32 ) string {
    // make sure terminating 0 (in count) is also included in the go string
    return string( d.data[offset:offset+count] )
}

func (d *Desc) getByte( offset uint32 ) uint8 {
    return d.data[offset]
}

func (d *Desc) getUnsignedBytes( offset, count uint32 ) []uint8 {
    return d.data[offset:offset+count]
}

func (d *Desc) getUnsignedShort( offset uint32 ) (us uint16) {
    b := bytes.NewBuffer( d.data[offset:offset+_ShortSize] )
    binary.Read( b, d.endian, &us )
    return
}

func (d *Desc) getUnsignedShorts( offset, count uint32 ) []uint16 {
    r := make( []uint16, count )
    d.readTIFFData( offset, &r )
    return r
}

func (d *Desc) getUnsignedLong( offset uint32 ) (ul uint32) {
    b := bytes.NewBuffer( d.data[offset:offset+_LongSize] )
    binary.Read( b, d.endian, &ul )
    return
}

func (d *Desc) getUnsignedLongs( offset, count uint32 ) []uint32 {
    r := make( []uint32, count )
    d.readTIFFData( offset, &r )
    return r
}

func (d *Desc) getSignedLong( offset uint32 ) (l int32) {
    b := bytes.NewBuffer( d.data[offset:offset+_LongSize] )
    binary.Read( b, d.endian, &l )
    return
}

func (d *Desc) getSignedLongs( offset, count uint32 ) []int32 {
    r := make( []int32, count )
    d.readTIFFData( offset, &r )
    return r
}

func (d *Desc) getUnsignedRational( offset uint32 ) unsignedRational {
    var r unsignedRational
    d.readTIFFData( offset, &r )
    return r
}

func (d *Desc) getUnsignedRationals( offset, count uint32 ) []unsignedRational {
    r := make( []unsignedRational, count )
    d.readTIFFData( offset, &r )
    return r
}

func (d *Desc) getSignedRational( offset uint32 ) signedRational {
    var r signedRational
    d.readTIFFData( offset, &r )
    return r
}
func (d *Desc) getSignedRationals( offset, count uint32 ) []signedRational {
    r := make( []signedRational, count )
    d.readTIFFData( offset, &r )
    return r
}

type ifdId  uint
const (
    _PRIMARY ifdId  = 0     // namespace for IFD0, first (TIFF) IFD
    _THUMBNAIL      = 1     // namespace for IFD1 (Thumbnail) pointed to by IFD0

    _EXIF           = 2     // EXIF namespace, embedded in IFD0
    _GPS            = 3     // GPS namespace, emvedded in IFD0 

    _IOP            = 4     // Interoperability namespace, embedded in EXIF IFD

    _MAKER          = 5     // one (non-standard) IFD for each maker note,
                            // embedded in EXIF IFD
)

type tTag    uint16             // TIFF tag
type tType   uint16             // TIIF type

const (                         // TIFF Types
    _UnsignedByte tType = 1 + iota
    _ASCIIString
    _UnsignedShort
    _UnsignedLong
    _UnsignedRational
    _SignedByte
    _Undefined
    _SignedShort
    _SignedLong
    _SignedRational
    _Float
    _Double

    // for special cases of _Undefined type actually being bytes or ASCII string
    _UndefinedByte
    _UndefinedString
)

const (                 // TIFF Type sizes (signed or unsigned)
    _ASCIIChar      = 1
    _ByteSize       = 1
    _ShortSize      = 2
    _LongSize       = 4
    _RationalSize   = 8
    _FloatSize      = 4
    _DoubleSize     = 8
)

func getTiffTString( t tType ) string {
    switch t {
        case _UnsignedByte: return "byte"
        case _ASCIIString: return "ASCII string"
        case _UnsignedShort: return "Unsigned short"
        case _UnsignedLong: return "Unsigned long"
        case _UnsignedRational: return "Unsigned rational"
        case _SignedByte: return "Signed byte"
        case _SignedShort: return "Signed short"
        case _SignedLong: return "Signed long"
        case _SignedRational: return "Signed rational"
        case _Float: return "Float"
        case _Double: return "Double"
        default: break
    }
    return "Undefined"
}

// common IFD entry structure (offset/value are specific to each value)
type tEntry struct {
    vTag    tTag
    vType   tType
    vCount  uint32
}

/*
    In order to allow mofdifying/removing TIFF metadata, each IFD entry is
    stored individually as a tiffvalue.

    TIFF types are converted in go types:
    _UnsignedByte       => []uint8
    _ASCIIString        => string
    _UnsignedShort      => []uint16
    _UnsignedLong       => []uint32
    _UnsignedRational   => []unsignedRational struct
    _SignedByte         => []int8
    _Undefined          => nil
    _SignedShort        => []int16
    _SignedLong         => []int32
    _SignedRational     => []signedRational struct
    _Float              => []float32
    _Double             => []float64

    Two other types are added:
    _embeddedIFD        => ifd structure
    _makerNote          => nkn structure
*/

type unsignedRational struct {
    Numerator, Denominator  uint32  // unexported type, but exported fields ;-)
}
type signedRational struct {
    Numerator, Denominator  int32
}

// A TIFF value is defined as its entry definition followed by one of the
// above go types and implementing the following interface:

type serializer interface {
// serialize the IFD entry of a value, return an error in case of failure
// By side effect, the parent ifd dOffset is updated for next calls with
// the size to be written later in the IFD data area or 0 if it fits in
// in the entry (less than or equal to _valOffSize)
    serializeEntry( w io.Writer) error

// serialize the IFD data of a value, return an error in case of failure
// By side effect the parent ifd dOffset is updated for next calls with the
// size written in the IFD data area or 0 if it fits in _valOffSize
    serializeData( w io.Writer ) error
}

// IFD generic support (conforming to TIFF, EXIF etc.)
type ifdd struct {
    id      ifdId           // namespace for each IFD
    desc    *Desc           // parent document descriptor
    values  []serializer    // stored IFD content

    dOffset uint32          // current offset in data-area during serializing
    dSize   uint32          // actual ifdd size or 0 if never serialized

    next    *ifdd           // next IFD in list

                            // current IFD field during parsing
    fTag    tTag            // field tag
    fType   tType           // field type
    fCount  uint32          // field count
    sOffset uint32          // field value/offset offset in desc.data
}

// ignore the actual entry tType and reads bytes instead
func (ifd *ifdd) getUnsignedBytes( ) []uint8 {
    if ifd.fCount <= 4 {
        return ifd.desc.getUnsignedBytes( ifd.sOffset, ifd.fCount )
    }
    rOffset := ifd.desc.getUnsignedLong( ifd.sOffset )
    return ifd.desc.getUnsignedBytes( rOffset, ifd.fCount )
}

func (ifd *ifdd) getAsciiString( ) string {
    var text string
    if ifd.fCount <= 4 {
        text = ifd.desc.getASCIIString( ifd.sOffset, ifd.fCount )
    } else {
        offset := ifd.desc.getUnsignedLong( ifd.sOffset )
        text = ifd.desc.getASCIIString( offset, ifd.fCount )
    }
    return text
}

// ignore the actual entry type and read shorts instead
func (ifd *ifdd) getUnsignedShorts( ) []uint16 {
    if ifd.fCount * _ShortSize <= 4 {
        return ifd.desc.getUnsignedShorts( ifd.sOffset, ifd.fCount )
    } else {
        rOffset := ifd.desc.getUnsignedLong( ifd.sOffset )
        return ifd.desc.getUnsignedShorts( rOffset, ifd.fCount )
    }
}


// storage does not presume any ifd data layout. This is done only at serializing
func (ifd *ifdd) storeValue( value serializer ) {
    i := len(ifd.values)
    if i >= cap(ifd.values) {
        panic( "storeValue called with no more current IFD entries\n" )
    }

    ifd.values = ifd.values[:i+1]         // extend slice within capacity
    fmt.Printf("storeValue: cap=%d len=%d i=%d\n", cap(ifd.values), len(ifd.values), i )
    ifd.values[i] = value
}

/*
    A complete IFD is made of:
         (_ShortSize + ( _IfdEntrySize * n ) + _LongSize) bytes
    plus the variable size data area

    Typically IFD0 is a TIFF IFD that embeds up to 4 other IFDs:

        GPS IFD     for specific Global Positioning System tags
        EXIF IFD    for specific Exif tags, which may include 2 other IFDs:

            IOP pointer         IOP IFD for specific Interoperability tags
            MN pointer          MakerNote (format similar to IFD but proprietary)

    Note that embedded IFDs are pointed to from a parent IFD, and do not use
    the next IFD pointer at the end. They are located in their parent IFD
    variable size data area. Their exact location in the data area does not
    matter.

    IFD0 is usually followed by IFD1, which is dedicated to embedded thumbnail
    images. In case of a JPEG thumbnails, the whole JPEG file without APPn
    marker is embedded in the IFD1 data area.

    Baseline TIFF requires only IFD0. EXIF requires that an EXIF IFD be embedded
    in IFD0, and if a thumnail image is included in addition to the main image,
    that IFD1 follows IFD0. IFD1 embeds the thumnail image.

    Metadata:

      IFD0 (Primary) ===================================
        n    (2 byte count)                            ^
        ...  (12-byte entries)                         |
        _ExifIFD ----------                            |
        ...                |         fixed size = (n * 12) + 2 + 4
        _GpsIFD ---------  |                           |
        ...              | |                           |
        ...              | |                           v
   -- next IFD (4 bytes) | | ===========================
   |  < IFD0 data        | |
   |     ...             | |
   |     GPS IFD <-------  | (optional)
   |       ...             |
   |       ...             |
   |     < GPS IFD data    |
   |       ...             |
   |     >                 |
   |     ...               |
   |     EXIF IFD <-------- 
   |       ...
   |       _MN ----------------- 
   |       ...                  |
   |       _IOP ----------      |
   |       ...            |     |
   |     < EXIF IFD data  |     |
   |       ...            |     |
   |       IOP IFD <------      |
   |         ...                |
   |       next IFD = 0         |
   |       < IOP IFD data >     |
   |       ...                  |
   |       MN IFD <------------- 
   |         ...
   |       next IFD = 0
   |       < MN IFD data >
   |       ...
   |     > (end EXIF IFD)
   |     ...
   |  > (end IFD0)
   -> IFD1 (THumbnail, optional)
        ...
      < IFD1 data
        Thumbnail JPEG image
      >
      next IFD = 0
*/

// Common value structure to embed in specific value definition
type tVal struct {
    ifd     *ifdd       // parent IFD
            tEntry      // common entry structure
}

// TIFF Value definitions

type ifdValue struct {
        tVal
    v   *ifdd           // embedded IFD
}
func (ifd *ifdd) newIfdValue( ifdVal *ifdd ) (iv *ifdValue) {
    iv = new( ifdValue )
    iv.ifd = ifd
    iv.vTag = ifd.fTag
    iv.vType = ifd.fType
    iv.vCount = 1
    iv.v = ifdVal
    return
}
func (iv *ifdValue) serializeEntry( w io.Writer ) (err error) {
    if err = binary.Write( w, iv.ifd.desc.endian, iv.tVal.tEntry ); err != nil {
        return
    }

    sz := iv.v.dSize
    if sz == 0 {
        fmt.Printf( "ifd %d Getting embedded IFD ID=%d size @offset %#08x\n",
                    iv.ifd.id, iv.v.id, iv.ifd.dOffset )
        _, err = iv.v.serializeEntries( io.Discard, 0 )
        if err != nil {
            fmt.Printf( "ifd %d Getting embedded IFD ID=%d size failed: %v\n",
                        iv.ifd.id, iv.v.id, err )
            return
        }
        sz = iv.v.dOffset   // since we serialzed from offset 0
        iv.v.dSize = sz     // save in case serializeEntry is called again
    }

    fmt.Printf( "ifd %d embedded IFD ID=%d size=%d\n", iv.ifd.id, iv.v.id, sz )
    err = binary.Write( w, iv.ifd.desc.endian, iv.ifd.dOffset )
    iv.ifd.dOffset += sz
    return
}
func (iv *ifdValue)serializeData( w io.Writer ) (err error) {
    fmt.Printf( "ifd %d Serialize embedded whole IFD %d @offset %#08x\n",
        iv.ifd.id, iv.v.id, iv.ifd.dOffset )
    var eSz, dSz uint32
    eSz, err = iv.v.serializeEntries( w, iv.ifd.dOffset )
    if err != nil {
        return
    }
    dSz, err = iv.v.serializeDataArea( w, iv.ifd.dOffset )
    if err == nil {
        iv.ifd.dOffset += eSz + dSz
    }
    return
}

type unsignedByteValue struct {
        tVal
    v   []uint8
}
func (ifd *ifdd) newUnsignedByteValue( ubVal []uint8 ) (ub *unsignedByteValue) {
    ub = new( unsignedByteValue )
    ub.ifd = ifd
    ub.vTag = ifd.fTag
    ub.vType = ifd.fType
    ub.vCount = uint32(len(ubVal))
    ub.v = ubVal
    return
}
func (ub *unsignedByteValue)serializeEntry( w io.Writer ) error {
    return ub.ifd.serializeSliceEntry( w, ub.tEntry, ub.v )
}
func (ub *unsignedByteValue)serializeData( w io.Writer ) error {
    return ub.ifd.serializeSliceData( w, ub.v )
}

// treat asciiStringgValue as unsignedByteValue 
func (ifd *ifdd) newAsciiStringValue( asVal string ) (as *unsignedByteValue) {
    as = new( unsignedByteValue )
    as.ifd = ifd
    as.vTag = ifd.fTag
    as.vType = ifd.fType
    as.vCount = uint32(len(asVal))
    as.v = []byte( asVal )
    return
}

type signedByteValue  struct {
        tVal
    v   []int8
}

func (ifd *ifdd) newSignedByteValue( sbVal []int8 ) (sb *signedByteValue) {
    sb = new( signedByteValue )
    sb.ifd = ifd
    sb.vTag = ifd.fTag
    sb.vType = ifd.fType
    sb.vCount = uint32(len(sbVal))
    sb.v = sbVal
    return
}
func (sb *signedByteValue)serializeEntry( w io.Writer ) error {
    return sb.ifd.serializeSliceEntry( w, sb.tEntry, sb.v )
}
func (sb *signedByteValue)serializeData( w io.Writer ) error {
    return sb.ifd.serializeSliceData( w, sb.v )
}

type unsignedShortValue struct {
        tVal
    v   []uint16
}
func (ifd *ifdd) newUnsignedShortValue( usVal []uint16 ) (us *unsignedShortValue) {
    us = new( unsignedShortValue )
    us.ifd = ifd
    us.vTag = ifd.fTag
    us.vType = ifd.fType
    us.vCount = uint32(len(usVal))
    us.v = usVal
    return
}
func (us *unsignedShortValue)serializeEntry( w io.Writer ) error {
    return us.ifd.serializeSliceEntry( w, us.tEntry, us.v)
}
func (us *unsignedShortValue)serializeData( w io.Writer ) error {
    return us.ifd.serializeSliceData( w, us.v )
}

type signedShortValue struct {
        tVal
    v   []int16
}
func (ifd *ifdd) newSignedShortValue( ssVal []int16 ) (ss *signedShortValue) {
    ss = new( signedShortValue )
    ss.ifd = ifd
    ss.vTag = ifd.fTag
    ss.vType = ifd.fType
    ss.vCount = uint32(len(ssVal))
    ss.v = ssVal
    return
}
func (ss *signedShortValue)serializeEntry( w io.Writer ) error {
    return ss.ifd.serializeSliceEntry( w, ss.tEntry, ss.v )
}
func (ss *signedShortValue)serializeData( w io.Writer ) error {
    return ss.ifd.serializeSliceData( w, ss.v )
}

type unsignedLongValue struct {
        tVal
    v   []uint32
}
func (ifd *ifdd) newUnsignedLongValue( ulVal []uint32 ) (ul *unsignedLongValue) {
    ul = new( unsignedLongValue )
    ul.ifd = ifd
    ul.vTag = ifd.fTag
    ul.vType = ifd.fType
    ul.vCount = uint32(len(ulVal))
    ul.v = ulVal
    return
}
func (ul *unsignedLongValue)serializeEntry( w io.Writer ) error {
    return ul.ifd.serializeSliceEntry( w, ul.tEntry, ul.v )
}
func (ul *unsignedLongValue)serializeData( w io.Writer ) error {
    return ul.ifd.serializeSliceData( w, ul.v )
}

type signedLongValue struct {
        tVal
    v   []int32
}
func (ifd *ifdd) newSignedLongValue( slVal []int32 ) (sl *signedLongValue) {
    sl = new( signedLongValue )
    sl.ifd = ifd
    sl.vTag = ifd.fTag
    sl.vType = ifd.fType
    sl.vCount = uint32(len(slVal))
    sl.v = slVal
    return
}
func (sl *signedLongValue)serializeEntry( w io.Writer ) error {
    return sl.ifd.serializeSliceEntry( w, sl.tEntry, sl.v )
}
func (sl *signedLongValue)serializeData( w io.Writer ) error {
    return sl.ifd.serializeSliceData( w, sl.v )
}

type unsignedRationalValue struct {
        tVal
    v  []unsignedRational
}
func (ifd *ifdd) newUnsignedRationalValue(
                    urVal []unsignedRational ) (ur *unsignedRationalValue) {
    ur = new( unsignedRationalValue )
    ur.ifd = ifd
    ur.vTag = ifd.fTag
    ur.vType = ifd.fType
    ur.vCount = uint32(len(urVal))
    ur.v = urVal
    return
}
func (ur *unsignedRationalValue)serializeEntry( w io.Writer ) error {
    return ur.ifd.serializeSliceEntry( w, ur.tEntry, ur.v )
}
func (ur *unsignedRationalValue)serializeData( w io.Writer ) error {
    return ur.ifd.serializeSliceData( w, ur.v )
}

type signedRationalValue struct {
        tVal
    v   []signedRational
}
func (ifd *ifdd) newSignedRationalValue(
                        srVal []signedRational ) (sr *signedRationalValue) {
    sr = new( signedRationalValue )
    sr.ifd = ifd
    sr.vTag = ifd.fTag
    sr.vType = ifd.fType
    sr.vCount = uint32(len(srVal))
    sr.v = srVal
    return
}
func (sr *signedRationalValue)serializeEntry( w io.Writer ) error {
    return sr.ifd.serializeSliceEntry( w, sr.tEntry, sr.v )
}
func (sr *signedRationalValue)serializeData( w io.Writer ) error {
    return sr.ifd.serializeSliceData( w, sr.v )
}

