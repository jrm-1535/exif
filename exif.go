// support for EXIF metadata parsing, removing and serializing
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

func GetCompressionString( c Compression ) string {
    switch c {
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

// Control Unknown tag BitMask:
// 0 => Keep unknown tag and metadata
// 1 => Remove tag and metadata
// 2 => stop in error at first unknown tag
const (
    Keep   = 0
    Remove = 1
    Stop   = 2
)

type Control struct {
    Unknown uint            // how to deal with unknown tags
    Warn    bool            // turn on warnings (unknown tags & non-fatal errors)
    ParsDbg bool            // turn on parse debug
    SrlzDbg bool            // turn on serialize debug
}

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

var ifdNames  = [...]string{ "Primary", "Thumbnail", "Exif",
                             "GPS", "Interoperability",
                             "Maker Note", "Maker Note Embedded" }

func (ifd *ifdd) getIfdName( ) string {
    id := ifd.id
    if id >= _IFD_N {
        panic("getIfdName: invalid Ifd Id\n")
    }
    return ifdNames[id]
}

type maker  struct {
    name    string
    try     func( *ifdd, uint32 ) (func( uint32 ) error)
}

var makerNotes = [...]maker{ { "Apple", tryAppleMakerNote },
                             { "Nikon", tryNikonMakerNote } }

type Desc struct {
    data    []byte          // starts at TIFF header (right after exif header)
    origin  uint32          // except for some maker notes
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
    if 0 == ifd.desc.Unknown & Remove {
        return ifd.storeAnyUnknownSilently( )
    }
    return nil
}

func (ifd *ifdd) processUnknownTag( ) error {
    if ifd.desc.Warn {
        fmt.Printf( "%s: unknown or unsupported tag (%#02x) @offset %#04x type %s count %d\n",
                    ifd.getIfdName(), ifd.fTag, ifd.sOffset-8,
                    getTiffTString( ifd.fType ), ifd.fCount )
    }
    if 0 != ifd.desc.Unknown & Stop {
        return fmt.Errorf( "%s: storeExifTags: stop at unknown tag %#02x\n",
                           ifd.getIfdName(), ifd.fTag )
    }
    if 0 == ifd.desc.Unknown & Remove {
        return ifd.storeAnyUnknownSilently( )
    }
    return nil
}

func dumpData( header string, indent string, data []byte ) {
    fmt.Printf( "%s:\n", header )
    for i := 0; i < len(data); i += 16 {
        fmt.Printf("%s%#04x: ", indent, i );
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
            fmt.Printf( "%02x ", data[i+j] )
        }
        for ; j < 16; j++ {
            fmt.Printf( "   " )
        }
        fmt.Printf( "%s\n", b.String() )
    }
}

func getEndianess( data []byte ) ( endian binary.ByteOrder, err error) {
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
// metadata: an EXIF header is expected at the starting offset and the whole
// metadata must fit in the following number of bytes. If the metadata size
// is unknown, dLen can be given as 0, in which case parsing will use the rest
// of the input slice.
//
// It returns the descriptor in case of success or an error in case of failure.
func Parse( data []byte, start, dLen int, ec *Control ) (*Desc, error) {
    if ! bytes.Equal( data[start:start+6], []byte( "Exif\x00\x00" ) ) {
        return nil, fmt.Errorf( "exif: invalid signature (%s)\n",
                                string(data[0:6]) )
    }
    // Exif\0\0 is followed immediately by TIFF header
    d := newDesc( data[start+_originOffset:start+dLen-_originOffset], ec )
    var err error
    d.endian, err = getEndianess( d.data )
    if err != nil {
        return nil, err
    }

    offset, err := d.checkValidTiff( )
    if err != nil {
        return nil, err
    }
//    fmt.Printf( "  Primary Image metadata @%#04x\n", offset )
    offset, d.root, err = d.storeIFD( PRIMARY, offset, storeTiffTags )
    if err != nil {
        return nil, err
    }

    if offset != 0 {
//        fmt.Printf( "  Thumbnail Image metadata @%#04x\n", offset )
        _, d.root.next, err = d.storeIFD( THUMBNAIL, offset, storeTiffTags )
        if err != nil {
            return nil, err
        }
    }
    return d, nil
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
// Otherwise it returns an error. 
//
// It implements the bitap (or shift-Or) algorithm to quickly find the exif
// header. Exif header is 6-byte long ("Exif\x0\x0") and requires only a 6-bit
// position mask. It uses a 256-byte mask array, which is is likely to stay in
// cache, and a bitmask that fits in a register. The time complexity is O(n).
func Search( data []byte, start int ) ([]byte, error) {

    bitMask := byte(0xfe)
    for i:= start; i < len(data); i++ {
        bitMask |= masks[data[i]]
        bitMask <<= 1
        if 0 == bitMask & 64 {
//            fmt.Printf("Found Exif header @%#08x (%v)\n", i-5, string(data[i-5:i-1]) )
            return data[i-5:], nil
        }
    }
    return data[0:0], fmt.Errorf("search: did not find Exif header in data\n")
}

// Read the file whose path name is given and parse the data.
// It takes the path name (path), a starting offset in that file.
// It searches for the EXIF header from that starting offset, which
// can therefore be given as 0 if it is unknown.
//
// It returns an exif descriptor in case of success or an error in
// case of failure.
func Read( path string, start int, ec *Control ) (*Desc, error) {
    data, err := ioutil.ReadFile( path )
    if err != nil {
        return nil, err
    }
    data, err = Search( data, start )
    if err != nil {
        return nil, err
    }
    return Parse( data, 0, len(data), ec )
}

// Write the parsed EXIF metadata into a file. It returns the number of bytes
// written in the file in case of success or an error in case of failure.
func (d *Desc)Write( path string ) (n int, err error) {

    var f *os.File
    f, err = os.OpenFile( path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
    if err != nil {
        return 0, err
    }
    n, err = d.serialize( f )
    if err != nil {
        return
    }
    fmt.Printf( "wrote %d bytes to %s\n", n, path )
    f.Close( )

    // temporary, save exif data into file - exif-src.txt
    {
        ifd := d.root
        if ifd.next != nil {
            ifd = ifd.next
        }
        dLen := int( ifd.dOffset )
	    f, err = os.OpenFile( "exif-src.bin", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
        _, err = f.Write( []byte( "Exif\x00\x00" ) )
        _, err = f.Write( d.data[0:dLen] )
        if err != nil { return 0, err }
        err = f.Close( )
        if err != nil { return 0, err }
    }
    // end of temporary stuff

    return
}

// GetThumnail returns information about a possible thumbnail.
// It returns the thumbnail starting offset in the original data slice,
// the size of the thumbnail data and the thumbnail compression type.
// If no thumbnail was found in the metadata, the starting offset and
// size are 0 and the comprssion type is exif.Undefined Compression.
func (d *Desc)GetThumbnail() (uint32, uint32, Compression) {
    tOffset, _ := d.global["thumbOffset"].(uint32)
    tLen, _ := d.global["thumbLen"].(uint32)
    tType, _ := d.global["thumbType"].(Compression)

    if tOffset != 0 {
        tOffset += _originOffset
    }
    return tOffset, tLen, tType
}

// Write formatted IFDs on the passed io.Writer argument w
// if w is nil, os.Stdout is used
// The IFDs to format are given by their IDs in the slice argument ifdIds
// Possible ID values are: PRIMARY, THUMBNAIL, EXIF, GPS, IOP, MAKER & EMBEDDED
func (d *Desc)Format( w io.Writer, ifdIds []IfdId ) error {
    if w == nil {
        w = os.Stdout
    }
    fmt.Fprintf( w, "Picture Metadata:\n" )
    for i := 0; i < len(ifdIds); i++ {
        id := ifdIds[i]
        if /*id >= PRIMARY &&(*/ id < _IFD_N {
            ifd := d.ifds[id]
            if ifd != nil {
                fmt.Fprintf( w, "--- %s IFD (id %d)\n", ifdNames[id], id )
                ifd.format( w )
            } else {
                fmt.Fprintf( w, "--- %s IFD (id %d) is absent\n", ifdNames[id], id )
            }
        }
    }
    return nil
}

