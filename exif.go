// support for EXIF metadata parsing, removing and serializing
//
// EXIFF metadata are usually embedded in pictures, such as JPEG files and
// provide information such as date, location and what camera was used.
//
// Before sharing a picture it might be necessary to remove some metadata
// that could reveal personal information. This library allows checking what
// metadata is available in a picture, removing some of those metadata and
// re-writing the resulting set of metadata.
//
// Part of the metadata are from the camera manufacturer, in the form of Maker
// notes. In this version, only a subset of Apple and Nikon maker notes are
// supported
package exif

import (
    "fmt"
    "bytes"
    "strings"
    "encoding/binary"
    "io/ioutil"
    "io"
    "os"
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
    _valOffSize   = 4           // value fits in if <= 4 bytes, otherwise offset
    _IfdEntrySize = ( _ShortSize + _LongSize) * 2
)

// Image Compression enum
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

// return the name of a compression code
func GetCompressionName( c Compression ) string {
    switch c {
    case Undefined:             return "Undefined"
    case NotCompressed:         return "Not compressed"
    case CCITT_1D:              return "CCITT 1D"
    case CCITT_Group3:          return "CCITT Group 3"
    case CCITT_Group4:          return "CCITT Group 4"
    case LZW:                   return "LZW"
    case JPEG:                  return "JPEG"
    case JPEG_Technote2:        return "JPEG (Technote 2)"
    case Deflate:               return "DEFLATE"
    case RFC_2301_BW_JBIG:      return "RFC_2301_BW_JBIG"
    case RFC_2301_Color_JBIG:   return "RFC_2301_Color_JBIG"
    case PackBits:              return "PACKBITS"
    default: break
    }
    return "Unknown"
}

// Control Unknown Tag bitMask
type ConUnTag uint
const (
    KeepTag ConUnTag = iota // Keep unknown tag and metadata
    RemoveTag               // Remove tag and metadata
    Stop                    // Stop in error at first unknown tag
)

type Control struct {
    Unknown ConUnTag        // how to deal with unknown tags
    Warn    bool            // turn on warnings (unknown tags & non-fatal errors)
    ParsDbg bool            // turn on parse debug
    SrlzDbg bool            // turn on serialize debug
}

// IFD ID, used as a namespace for IFD tags
type IfdId  uint
const (
    PRIMARY IfdId = iota    // namespace for IFD0, first (TIFF) IFD
    THUMBNAIL               // namespace for IFD1 (Thumbnail) pointed to by IFD0
    EXIF                    // EXIF namespace, embedded in IFD0
    GPS                     // GPS namespace, embedded in IFD0 
    IOP                     // Interoperability namespace, embedded in EXIF IFD
    MAKER                   // non-standard IFD for each maker, embedded in EXIF IFD
    EMBEDDED                // possible non-standard IFD embedded in MAKER
    _IFD_N                  // last entry + 1 to size arrays
)

type ThumbnailInfo struct {
    Origin  IfdId           // either THUMBNAIL or EMBEDDED
    Comp    Compression     // type of image compression
    Size    uint32          // image size
}

var ifdNames  = [...]string{ "Primary", "Thumbnail", "Exif",
                             "GPS", "Interoperability",
                             "Maker Note", "Maker Note Embedded" }

// Given an IfdId, return the corresponding Ifd nickname
func GetIfdName( id IfdId ) string {
    if id < _IFD_N {
        return ifdNames[id]
    }
    return "Unknown Ifd"
}

type maker  struct {
    name    string
    try     func( *ifdd, uint32 ) (func( uint32 ) error)
}

var makerNotes = [...]maker{ { "Apple", tryAppleMakerNote },
                             { "Nikon", tryNikonMakerNote } }

type Desc struct {
    data    []byte          // starts at TIFF header (right after exif header)
    origin  uint32          // except for some maker notes (e.g. apple)
    dataEnd uint32          // data area end, updated during parsing

    endian  binary.ByteOrder // endianess as defined in binary

    global  map[string]interface{}  // storage for global information

            control         // what to do when parsing

    root    *ifdd           // tree of ifd for rewriting exif metadata
    ifds    [_IFD_N]*ifdd   // flat access to ifd by id
}

type control struct {
            Control         // to keep Desc fully opaque
}

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

func getTiffTypeSize( t tType ) uint32 {
    switch t {
        case _UnsignedByte: return _ByteSize
        case _ASCIIString: return _ByteSize
        case _UnsignedShort: return _ShortSize
        case _UnsignedLong: return _LongSize
        case _UnsignedRational: return _RationalSize
        case _SignedByte: return _ByteSize
        case _Undefined: return _ByteSize   // count in bytes
        case _SignedShort: return _ShortSize
        case _SignedLong: return _LongSize
        case _SignedRational: return _RationalSize
        case _Float: return _FloatSize
        case _Double: return _DoubleSize
        default:
            break
    }
     panic(fmt.Sprintf("TIFF type %s does not have a size", getTiffTString(t)))
}

func getTiffTString( t tType ) string {
    switch t {
        case _UnsignedByte: return "Unsigned byte"
        case _ASCIIString: return "ASCII string"
        case _UnsignedShort: return "Unsigned short"
        case _UnsignedLong: return "Unsigned long"
        case _UnsignedRational: return "Unsigned rational"
        case _SignedByte: return "Signed byte"
        case _Undefined: return "Undefined"
        case _SignedShort: return "Signed short"
        case _SignedLong: return "Signed long"
        case _SignedRational: return "Signed rational"
        case _Float: return "Float"
        case _Double: return "Double"
        default: break
    }
    return fmt.Sprintf("Unknown (%d)", t )
}

func (d *Desc) readTIFFData( offset uint32, dest interface{} ) {
    b := bytes.NewBuffer( d.data[offset:] )
    binary.Read( b, d.endian, dest )
    return
}

// (d *Desc)get<tType>s(offset, count) functions read the requested count of
// typed data from an offset anywhere in the data slice, using the endianess
// and data slice from d. The result is a slice of the corresponding go type.

func (d *Desc) getByte( offset uint32 ) uint8 {
    return d.data[offset]
}

func (d *Desc) getUnsignedBytes( offset, count uint32 ) []uint8 {
    return d.data[offset:offset+count]
}

func (d *Desc) getSignedBytes( offset uint32, count uint32 ) []int8 {
    r := make( []int8, count )
    d.readTIFFData( offset, &r )
    return r
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

func (d *Desc) getSignedShorts( offset uint32, count uint32 ) []int16 {
    r := make( []int16, count )
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

func (d *Desc) getUnsignedRationals( offset, count uint32 ) []unsignedRational {
    r := make( []unsignedRational, count )
    d.readTIFFData( offset, &r )
    return r
}

func (d *Desc) getSignedRationals( offset, count uint32 ) []signedRational {
    r := make( []signedRational, count )
    d.readTIFFData( offset, &r )
    return r
}

func (d *Desc) checkValidTiff( ) (uint32, error) {
    validTiff := d.getUnsignedShort( 2 )
    if validTiff != 0x2a {
        return 0, fmt.Errorf(
            "checkValidTiff: invalid TIFF header (invalid identifier: %#02x)\n",
             validTiff )
    }
    // followed by Primary Image File directory (IFD) offset
    return d.getUnsignedLong( 4 ), nil
}

// IFD generic support (conforming to TIFF, EXIF etc.)
type ifdd struct {
    id      IfdId           // namespace for each IFD
    desc    *Desc           // parent document descriptor
    pValue  serializer      // parent value in case of embedded ifd or desc


    values  []serializer    // stored IFD content

    dOffset uint32          // current offset in data-area during serializing
    dSize   uint32          // actual ifdd size or 0 if never serialized

    next    *ifdd           // next IFD in list

                            // current IFD field during parsing
    fTag    tTag            // field tag
    fType   tType           // field type
    fCount  uint32          // field count
    sOffset uint32          // field value or offset in desc.data
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

func (ifd *ifdd) processPadding( ) error {
    if 0 == ifd.desc.Unknown & RemoveTag {
        return ifd.storeAnyUnknownSilently( )
    }
    return nil
}

func (ifd *ifdd) processUnknownTag( ) error {
    if ifd.desc.Warn {
        fmt.Printf( "%s: unknown or unsupported tag (%#02x) @offset %#04x type %s count %d\n",
                    GetIfdName(ifd.id), ifd.fTag, ifd.sOffset-8,
                    getTiffTString( ifd.fType ), ifd.fCount )
    }
    if 0 != ifd.desc.Unknown & Stop {
        return fmt.Errorf( "%s: storeExifTags: stop at unknown tag %#02x\n",
                           GetIfdName(ifd.id), ifd.fTag )
    }
    if 0 == ifd.desc.Unknown & RemoveTag {
        return ifd.storeAnyUnknownSilently( )
    }
    return nil
}

func dumpData(w io.Writer,  header, indent string, noLf bool, data []byte ) {
    fmt.Fprintf( w, "%s:\n", header )
    for i := 0; i < len(data); i += 16 {
        fmt.Fprintf( w, "%s%#04x: ", indent, i );
        l := 16
        if len(data)-i < 16 {
            l = len(data)-i
        }
        var b strings.Builder
        j := 0
        for ; j < l; j++ {
            if data[i+j] < 0x20 || data[i+j] > 0x7f {
                b.WriteByte( '.' )
            } else {
                b.WriteByte( data[i+j] )
            }
            fmt.Fprintf( w, "%02x ", data[i+j] )
        }
        for ; j < 16; j++ {
            io.WriteString( w, "   " )
        }
        if noLf && i + 16 >= len(data) {
            io.WriteString( w, b.String() )
        } else {
            fmt.Fprintf( w, "%s\n", b.String() )
        }
    }
}

func (ifd *ifdd)removeIfdTag( tag tTag ) {
    for i, v := range( ifd.values ) {
        if v != nil {
            t := v.getTag()
            if t == tag {
//                fmt.Printf( "removeTag: found tag %d @ entry %d in ifd %s (%d)\n",
//                            tag, i, GetIfdName(ifd.id), ifd.id )
                ifd.values[i] = nil
                return
            }
        }
    }
    if ifd.desc.Warn {
        fmt.Printf( "removeTag: missing tag %d in ifd %s (%d)\n",
                    tag, GetIfdName(ifd.id), ifd.id )
    }
}

func (d *Desc)removeIfdTag( id IfdId, tag uint ) error {
    if id >= _IFD_N {
        return fmt.Errorf( "RemoveIfdTag: id %d is not valid for an ifd\n", id )
    }
    ifd := d.ifds[id]
    if ifd == nil {
        return fmt.Errorf( "RemoveIfdTag: ifd %d is not present\n", id )
    }
    if tag >0xffff {
        return fmt.Errorf( "RemoveIfdTag: tag %d is out of range\n", tag )
    }
    eTag := tTag(tag)
    ifd.removeIfdTag( eTag )

    // special cases for JPEGInterchangeFormat/Length
    if id == PRIMARY || id == THUMBNAIL || id == EMBEDDED {
        if eTag == _JPEGInterchangeFormat {
            eTag = _JPEGInterchangeFormatLength
        } else if eTag == _JPEGInterchangeFormatLength {
            eTag = _JPEGInterchangeFormat
        } else {
            eTag = 0
        }
        if eTag != 0 {
            ifd.removeIfdTag( eTag )
         }
    }
    return nil
}

func removeVal( val serializer) {
    if ifdVal, ok := val.(*ifdValue); ok == true {
        ifd := ifdVal.ifd
        for i, v := range( ifd.values ) {
            if iv, ok := v.(*ifdValue); ok == true {
                if iv == ifdVal {
//                    fmt.Printf( "Found Ifd value at index %d in parent ifd id %d\n",
//                                i, ifd.id )
                    ifd.values[i] = nil
                    return
                }
            }
        }
    } else if descVal, ok := val.(*descValue); ok == true {
        ifd := descVal.ifd
        for i, v := range( ifd.values ) {
            if dv, ok := v.(*descValue); ok == true {
                if dv == descVal {
//                    fmt.Printf( "Found Desc value at index %d in parent ifd id %d\n",
//                                i, ifd.id )
                    ifd.values[i] = nil
                    return
                }
            }
        }
    }
    panic( "removeVal: value not found\n")
}

func (d *Desc)removeIfd( id IfdId ) error {
    if id >= _IFD_N {
        return fmt.Errorf( "RemoveIfd: id %d is not valid for an ifd\n", id )
    }
    if id == PRIMARY {
        return fmt.Errorf( "RemoveIfd: removing ifd PRIMARY is not possible\n")
    }

    ifd := d.ifds[id]
    if ifd == nil {
        return fmt.Errorf( "RemoveIfd: ifd %d is not present\n", id )
    }

    // 1. remove entry in parent ifd, if any
    if pVal := ifd.pValue; pVal != nil {
        removeVal( pVal )
    }

    // 2. remove ifd from the desc main Desc chain, if any
    chain := d.ifds[PRIMARY]
    if chain.next != nil && chain.next == ifd {
        chain.next = nil
    }

    // 3. remove ifd in ifid ids in main Desc
    d.ifds[id] = nil

    return nil
}

// Remove tag from the list of entries in the specified ifd. Call to Write
// afterwards will not include this tag in the metadata. Since ifds act as
// namespace, the same tag value can appear in multiple ifds, and the ifd id is
// necessary to uniquely identify a tag.
//
// The argument id indicates the enclosing ifd and the argument tag specifies
// the tag to remove. Errors are returned in case of invalid ifds ids and out-
// of-range tags (>0x0ffff). If the ifd id is not present an error is returned,
// but if the tag is not found it is just ignored.
//
// If the tag to remove is given as -1, the whole ifd is removed (-1 stands for
// all tags in the ifd). Beware that removing a whole IFD removes all embedded
// IFDs and any embedded thumbnail as well, and removing the whole PRIMARY ifd
// will remove all existing ifds, resulting in an empty metadata descriptor.
//
// Removing a tag can make the enclosing ifd meaningless. Some tags come in
// couples, like _JPEGInterchangeFormat and _JPEGInterchangeFormatLength and
// must always be both removed even if only one is specified. This case is
// handled here, but other possible similar cases, in maker notes for example,
// are not.
func (d *Desc)Remove( id IfdId, tag int ) (err error) {
    if id == 0 {        // remove all exif metadata
        d.root = nil
        for id := PRIMARY; id < _IFD_N; id ++ {
            d.ifds[id] = nil
        }
        return
    }
    if tag < -1 {
        return fmt.Errorf( "Remove: invalid tag %d\n", tag )
    }
    defer func ( ) { if err != nil { err = fmt.Errorf( "Remove: %v", err ) } }()

    if tag == -1 {      // remove the whole ifd
        err = d.removeIfd( IfdId(id) )
    } else {
        err = d.removeIfdTag( IfdId(id), uint(tag) )
    }
    return
}

func getEndianess( data []byte ) ( endian binary.ByteOrder, err error ) {
    endian = binary.BigEndian
    err = nil
    // TIFF header starts with 2 bytes indicating the byte ordering ("II" short
    // for Intel or "MM" short for Motorola, indicating little or big endian
    // respectively)
    if bytes.Equal( data[:2], []byte( "II" ) ) {
        endian = binary.LittleEndian
    } else if ! bytes.Equal( data[:2], []byte( "MM" ) ) {
        err = fmt.Errorf(
                "getEndianess: invalid TIFF header (unknown byte ordering: %v)\n",
                data[:2] )
    }
    return
}

func newDesc( data []byte, c *Control ) *Desc {
    d := new( Desc )
    d.data = data
    d.Control = *c
    d.global = make(map[string]interface{})
    return d
}

// Parse data for exif metadata and build up an exif descriptor.
//
// It takes a byte slice as input (data), a starting offset in that slice
// (start) and the following number of bytes (dLen) that contains the exif
// metadata.
//
// An EXIF header is expected at the starting offset and the whole metadata
// must fit in the following number of bytes. If the metadata size is unknown,
// dLen can be given as 0, in which case parsing will use the rest of the input
// slice.
//
// It returns the descriptor in case of success or a non-nil error in case of
// failure.
func Parse( data []byte, start, dLen uint, ec *Control ) (desc *Desc, err error) {
    if ! bytes.Equal( data[start:start+6], []byte( "Exif\x00\x00" ) ) {
        return nil, fmt.Errorf( "Parse: invalid signature (%s)\n",
                                string(data[0:6]) )
    }

    // Exif\0\0 is followed immediately by TIFF header
    d := newDesc( data[start+_originOffset:start+dLen-_originOffset], ec )
    defer func ( ) {
        if err != nil {
            err = fmt.Errorf( "Parse: %v", err )
        } else {
            desc = d
        }
    }()

    d.endian, err = getEndianess( d.data )
    if err != nil {
        return
    }

    var offset uint32
    offset, err = d.checkValidTiff( )
    if err != nil {
        return
    }
    offset, d.root, err = d.storeIFD( PRIMARY, offset, storeTiffTags )
    if err != nil {
        return
    }
    if offset != 0 {
        _, d.root.next, err = d.storeIFD( THUMBNAIL, offset, storeTiffTags )
    }
    return
}

var masks [256]byte

func init() {
    for i:= 0; i < 256; i++ { masks[i] = 0xff }
    masks['E'] = 0xfe   // position at bit 0 in pattern
    masks['x'] = 0xfd   // position at bit 1 ...
    masks['i'] = 0xfb
    masks['f'] = 0xf7
    masks[ 0 ] = 0xcf   // 2 positions for \x0 (bits 4 and 5)
}

// Search looks up for the EXIF header in memory. It takes a source data slice
// and a start offset in that slice. If the EXIF header is found it returns the
// slice starting at the EXIF header, till the end of the original data slice.
// Otherwise it returns a non-nil error.
//
// It implements the bitap (or shift-Or) algorithm to quickly find the exif
// header. Exif header is 6-byte long ("Exif\x0\x0") and requires only a 6-bit
// position mask. It uses a 256-byte mask array, which is is likely to stay in
// cache, and a bitmask that fits in a register. The time complexity is O(n).
func Search( data []byte, start uint ) ([]byte, error) {

    bitMask := byte(0xfe)
    for i:= int(start); i < len(data); i++ {
        bitMask |= masks[data[i]]
        bitMask <<= 1
        if 0 == bitMask & 64 {
//            fmt.Printf("Found Exif header @%#08x (%v)\n", i-5, string(data[i-5:i-1]) )
            return data[i-5:], nil
        }
    }
    return nil, fmt.Errorf("search: did not find Exif header in data\n")
}

// Read the file whose path name is given and parse the data.
//
// It takes the path name (path) and a starting offset in that file.
// It searches for the EXIF header from that starting offset, which
// should therefore be given as 0 if it is unknown.
//
// It returns an exif descriptor in case of success or an error in
// case of failure.
func Read( path string, start uint, ec *Control ) (d *Desc, err error) {
    defer func ( ) {
        if err != nil { err = fmt.Errorf( "Read: %v", err ) }
    }()

    var data []byte
    data, err = ioutil.ReadFile( path )
    if err != nil {
        return
    }
    data, err = Search( data, start )
    if err != nil {
        return
    }
    d, err = Parse( data, 0, uint(len(data)), ec )
    return
}

// Write the parsed EXIF metadata into a file.
// The argument path gives the path of the new file to write.
//
// It returns the number of bytes written in the file in case of success
// or a non-nil error in case of failure.
func (d *Desc)Write( path string ) (n int, err error) {

    defer func ( ) {
        if err != nil { err = fmt.Errorf( "Write: %v", err ) }
    }()

    var f *os.File
    f, err = os.OpenFile( path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
    if err != nil {
        return
    }
    defer func ( ) { if e := f.Close(); err == nil { err = e } }()
    n, err = d.Serialize( f )
    return
}

// WriteOriginal writes the original EXIF metadata into a new seperate file.
// The argument path gives the path of the new file to write.
//
// This useful if the file that was parsed included the EXIF metadata along
// with other data, such as in a JPEG file.
//
// If succesful, it returns the number of bytes written, otherwise it returns
// a non-nil error.
func (d *Desc)WriteOriginal( path string ) (n int, err error) {

    defer func ( ) {
        if err != nil { err = fmt.Errorf( "WriteOriginal: %v", err ) }
    }()
    var f *os.File
    f, err = os.OpenFile( path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
    if err != nil {
        return
    }

    defer func ( ) { if e := f.Close(); err == nil { err =e  } }()
    n, err = f.Write( []byte( "Exif\x00\x00" ) )
    if err != nil {
        return
    }
    var written int
    written, err = f.Write( d.data[0:d.dataEnd] )
    n += written
    return
}

// GetThumnailData
// The argument id gives the id of the ifd that provides the thumbnail.
//
// As long as an ifd referred by the id exists, associated thumbnail
// information is retrieved and if the thumbnail size is not 0, it is
// returned as a byte slice.
//
// Actually any existing ifd in [ exif.PRIMARY, exif.THUMBNAIL, exif.EXIF,
// exif.GPS, exif.IOP ] will return the exif thumbnail (if it exists),
// while any existing ifd in [ exif.MAKER, exif.EMBEDDED] will return the
// maker thumbnail (or preview image) if it exists.
func (d *Desc)GetThumbnailData( id IfdId ) ([]byte, error) {
    var ifd *ifdd
// First locate the ifd in the main descriptor ifd list, then use the ifd 
// parent desc as the source of thumbnail data (EMBEDDED IFD has a different
// desc and different data origin).
    if id < _IFD_N {
        ifd = d.ifds[id]
    }
    if ifd == nil {
        return nil, fmt.Errorf( "ifd %d not found\n", id )
    }
    tOffset, _ := ifd.desc.global["thumbOffset"].(uint32)
    if tOffset == 0 {
        return nil, fmt.Errorf( "thumbnail not found in ifd %d\n", id )
    }
    tLen, _ := ifd.desc.global["thumbLen"].(uint32)
    if tLen == 0 {
        return nil, fmt.Errorf( "empty thumbnail found in ifd %d\n", id )
    }
    return ifd.desc.data[tOffset:tOffset+tLen], nil
}

// WriteThumbnail writes the thumbnail data into a new seperate file.
//
// The argument path gives the path of the new file to write.
// The argument from gives the id of the ifd that provides the thumbnail.
//
// As long as an ifd referred by the id exists, thumbnail information is
// retrieved and if the thumbnail is not empty it is written to the file.
//
// Actually any existing ifd in [ exif.PRIMARY, exif.THUMBNAIL, exif.EXIF,
// exif.GPS, exif.IOP ] will write the exif thumbnail (if it exists),
// while any existing ifd in [ exif.MAKER, exif.EMBEDDED] will write the
// maker thumbnail (or preview image) if it exists.
//
// If succesful, it returns the number of bytes written, otherwise it returns
// a non-nil error.
func (d *Desc)WriteThumbnail( path string, from IfdId ) (n int, err error) {

    defer func ( ) {
        if err != nil { err = fmt.Errorf(  "WriteThumbail: %v", err ) }
    }()

    var data []byte
    data, err = d.GetThumbnailData( from ); if err != nil {
        return
    }

    var f *os.File
    f, err = os.OpenFile( path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
    if err != nil {
        return
    }
    defer func ( ) { if e := f.Close(); err == nil { err = e } }()
    return f.Write( data )
}

// GetThumbnailInfo returns information about all possible thumbnails.
// It returns a slice of ThumnailInfo structures. In each ThumbailInfo, it
// gives the thumbnail origin (either "Thumbnail" or "Maker Note Embedded"),
// the size of the thumbnail data and the thumbnail compression type.
func (d *Desc)GetThumbnailInfo() (ti []ThumbnailInfo) {
    ti = make( []ThumbnailInfo, 0, 2 )
    tOffset, _ := d.global["thumbOffset"].(uint32)
    if tOffset != 0 {
        tLen, _ := d.global["thumbLen"].(uint32)
        tType, _ := d.global["thumbType"].(Compression)
        ti = append( ti, ThumbnailInfo{ THUMBNAIL, tType, tLen } )
    }

    // lookup for EMBEDDED IfID:
    for id := IfdId(0); id < _IFD_N; id++ {
        ifd := d.ifds[id]
        if ifd != nil && ifd.id == EMBEDDED {
            tOffset, _ = ifd.desc.global["thumbOffset"].(uint32)
            if tOffset != 0 {
                tLen, _ := ifd.desc.global["thumbLen"].(uint32)
                tType, _ := ifd.desc.global["thumbType"].(Compression)
                ti = append( ti, ThumbnailInfo{ EMBEDDED, tType, tLen } )
            }
            break
        }
    }
    return
}

// Format all existing IDs
// The argument w is the io.Writer to use (e.g. os.File or). If w is nil,
// os.Stdout is used instead.
//
// It returns always a nil error.
func (d *Desc)Format( w io.Writer) error {
    if w == nil {
        w = os.Stdout
    }
    fmt.Fprintf( w, "------ Picture Metadata:\n\n" )
    for id:= PRIMARY; id < _IFD_N; id++ {
        ifd := d.ifds[id]
        if ifd != nil {
            fmt.Fprintf( w, "--- %s IFD (id %d)\n", ifdNames[id], id )
            ifd.format( w )
        }
    }
    fmt.Fprintf( w, "------\n" )
    return nil
}

// Format IFDs.
// The argument w is the io.Writer to use (e.g. os.File). If w is nil, os.Stdout
// is used instead. The IFDs to format are given by their IDs in the slice argument
// ifdIds. Possible ID values are: PRIMARY, THUMBNAIL, EXIF, GPS, IOP, MAKER & EMBEDDED
//
// it returns a non-nil error if one ifd in the slice is not in the range of valid
// IFD IDs. In an IFD is not present in the metadata it silently skips it, unless
// the flasg Warn is set, in which case it prints the name of the missing IFD.
func (d *Desc)FormatIfds( w io.Writer, ifdIds []IfdId ) error {
    if w == nil {
        w = os.Stdout
    }
    fmt.Fprintf( w, "Picture Metadata:\n\n" )
    for _, id := range ifdIds {
        if id < _IFD_N {
            ifd := d.ifds[id]
            if ifd != nil {
                fmt.Fprintf( w, "--- %s IFD (id %d)\n", ifdNames[id], id )
                ifd.format( w )
            } else {
                if d.Warn {
                    fmt.Printf( "--- %s IFD (id %d) is absent\n", ifdNames[id], id )
                }
            }
        } else {
            return fmt.Errorf( "FormatIfds: id %d is not valid for an ifd\n", id )
        }
    }
    return nil
}
