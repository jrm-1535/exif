package exif

// support for EXIF metadata parsing and removing

import (
    "fmt"
    "bytes"
    "strings"
//    "os"
)

// exported types and constants
type Control struct {
    print bool
}

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

// exif data structure
type entry struct {
    size    uint
    tType   byte
    tCount  uint
    tData   interface{}
}

type ifd struct {
    size    uint
    entries []entry
    next    *ifd
}

type control struct {
        Control         // to keep Desc opaque
}

type Desc struct {
    data    []byte      // data starts at TIFF header (origin after exif header)
    offset  uint        // IFD entry offset inside data
    lEndian bool        // little endian (true) or big endian (false)

    tType   Compression // Thumbnail information if any
    tOffset uint
    tLen    uint

            control // what to do

    ifds    []ifd       // storage for rewriting the exif metadata
}

// TIFF specific support
const (                 // TIFF Types
    _UnsignedByte = 1
    _ASCIIString = 2
    _UnsignedShort = 3
    _UnsignedLong = 4
    _UnsignedRational = 5
    _SignedByte = 6
    _Undefined = 7
    _SignedShort = 8
    _SignedLong = 9
    _SignedRational = 10
    _Float = 11
    _Double = 12
)

const (                 // TIFF Type sizes
    _ByteSize       = 1
    _ShortSize      = 2
    _LongSize       = 4
    _RationalSize   = 8
    _FloatSize      = 4
    _DoubleSize     = 8
)

func (ed *Desc) getByte( offset uint ) byte {
    return ed.data[offset]
}

func (ed *Desc) getBytes( offset, count uint ) []byte {
    vSlice := make( []byte, count )
    for i := uint(0); i < count; i++ {
        vSlice[i] = ed.data[ offset ]
        offset += _ByteSize
    }
    return vSlice
}

func (ed *Desc) getASCIIString( offset, count uint ) string {
    var b strings.Builder
    b.Write( ed.data[offset:offset+count] )
    return b.String()
}

func (ed *Desc) getUnsignedShort( offset uint ) uint {
    if ed.lEndian {
        return (uint(ed.data[offset+1]) << 8) + uint(ed.data[offset])
    }
    return (uint(ed.data[offset]) << 8) + uint(ed.data[offset+1])
}

func (ed *Desc) getUnsignedShorts( offset, count uint ) []uint {
    vSlice := make( []uint, count )
    for i := uint(0); i < count; i++ {
        vSlice[i] = ed.getUnsignedShort( offset )
        offset += _ShortSize
    }
    return vSlice
}

func (ed *Desc) getBytesFromIFD( count uint ) []byte {
    if count <= 4 {
        return ed.getBytes( ed.offset, count )
    }
    rOffset := ed.getUnsignedLong( ed.offset )
    return ed.getBytes( rOffset, count )
}

func (ed *Desc) getTiffUnsignedShortsFromIFD( count uint ) []uint {
    if count * _ShortSize <= 4 {
        return ed.getUnsignedShorts( ed.offset, count )
    } else {
        rOffset := ed.getUnsignedLong( ed.offset )
        return ed.getUnsignedShorts( rOffset, count )
    }
}

// TODO: see if using uint32 & int32 would make more sense...
func (ed *Desc) getUnsignedLong( offset uint ) uint {
    if ed.lEndian {
        return (uint(ed.data[offset+3]) << 24) + (uint(ed.data[offset+2]) << 16) +
                (uint(ed.data[offset+1]) << 8) + uint(ed.data[offset])
    }
    return (uint(ed.data[offset]) << 24) + (uint(ed.data[offset+1]) << 16) +
            (uint(ed.data[offset+2]) << 8) + uint(ed.data[offset+3])
}

func (ed *Desc) getSignedLong( offset uint ) int {
    if ed.lEndian {
        return int((int32(ed.data[offset+3]) << 24) + (int32(ed.data[offset+2]) << 16) +
                (int32(ed.data[offset+1]) << 8) + int32(ed.data[offset]))
    }
    return int((int32(ed.data[offset]) << 24) + (int32(ed.data[offset+1]) << 16) +
            (int32(ed.data[offset+2]) << 8) + int32(ed.data[offset+3]))
}

func (ed *Desc) getUnsignedLongs( offset, count uint ) []uint {
    vSlice := make( []uint, count )
    for i := uint(0); i < count; i++ {
        vSlice[i] = ed.getUnsignedLong( offset )
        offset += _LongSize
    }
    return vSlice
}

type rational struct {
    numerator, denominator  uint
}

func (ed *Desc) getUnsignedRational( offset uint ) rational {
    var rVal rational
    rVal.numerator = ed.getUnsignedLong( offset )
    rVal.denominator = ed.getUnsignedLong( offset + _LongSize )
    return rVal
}

func (ed *Desc) getTiffUnsignedRationalsFromIFD( count uint ) []rational {
    rOffset := ed.getUnsignedLong( ed.offset )
    vSlice := make( []rational, count )
    for i := uint(0); i < count; i++ {
        vSlice[i] = ed.getUnsignedRational( rOffset )
        rOffset += _RationalSize
    }
    return vSlice
}

type sRational struct {
    numerator, denominator  int
}

func (ed *Desc) getSignedRational( offset uint ) sRational {
    var srVal sRational
    srVal.numerator = ed.getSignedLong( offset )
    srVal.denominator = ed.getSignedLong( offset + _LongSize )
    return srVal
}

func getTiffTString( tiffT uint ) string {
    switch tiffT {
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

func (ed *Desc) checkTiffByte( name string, fType, fCount uint,
                                   f func( v byte) ) error {
    if fType != _UnsignedByte {
        return fmt.Errorf( "%s: invalid type (%s)\n", name, getTiffTString( fType ) )
    }
    if fCount != 1 {
        return fmt.Errorf( "%s: invalid count (%d)\n", name, fCount )
    }
    if ed.print {
        value := ed.getByte( ed.offset )
        if f == nil {
            fmt.Printf( "    %s: %d\n", name, value )
        } else {
            fmt.Printf( "    %s: ", name )
            f( value )
        }        
    }
    return nil
}

func (ed *Desc) checkTiffAscii( name string, fType, fCount uint ) error {
    if fType != _ASCIIString {
        return fmt.Errorf( "%s: invalid type (%s)\n",
                           name, getTiffTString( fType ) )
    }
    if ed.print {
        var text string
        if fCount <= 4 {
            text = ed.getASCIIString( ed.offset, fCount )
        } else {
            offset := ed.getUnsignedLong( ed.offset )
            text = ed.getASCIIString( offset, fCount )
        }
        fmt.Printf( "    %s: %s\n", name, text )
    }
    return nil
}

func (ed *Desc) checkTiffUnsignedShort( name string, fType, fCount uint,
                                            f func( v uint) ) error {
    if fType != _UnsignedShort {
        return fmt.Errorf( "%s: invalid type (%s)\n", name, getTiffTString( fType ) )
    }
    if fCount != 1 {
        return fmt.Errorf( "%s: invalid count (%d)\n", name, fCount )
    }
    if ed.print {
        value := ed.getUnsignedShort( ed.offset )
        if f == nil {
            fmt.Printf( "    %s: %d\n", name, value )
        } else {
            fmt.Printf( "    %s: ", name )
            f( value )
        }        
    }
    return nil
}

func (ed *Desc) checkTiffUnsignedShorts( name string,
                                            fType, fCount uint ) error {
    if fType != _UnsignedShort {
        return fmt.Errorf( "%s: invalid type (%s)\n",
                            name, getTiffTString( fType ) )
    }
    if ed.print {
        values := ed.getTiffUnsignedShortsFromIFD( fCount )
        fmt.Printf( "    %s:", name )
        for _, v := range values {
            fmt.Printf( " %d", v )
        }
        fmt.Printf( "\n");
    }
    return nil
}

func (ed *Desc) checkTiffUnsignedLong( name string,
                                           fType, fCount uint,
                                           f func( v uint) ) error {
    if fType != _UnsignedLong {
        return fmt.Errorf( "%s: invalid type (%s)\n", name, getTiffTString( fType ) )
    }
    if fCount != 1 {
        return fmt.Errorf( "%s: invalid count (%d)\n", name, fCount )
    }
    if ed.print {
        value := ed.getUnsignedLong( ed.offset )
        if f == nil {
            fmt.Printf( "    %s: %d\n", name, value )
        } else {
            fmt.Printf( "    %s: ", name )
            f( value )
        }        
    }
    return nil
}

func (ed *Desc) checkTiffSignedLong( name string,
                                         fType, fCount uint,
                                         f func( v int) ) error {
    if fType != _SignedLong {
        return fmt.Errorf( "%s: invalid type (%s)\n", name, getTiffTString( fType ) )
    }
    if fCount != 1 {
        return fmt.Errorf( "%s: invalid count (%d)\n", name, fCount )
    }
    if ed.print {
        value := ed.getSignedLong( ed.offset )
        if f == nil {
            fmt.Printf( "    %s: %d\n", name, value )
        } else {
            fmt.Printf( "    %s: ", name )
            f( value )
        }        
    }
    return nil
}

func (ed *Desc) checkTiffUnsignedRational( name string, 
                                               fType, fCount uint,
                                               f func( v rational ) ) error {
    if fType != _UnsignedRational {
        return fmt.Errorf( "%s: invalid type (%s)\n",
                            name, getTiffTString( fType ) )
    }
    if fCount != 1 {
        return fmt.Errorf( "%s: invalid count (%d)\n", name, fCount )
    }
    if ed.print {
        // a rational never fits directly in valOffset (requires more than 4 bytes)
        offset := ed.getUnsignedLong( ed.offset )
        v := ed.getUnsignedRational( offset )
        if f == nil {
            fmt.Printf( "    %s: %d/%d=%f\n", name, v.numerator, v.denominator,
                        float32(v.numerator)/float32(v.denominator) )
        } else {
            fmt.Printf( "    %s: ", name )
            f( v )
        }
    }
    return nil
}

func (ed *Desc) checkTiffSignedRational( name string,
                                             fType, fCount uint,
                                             f func( v sRational ) ) error {
    if fType != _SignedRational {
        return fmt.Errorf( "%s: invalid type (%s)\n",
                            name, getTiffTString( fType ) )
    }
    if fCount != 1 {
        return fmt.Errorf( "%s: invalid count (%d)\n", name, fCount )
    }
    if ed.print {
        // a rational never fits directly in valOffset (requires more than 4 bytes)
        offset := ed.getUnsignedLong( ed.offset )
        v := ed.getSignedRational( offset )
        if f == nil {
            fmt.Printf( "    %s: %d/%d=%f\n", name, v.numerator, v.denominator,
                        float32(v.numerator)/float32(v.denominator) )
        } else {
            fmt.Printf( "    %s: ", name )
            f( v )
        }
    }
    return nil
}

func (ed *Desc) checkTiffUnsignedRationals( name string,
                                                fType, fCount uint ) error {
    if fType != _UnsignedRational {
        return fmt.Errorf( "%s: invalid type (%s)\n",
                            name, getTiffTString( fType ) )
    }
    if ed.print {
        // a rational never fits directly in valOffset (requires 8 bytes)
        values := ed.getTiffUnsignedRationalsFromIFD( fCount )
        fmt.Printf( "    %s:", name )
        for _, v := range values {
            fmt.Printf( " %d/%d", v.numerator, v.denominator,
                        float32(v.numerator)/float32(v.denominator) )
        }
        fmt.Printf( "\n");
    }
    return nil
}

const (
    _PRIMARY    = 0     // namespace for IFD0, first IFD
    _THUMBNAIL  = 1     // namespace for IFD1 pointed to by IFD0
    _EXIF       = 2     // exif namespace, pointed to by IFD0
    _GPS        = 3     // gps namespace, pointed to by IFD0
    _IOP        = 4     // Interoperability namespace, pointed to by Exif IFD

    _APPLE      = 5     // one IFD for each maker note
)

var IfdNames = [...]string{ "Primary Image data", "Thumbnail Image data",
                            "Exif data", "GPS data", "Interoperability data",
                            "Apple data" }

const (                                     // _PRIMARY & _THUMBNAIL IFD tags
//    _NewSubfileType             = 0xfe    // unused in Exif files
//    _SubfileType                = 0xff    // unused in Exif files
    _ImageWidth                 = 0x100
    _ImageLength                = 0x101
    _BitsPerSample              = 0x102
    _Compression                = 0x103

    _PhotometricInterpretation  = 0x106
    _Threshholding              = 0x107
    _CellWidth                  = 0x108
    _CellLength                 = 0x109
    _FillOrder                  = 0x10a

    _DocumentName               = 0x10d
    _ImageDescription           = 0x10e
    _Make                       = 0x10f
    _Model                      = 0x110
    _StripOffsets               = 0x111
    _Orientation                = 0x112

    _SamplesPerPixel            = 0x115
    _RowsPerStrip               = 0x116
    _StripByteCounts            = 0x117
    _MinSampleValue             = 0x118
    _MaxSampleValue             = 0x119
    _XResolution                = 0x11a
    _YResolution                = 0x11b
    _PlanarConfiguration        = 0x11c
    _PageName                   = 0x11d
    _XPosition                  = 0x11e
    _YPosition                  = 0x11f
    _FreeOffsets                = 0x120
    _FreeByteCounts             = 0x121
    _GrayResponseUnit           = 0x122
    _GrayResponseCurve          = 0x123
    _T4Options                  = 0x124
    _T6Options                  = 0x125

    _ResolutionUnit             = 0x128
    _PageNumber                 = 0x129

    _TransferFunction           = 0x12d

    _Software                   = 0x131
    _DateTime                   = 0x132

    _Artist                     = 0x13b
    _HostComputer               = 0x13c
    _Predictor                  = 0x13d
    _WhitePoint                 = 0x13e
    _PrimaryChromaticities      = 0x13f
    _ColorMap                   = 0x140
    _HalftoneHints              = 0x141
    _TileWidth                  = 0x142
    _TileLength                 = 0x143
    _TileOffsets                = 0x144
    _TileByteCounts             = 0x145

    _InkSet                     = 0x14c
    _InkNames                   = 0x14d
    _NumberOfInks               = 0x14e

    _DotRange                   = 0x150
    _TargetPrinter              = 0x151
    _ExtraSamples               = 0x152
    _SampleFormat               = 0x153
    _SMinSampleValue            = 0x154
    _SMaxSampleValue            = 0x155
    _TransferRange              = 0x156

    _JPEGProc                   = 0x200
    _JPEGInterchangeFormat      = 0x201
    _JPEGInterchangeFormatLength = 0x202
    _JPEGRestartInterval        = 0x203

    _JPEGLosslessPredictors     = 0x205
    _JPEGPointTransforms        = 0x206
    _JPEGQTables                = 0x207
    _JPEGDCTables               = 0x208
    _JPEGACTables               = 0x209

    _YCbCrCoefficients          = 0x211
    _YCbCrSubSampling           = 0x212
    _YCbCrPositioning           = 0x213
    _ReferenceBlackWhite        = 0x214

    _Copyright                  = 0x8298

    _ExifIFD                    = 0x8769

    _GpsIFD                     = 0x8825

    _Padding                    = 0xea1c    // May be used in IFD0, IFD1 and Exif IFD?
)

func (ed *Desc) checkTiffCompression( ifd, fType, fCount uint ) error {
/*
    Exif2-2: optional in Primary IFD and in thumbnail IFD
When a primary image is JPEG compressed, this designation is not necessary and is omitted.
When thumbnails use JPEG compression, this tag value is set to 6.
*/
    fmtCompression := func( v uint ) {
        var cString string
        var cType Compression
        switch( v ) {
        case 1: cString = "No compression"; cType = NotCompressed
        case 2: cString = "CCITT 1D modified Huffman RLE"; cType = CCITT_1D
        case 3: cString = "CCITT Group 3 fax encoding"; cType = CCITT_Group3
        case 4: cString = "CCITT Group 4 fax encoding"; cType = CCITT_Group4
        case 5: cString = "LZW"; cType = LZW
        case 6: cString = "JPEG"; cType = JPEG
        case 7: cString = "JPEG (Technote2)"; cType = JPEG_Technote2
        case 8: cString = "Deflate"; cType = Deflate
        case 9: cString = "RFC 2301 (black and white JBIG)."; cType = RFC_2301_BW_JBIG
        case 10: cString = "RFC 2301 (color JBIG)."; cType = RFC_2301_Color_JBIG
        case 32773: cString = "PackBits compression (Macintosh RLE)"; cType = PackBits
        default:
            fmt.Printf( "Illegal compression (%d)\n", v )
            cType = Undefined
            return
        }
        fmt.Printf( "%s\n", cString )
        if ifd == _PRIMARY {
            if v != 6 {
                fmt.Printf("    Warning: non-JPEG compression specified in a JPEG file\n" )
            } else {
                fmt.Printf("    Warning: Exif2-2 specifies that in case of JPEG picture compression be omited\n")
            }
        } else {    // _THUMBNAIL
            ed.tType = cType    // remember thumnail compression type
        }
    }
    return ed.checkTiffUnsignedShort( "Compression", fType, fCount, fmtCompression )
}

func (ed *Desc) checkTiffOrientation( ifd, fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var oString string
        switch( v ) {
        case 1: oString = "Row #0 Top, Col #0 Left"
        case 2: oString = "Row #0 Top, Col #0 Right"
        case 3: oString = "Row #0 Bottom, Col #0 Right"
        case 4: oString = "Row #0 Bottom, Col #0 Left"
        case 5: oString = "Row #0 Left, Col #0 Top"
        case 6: oString = "Row #0 Right, Col #0 Top"
        case 7: oString = "Row #0 Right, Col #0 Bottom"
        case 8: oString = "Row #0 Left, Col #0 Bottom"
        default:
            fmt.Printf( "Illegal orientation (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", oString )
    }
    return ed.checkTiffUnsignedShort( "Orientation", fType, fCount, fmtv )
}

func (ed *Desc) checkTiffResolutionUnit( ifd, fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var ruString string
        switch( v ) {
        case 1 : ruString = "Dots per Arbitrary unit"
        case 2 : ruString = "Dots per Inch"
        case 3 : ruString = "Dots per Cm"
        default:
            fmt.Printf( "Illegal resolution unit (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", ruString )
    }
    return ed.checkTiffUnsignedShort( "ResolutionUnit", fType, fCount, fmtv )
}

func (ed *Desc) checkTiffYCbCrPositioning( ifd, fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var posString string
        switch( v ) {
        case 1 : posString = "Centered"
        case 2 : posString = "Cosited"
        default:
            fmt.Printf( "Illegal positioning (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", posString )
    }
    return ed.checkTiffUnsignedShort( "YCbCrPositioning", fType, fCount, fmtv )
}

func (ed *Desc) checkPadding( fType, fCount uint ) error {
    if ed.print {
        fmt.Printf("    Padding: %d bytes - ignored\n", fCount )
    }
    return nil
}

func (ed *Desc) checkTiffTag( ifd, tag, fType, fCount uint ) error {
    switch tag {
    case _Compression:
        return ed.checkTiffCompression( ifd, fType, fCount )
    case _ImageDescription:
        return ed.checkTiffAscii( "ImageDescription", fType, fCount )
    case _Make:
        return ed.checkTiffAscii( "Make", fType, fCount )
    case _Model:
        return ed.checkTiffAscii( "Model", fType, fCount )
    case _Orientation:
        return ed.checkTiffOrientation( ifd, fType, fCount )
    case _XResolution:
        return ed.checkTiffUnsignedRational( "XResolution", fType, fCount, nil )
    case _YResolution:
        return ed.checkTiffUnsignedRational( "YResolution", fType, fCount, nil )
    case _ResolutionUnit:
        return ed.checkTiffResolutionUnit( ifd, fType, fCount )
    case _Software:
        return ed.checkTiffAscii( "Software", fType, fCount )
    case _DateTime:
        return ed.checkTiffAscii( "Date", fType, fCount )
    case _HostComputer:
        return ed.checkTiffAscii( "HostComputer", fType, fCount )
    case _YCbCrPositioning:
        return ed.checkTiffYCbCrPositioning( ifd, fType, fCount )
    case _Copyright:
        return ed.checkTiffAscii( "Copyright", fType, fCount )

    case _Padding:
        return ed.checkPadding( fType, fCount )
    }
    return fmt.Errorf( "checkTiffTag: unknown or unsupported tag (%#02x) @offset %#04x type %s count %d\n",
                       tag, ed.offset-8, getTiffTString( fType ), fCount )
}

const (                                     // _EXIF IFD specific tags
    _ExposureTime               = 0x829a

    _FNumber                    = 0x829d

    _ExposureProgram            = 0x8822

    _ISOSpeedRatings            = 0x8827

    _ExifVersion                = 0x9000

    _DateTimeOriginal           = 0x9003
    _DateTimeDigitized          = 0x9004

    _OffsetTime                 = 0x9010    // perhaps obsolete - not in standard
    _OffsetTimeOriginal         = 0x9011    // perhaps obsolete - not in standard
    _OffsetTimeDigitized        = 0x9012    // perhaps obsolete - not in standard

    _ComponentsConfiguration    = 0x9101
    _CompressedBitsPerPixel     = 0x9102

    _ShutterSpeedValue          = 0x9201
    _ApertureValue              = 0x9202
    _BrightnessValue            = 0x9203
    _ExposureBiasValue          = 0x9204
    _MaxApertureValue           = 0x9205
    _SubjectDistance            = 0x9206
    _MeteringMode               = 0x9207
    _LightSource                = 0x9208
    _Flash                      = 0x9209
    _FocalLength                = 0x920a

    _SubjectArea                = 0x9214

    _MakerNote                  = 0x927c

    _UserComment                = 0x9286

    _SubsecTime                 = 0x9290
    _SubsecTimeOriginal         = 0x9291
    _SubsecTimeDigitized        = 0x9292

    _FlashpixVersion            = 0xa000
    _ColorSpace                 = 0xa001
    _PixelXDimension            = 0xa002
    _PixelYDimension            = 0xa003

    _InteroperabilityIFD        = 0xa005

    _SubjectLocation            = 0xa214
    _SensingMethod              = 0xa217

    _FileSource                 = 0xa300
    _SceneType                  = 0xa301
    _CFAPattern                 = 0xa302

    _CustomRendered             = 0xa401
    _ExposureMode               = 0xa402
    _WhiteBalance               = 0xa403
    _DigitalZoomRatio           = 0xa404
    _FocalLengthIn35mmFilm      = 0xa405
    _SceneCaptureType           = 0xa406
    _GainControl                = 0xa407
    _Contrast                   = 0xa408
    _Saturation                 = 0xa409
    _Sharpness                  = 0xa40a

    _SubjectDistanceRange       = 0xa40c

    _ImageUniqueID              = 0xa420

    _LensSpecification          = 0xa432
    _LensMake                   = 0xa433
    _LensModel                  = 0xa434
)

func (ed *Desc) checkExifVersion( fType, fCount uint ) error {
  // special case: tiff type is undefined, but it is actually ASCII
    if fType != _Undefined {
        return fmt.Errorf( "ExifVersion: invalid byte type (%s)\n", getTiffTString( fType ) )
    }
    return ed.checkTiffAscii( "ExifVersion", _ASCIIString, fCount )
}

func (ed *Desc) checkExifExposureTime( fType, fCount uint ) error {
    fmtv := func( v rational ) {
        fmt.Printf( "%f seconds\n", float32(v.numerator)/float32(v.denominator) )
    }
    return ed.checkTiffUnsignedRational( "ExposureTime", fType, fCount, fmtv )
}

func (ed *Desc) checkExifExposureProgram( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var epString string
        switch v {
        case 0 : epString = "Undefined"
        case 1 : epString = "Manual"
        case 2 : epString = "Normal program"
        case 3 : epString = "Aperture priority"
        case 4 : epString = "Shutter priority"
        case 5 : epString = "Creative program (biased toward depth of field)"
        case 6 : epString = "Action program (biased toward fast shutter speed)"
        case 7 : epString = "Portrait mode (for closeup photos with the background out of focus)"
        case 8 : epString = "Landscape mode (for landscape photos with the background in focus) "
        default:
            fmt.Printf( "Illegal Exposure Program (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", epString )
    }
    return ed.checkTiffUnsignedShort( "ExposureProgram", fType, fCount, fmtv )
}

func (ed *Desc) checkExifComponentsConfiguration( fType, fCount uint ) error {
    if fType != _Undefined {  // special case: tiff type is undefined, but it is actually bytes
        return fmt.Errorf( "ComponentsConfiguration: invalid type (%s)\n", getTiffTString( fType ) )
    }
    if fCount != 4 {
        return fmt.Errorf( "ComponentsConfiguration: invalid byte count (%d)\n", fCount )
    }
    if ed.print {
        bSlice := ed.getBytes( ed.offset, fCount )
        var config strings.Builder
        for _, b := range bSlice {
            switch b {
            case 0:
            case 1: config.WriteByte( 'Y' )
            case 2: config.WriteString( "Cb" )
            case 3: config.WriteString( "Cr" )
            case 4: config.WriteByte( 'R' )
            case 5: config.WriteByte( 'G' )
            case 6: config.WriteByte( 'B' )
            default: config.WriteByte( '?' )
            }
        }
        fmt.Printf( "    ComponentsConfiguration: %s\n", config.String() )
    }
    return nil
}

func (ed *Desc) checkExifSubjectDistance( fType, fCount uint ) error {
    fmtv := func( v rational ) {
        if v.numerator == 0 {
            fmt.Printf( "Unknown\n" )
        } else if v.numerator == 0xffffffff {
            fmt.Printf( "Infinity\n" )
        } else {
            fmt.Printf( "%f meters\n", float32(v.numerator)/float32(v.denominator) )
        }
    }
    return ed.checkTiffUnsignedRational( "SubjectDistance", fType, fCount, fmtv )
}

func (ed *Desc) checkExifMeteringMode( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var mmString string
        switch v {
        case 0 : mmString = "Unknown"
        case 1 : mmString = "Average"
        case 2 : mmString = "CenterWeightedAverage program"
        case 3 : mmString = "Spot"
        case 4 : mmString = "MultiSpot"
        case 5 : mmString = "Pattern"
        case 6 : mmString = "Partial"
        case 255: mmString = "Other"
        default:
            fmt.Printf( "Illegal Metering Mode (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", mmString )
    }
    return ed.checkTiffUnsignedShort( "MeteringMode", fType, fCount, fmtv )
}

func (ed *Desc) checkExifLightSource( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var lsString string
        switch v {
        case 0 : lsString = "Unknown"
        case 1 : lsString = "Daylight"
        case 2 : lsString = "Fluorescent"
        case 3 : lsString = "Tungsten (incandescent light)"
        case 4 : lsString = "Flash"
        case 9 : lsString = "Fine weather"
        case 10 : lsString = "Cloudy weather"
        case 11 : lsString = "Shade"
        case 12 : lsString = "Daylight fluorescent (D 5700 - 7100K)"
        case 13 : lsString = "Day white fluorescent (N 4600 - 5400K)"
        case 14 : lsString = "Cool white fluorescent (W 3900 - 4500K)"
        case 15 : lsString = "White fluorescent (WW 3200 - 3700K)"
        case 17 : lsString = "Standard light A"
        case 18 : lsString = "Standard light B"
        case 19 : lsString = "Standard light C"
        case 20 : lsString = "D55"
        case 21 : lsString = "D65"
        case 22 : lsString = "D75"
        case 23 : lsString = "D50"
        case 24 : lsString = "ISO studio tungsten"
        case 255: lsString = "Other light source"
        default:
            fmt.Printf( "Illegal light source (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", lsString )
    }
    return ed.checkTiffUnsignedShort( "LightSource", fType, fCount, fmtv )
}

func (ed *Desc) checkExifFlash( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var fString string
        switch v {
        case 0x00 : fString = "Flash did not fire"
        case 0x01 : fString = "Flash fired"
        case 0x05 : fString = "Flash fired, strobe return light not detected"
        case 0x07 : fString = "Flash fired, strobe return light detected"
        case 0x09 : fString = "Flash fired, compulsory flash mode, return light not detected"
        case 0x0F : fString = "Flash fired, compulsory flash mode, return light detected"
        case 0x10 : fString = "Flash did not fire, compulsory flash mode"
        case 0x18 : fString = "Flash did not fire, auto mode"
        case 0x19 : fString = "Flash fired, auto mode"
        case 0x1D : fString = "Flash fired, auto mode, return light not detected"
        case 0x1F : fString = "Flash fired, auto mode, return light detected"
        case 0x20 : fString = "No flash function"
        case 0x41 : fString = "Flash fired, red-eye reduction mode"
        case 0x45 : fString = "Flash fired, red-eye reduction mode, return light not detected"
        case 0x47 : fString = "Flash fired, red-eye reduction mode, return light detected"
        case 0x49 : fString = "Flash fired, compulsory flash mode, red-eye reduction mode"
        case 0x4D : fString = "Flash fired, compulsory flash mode, red-eye reduction mode, return light not detected"
        case 0x4F : fString = "Flash fired, compulsory flash mode, red-eye reduction mode, return light detected"
        case 0x59 : fString = "Flash fired, auto mode, red-eye reduction mode"
        case 0x5D : fString = "Flash fired, auto mode, return light not detected, red-eye reduction mode"
        case 0x5F : fString = "Flash fired, auto mode, return light detected, red-eye reduction mode"
        default:
            fmt.Printf( "Illegal Flash (%#02x)\n", v )
            return
        }
        fmt.Printf( "%s\n", fString )
    }
    return ed.checkTiffUnsignedShort( "Flash", fType, fCount, fmtv )
}

func (ed *Desc) checkExifSubjectArea( fType, fCount uint ) error {
    if fCount < 2 && fCount > 4 {
        return fmt.Errorf( "ComponentsConfiguration: invalid count (%d)\n", fCount )
    }
    if ed.print {
        loc := ed.getTiffUnsignedShortsFromIFD( fCount )
        switch fCount {
        case 2:
            fmt.Printf( "    Subject Area: Point x=%d, y=%d\n", loc[0], loc[1] )
        case 3:
            fmt.Printf( "    Subject Area: Circle center x=%d, y=%d diameter=%d\n",
                        loc[0], loc[1], loc[2] )
        case 4:
            fmt.Printf( "    Subject Area: Rectangle center x=%d, y=%d width=%d height=%d\n",
                        loc[0], loc[1], loc[2], loc[3] )
        }
    }
    return nil
}

const (             // Apple Maker note IFD
    _Apple001                   = 0x0001  // should be _SignedLong
    _Apple002                   = 0x0002  // should be _Undefined, actually _UnsignedLong offset to a plist
    _AppleRunTime               = 0x0003  // offset to subirectory runtime
    _Apple004                   = 0x0004  // 1 _SignedLong, either 0 or 1
    _Apple005                   = 0x0005  // 1 _SignedLong
    _Apple006                   = 0x0006  // 1 _SignedLong
    _Apple007                   = 0x0007  // 1 _SignedLong (1)
    _AppleAccelerationVector    = 0x0008  // 3 _SignedRational
    _Apple009                   = 0x0009  // 1 _SignedLong
    _AppleHDRImageType          = 0x000a  // 1 _SignedLong: 2=iPad mini 2, 3=HDR Image, 4=Original Image
    _BurstUUID                  = 0x000b  // 1 _ASCIIString unique ID for all images in a burst
    _Apple00c                   = 0x000c  // 2 _SignedRational
    _Apple00d                   = 0x000d  // 1 _SignedLong
    _Apple00e                   = 0x000e  // 1 _SignedLong Orientation? 0=landscape? 4=portrait?
    _Apple00f                   = 0x000f  // 1 _SignedLong
    _Apple010                   = 0x0010  // 1 _SignedLong
    _AppleMediaGroupUUID        = 0x0011  // 1 _ASCIIString

    _Apple0014                  = 0x0014  // 1 _SignedLong 1, 2, 3, 4, 5 (iPhone 6s, iOS 6.1)
    _AppleImageUniqueID         = 0x0015  // 1 _ASCIIString
    _Apple0016                  = 0x0016  // 1 _ASCIIString [29]"AXZ6pMTOh2L+acSh4Kg630XCScoO\0"
    _Apple0017                  = 0x0017  // 1 _SignedLong

    _Apple0019                  = 0x0019  // 1 _SignedLong
    _Apple001a                  = 0x001a  // 1 _ASCIIString [6]"q825s\0"

    _Apple001f                  = 0x001f  // 1 _SignedLong

/*
    0x0008 => 'AccelerationVector',
        # Note: the directions are contrary to the Apple documentation (which have the
        # signs of all axes reversed -- apparently the Apple geeks aren't very good
        # with basic physics, and don't understand the concept of acceleration.  See
        # http://nscookbook.com/2013/03/ios-programming-recipe-19-using-core-motion-to-access-gyro-and-accelerometer/
        # for one of the few correct descriptions of this).  Note that this leads to
        # a left-handed coordinate system for acceleration.
            XYZ coordinates of the acceleration vector in units of g.  As viewed from
            the front of the phone, positive X is toward the left side, positive Y is
            toward the bottom, and positive Z points into the face of the phone

# PLIST-format CMTime structure (ref PH)
# (CMTime ref https://developer.apple.com/library/ios/documentation/CoreMedia/Reference/CMTime/Reference/reference.html)
%Image::ExifTool::Apple::RunTime = (
    PROCESS_PROC => \&Image::ExifTool::PLIST::ProcessBinaryPLIST,
    GROUPS => { 0 => 'MakerNotes', 2 => 'Image' },
    NOTES => q{
        This PLIST-format information contains the elements of a CMTime structure
        representing the amount of time the phone has been running since the last
        boot, not including standby time.
    },
    timescale => { Name => 'RunTimeScale' }, # (seen 1000000000 --> ns)
    epoch     => { Name => 'RunTimeEpoch' }, # (seen 0)
    value     => { Name => 'RunTimeValue' }, # (should divide by RunTimeScale to get seconds)
    flags => {
        Name => 'RunTimeFlags',
        PrintConv => { BITMASK => {
            0 => 'Valid',
            1 => 'Has been rounded',
            2 => 'Positive infinity',
            3 => 'Negative infinity',
            4 => 'Indefinite',

*/
)

func dumpData( header string, data []byte ) {
    fmt.Printf( "    %s:\n", header )
    for i := 0; i < len(data); i += 16 {
        fmt.Printf("      %#04x: ", i );
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

func printPlist( name string, pList []byte ) error {
    if ! bytes.Equal( pList[:8], []byte("bplist00") ) {
        return fmt.Errorf( "%s (PList): not an Apple plist (%s)\n", name, string(pList[:8]) )
    }

    // get trailer info first
    trailer := len( pList ) - 32
    if trailer < 8 {
        return fmt.Errorf( "%s (PList): wrong size for an Apple plist (%d)\n", name, len( pList) )
    }

    getBigEndianUInt64 := func( b []byte ) (v uint64) {
        v = (uint64(b[0]) << 56) + (uint64(b[1]) << 48) +
            (uint64(b[2]) << 40) + (uint64(b[3]) << 32) +
            (uint64(b[4]) << 24) + (uint64(b[5]) << 16) +
            (uint64(b[6]) << 8) + uint64(b[7])
        return
    }

    // Skip 5 unused bytes + 1-byte _sortVersion
    offsetEntrySize := pList[trailer+6]     // 1-byte _offsetIntSize
//    TODO: used in arrays, sets and dictionaries only (TBI)
//    objectRefSize := pList[trailer+7]       // 2-byte _objectRefSize
    // 8-byte _numObjects
    numObjects := getBigEndianUInt64( pList[trailer+8:trailer+16] )
    // 8-byte _topObject
    topObjectOffset := getBigEndianUInt64( pList[trailer+16:trailer+24] )
    // 8-byte _offsetTableOffset
    offsetTableOffset := getBigEndianUInt64( pList[trailer+24:trailer+32] )
/*
    fmt.Printf( "offsetEntrySize: %d bytes\n", offsetEntrySize )
    fmt.Printf( "objectRefSize: %d bytes\n", objectRefSize )
    fmt.Printf( "numObjects: %d\n", numObjects )
    fmt.Printf( "topObjectOffset: %d\n", topObjectOffset )
    fmt.Printf( "offsetTableOffset: %d\n", offsetTableOffset )
*/
    getOffsetTableEntry := func( offset uint64 ) uint64 { // assume worst case size
        o := offsetTableOffset+offset
        if o > uint64(trailer) {
            panic( fmt.Sprintf("Invalid offsetTable offset %#04x\n", o))
        }
        switch( offsetEntrySize ) {
        case 1: return uint64(pList[o])
        case 2: return (uint64(pList[o]) << 8) + uint64(pList[o+1])
        case 4: return (uint64(pList[o]) << 24) + (uint64(pList[o+1]) << 16) +
                       (uint64(pList[o+2]) << 8) + uint64(pList[o+3])
        case 8: return getBigEndianUInt64( pList[o:o+8] )
        default:
            panic(fmt.Sprintf("invalid offsetEntrySize %d\n", o))
        }
    }

    getOSize := func( object []byte, offset uint64 ) (uint64, uint64) { // size 0 is an error
        size := uint64(object[offset] & 0x0f)
        if size == 0x0f {
            eSize := uint64(object[offset+1])   // encoded size in bytes
            if (eSize & 0xf0) != 0x10 {
                return offset, 0                // with size 0 => error
            }
            eSize = 1 << (eSize & 0x0f)
            offset += 1

            size = 0
            for j := uint64(0); j < eSize; j++ {
                offset += 1
                size = (size << 8) + uint64(pList[offset])
            }
        }
        return offset, size
    }

    printObject := func ( object uint64 ) error {

        switch pList[object] & 0xf0 {
        case 0x00:  // special
            switch pList[object] & 0x0f {
            case 0x00:  fmt.Printf( "      null\n" )
            case 0x01:  fmt.Printf( "      true\n" )
            case 0x08:  fmt.Printf( "      false\n" )
            case 0x0f:  fmt.Printf( "      fill\n" )
            }

        case 0x10:  // int, less significant 4 bits are exponent of following size
            size := 1 << (pList[object] & 0x0f)
            v := uint64(pList[object+1])
            switch size {
            case 1:
                fmt.Printf( "      byte %d\n", v )
            case 2:
                v = (v << 8) + uint64(pList[object+2])
                fmt.Printf( "      short %d\n", v )
            case 4:
                v = (v << 24) + (uint64(pList[object+2]) << 16) +
                    (uint64(pList[object+3]) << 8) + uint64(pList[object+4])
                fmt.Printf( "      long %d\n", v )
            case 8:
                v = (v << 56) + (uint64(pList[object+2]) << 48) +
                    (uint64(pList[object+3]) << 40) + (uint64(pList[object+4]) << 32) +
                    (uint64(pList[object+5]) << 24) + (uint64(pList[object+6]) << 16) +
                    (uint64(pList[object+7]) << 8) + uint64(pList[object+8])
                fmt.Printf( "      long long %d\n", v )
            }

        case 0x20:  // real, less significant 4 bits are exponent of following size
            size := 1 << (pList[object] & 0x0f)
//            v := uint64(pList[i+1])
            fmt.Printf( "      real (size %d)\n", size )

        case 0x30:  // should be date as 8-byte float
            if pList[object] != 0x33 {
                return fmt.Errorf( "%s (PList): invalid object (%#02x)\n", name, pList[object] )
            }
            // TODO
            fmt.Print( "      float\n")

        case 0x40:  // raw data array
            _, size := getOSize( pList, object )
            if size == 0 {
                return fmt.Errorf( "%s (PList): invalid data size encoding (%#02x)\n", name, pList[object+1] )
            }
            fmt.Printf( "      data (size %d)\n", size )

        case 0x50:  // ASCII string
            j, size := getOSize( pList, object )
            if size == 0 {
                return fmt.Errorf( "%s (PList): invalid data size encoding (%#02x)\n", name, pList[object+1] )
            }
            fmt.Printf( "      ASCII (size %d): %s\n", size, pList[j+1-size:j+1] )

        case 0x60:  // Unicode string
            _, size := getOSize( pList, object )
            if size == 0 {
                return fmt.Errorf( "%s (PList): invalid data size encoding (%#02x)\n", name, pList[object+1] )
            }
            fmt.Printf( "      Unicode (size %d)\n", size )

        case 0x80:  // uid
            size := 1 + (pList[object] & 0x0f)
            fmt.Printf( "      UID (size %d)\n", size )

        case 0xa0:  // Array
            _, size := getOSize( pList, object )
            if size == 0 {
                return fmt.Errorf( "%s (PList): invalid array size encoding (%#02x)\n", name, pList[object+1] )
            }
            fmt.Printf( "      Array (size %d)\n", size )

        case 0xc0:  // set
            fmt.Printf( "      Set\n" )

        case 0xd0:  // dict
            fmt.Printf( "      Dict\n" )
        default:
            return fmt.Errorf( "%s (PList): invalid object (%#02x)\n", name, pList[object] )
        }
        return nil
    }


    topObjectStart := getOffsetTableEntry( topObjectOffset )
    fmt.Printf( "pList: %d object(s) - top level object starts at offset %#04x in plist\n",
                numObjects, topObjectStart )

    err := printObject( topObjectStart )
    if err != nil {
        return err
    }
    return nil
}

func (ed *Desc) checkApplePLIST( name string, fType, fCount uint ) error {

    if fType != _Undefined {
        return fmt.Errorf( "%s (PList): invalid type (%s)\n", name, getTiffTString( fType ) )
    }

    if ed.print {
        pList := ed.getBytesFromIFD( fCount )
        dumpData( name, pList )
        err := printPlist( name, pList )
        if err != nil {
            return err
        }
    }
    return nil
}

func (ed *Desc) checkAppleAccelerationVector( fType, fCount uint ) error {
    if fType != _SignedRational {
        return fmt.Errorf( "AccelerationVector: invalid type (%s)\n", getTiffTString( fType ) )
    }
    if fCount != 3 {
        return fmt.Errorf( "AccelerationVector: invalid count (%d)\n", fCount )
    }

    if ed.print {
        offset := ed.getUnsignedLong( ed.offset )
        v := ed.getSignedRational( offset )
        fmt.Printf( "Acceleration Vector X: %d %d\n", v.numerator, v.denominator )
        v = ed.getSignedRational( offset + 8 )
        fmt.Printf( "Acceleration Vector Y: %d %d\n", v.numerator, v.denominator )
        v = ed.getSignedRational( offset + 16 )
        fmt.Printf( "Acceleration Vector Z: %d %d\n", v.numerator, v.denominator )
    }
    return nil
}

func (ed *Desc) checkApple( ifd, tag, fType, fCount uint ) error {
    fmt.Printf( "tag %#02x fType %d fCount %d fOffset %#04x\n", tag, fType, fCount, ed.offset )

    switch tag {
    case _Apple001:
        return ed.checkTiffSignedLong( "Apple #0001", fType, fCount, nil )
    case _Apple002:
        return ed.checkApplePLIST( "Apple #0002", fType, fCount )
    case _AppleRunTime:
        return ed.checkApplePLIST( "RunTime", fType, fCount )
    case _Apple004:
        return ed.checkTiffSignedLong( "Apple #0004", fType, fCount, nil )
    case _Apple005:
        return ed.checkTiffSignedLong( "Apple #0005", fType, fCount, nil )
    case _Apple006:
        return ed.checkTiffSignedLong( "Apple #0006", fType, fCount, nil )
    case _Apple007:
        return ed.checkTiffSignedLong( "Apple #0007", fType, fCount, nil )
    case _AppleAccelerationVector:
        return ed.checkAppleAccelerationVector( fType, fCount )
    case _Apple009:
        panic("debug")
    }
    return nil
}

func (ed *Desc) processMakerNote( fOffset, fCount uint ) error {
    if bytes.Equal( ed.data[fOffset:fOffset+10], []byte( "Apple iOS\x00" ) ) {
        fmt.Printf("    MakerNote: Apple iOS\n" )
        offset := fOffset + 12
        count  := fCount - 12

        size, lEndian, err := getEndianess( ed.data[offset:offset+count] )
        if err != nil {
            return err
        }

        mknd := new(Desc)
        mknd.data = ed.data[fOffset:fOffset+fCount] // origin is before Apple name
        mknd.offset = 12 + size
        mknd.lEndian = lEndian
        mknd.print = ed.print

        fmt.Printf( "Apple maker notes: origin %#04x start %#04x, end %#04x, little endian %v\n",
                    fOffset, mknd.offset, fOffset + fCount, lEndian )

        _, _, _, err = mknd.checkIFD( _APPLE, -1, -1 ) 
        if err != nil {
            return err
        }
    }
    if ed.print {
        dumpData( "MakerNote", ed.data[fOffset:fOffset+fCount] )
    }
    return nil
}


func (ed *Desc) checkExifMakerNote( fType, fCount uint ) error {
    if fType != _Undefined {
        return fmt.Errorf( "MakerNote: invalid type (%s)\n", getTiffTString( fType ) )
    }
    if fCount < 4 {
        dumpData( "MakerNote", ed.data[ed.offset:ed.offset+fCount] )
    } else {
        offset := ed.getUnsignedLong( ed.offset )
        return ed.processMakerNote( offset, fCount )
    }
    return nil
}

func (ed *Desc) checkExifUserComment( fType, fCount uint ) error {
    if fType != _Undefined {
        return fmt.Errorf( "UserComment: invalid type (%s)\n", getTiffTString( fType ) )
    }
    if fCount < 8 {
        return fmt.Errorf( "UserComment: invalid count (%s)\n", fCount )
    }
    //  first 8 Bytes are the encoding
    offset := ed.getUnsignedLong( ed.offset )
    encoding := ed.getBytes( offset, 8 )
    switch encoding[0] {
    case 0x41:  // ASCII?
        if bytes.Equal( encoding, []byte{ 'A', 'S', 'C', 'I', 'I', 0, 0, 0 } ) {
            if ed.print {
                fmt.Printf( "    UserComment: ITU-T T.50 IA5 (ASCII) [%s]\n", 
                            string(ed.getBytes( offset+8, fCount-8 )) )
            }
            return nil
        }
    case 0x4a: // JIS?
        if bytes.Equal( encoding, []byte{ 'J', 'I', 'S', 0, 0, 0, 0, 0 } ) {
            if ed.print {
                fmt.Printf( "    UserComment: JIS X208-1990 (JIS):" )
                dumpData( "UserComment", ed.data[offset+8:offset+fCount] )
            }
            return nil
        }
    case 0x55:  // UNICODE?
        if bytes.Equal( encoding, []byte{ 'U', 'N', 'I', 'C', 'O', 'D', 'E', 0 } ) {
            if ed.print {
                fmt.Printf( "    UserComment: Unicode Standard:" )
                dumpData( "UserComment", ed.data[offset+8:offset+fCount] )
            }
            return nil
        }
    case 0x00:  // Undefined
        if bytes.Equal( encoding, []byte{ 0, 0, 0, 0, 0, 0, 0, 0 } ) {
            if ed.print {
                fmt.Printf( "    UserComment: Undefined encoding:" )
                dumpData( "UserComment", ed.data[offset+8:offset+fCount] )
            }
            return nil
        }
    }
    return fmt.Errorf( "UserComment: invalid encoding\n" )
}

func (ed *Desc) checkFlashpixVersion( fType, fCount uint ) error {
    if fType == _Undefined && fCount == 4 {
        return ed.checkTiffAscii( "FlashpixVersion", _ASCIIString, fCount )
    } else if fType != _Undefined {
        return fmt.Errorf( "FlashpixVersion: invalid type (%s)\n", getTiffTString( fType ) )
    }
    return fmt.Errorf( "FlashpixVersion: incorrect count (%d)\n", fCount )
}

func (ed *Desc) checkExifColorSpace( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var csString string
        switch v {
        case 1 : csString = "sRGB"
        case 65535: csString = "Uncalibrated"
        default:
            fmt.Printf( "Illegal color space (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", csString )
    }
    return ed.checkTiffUnsignedShort( "ColorSpace", fType, fCount, fmtv )
}

func (ed *Desc) checkExifDimension( name string,
                                        fType, fCount uint ) error {
    if fType == _UnsignedShort {
        return ed.checkTiffUnsignedShort( name, fType, fCount, nil )
    } else if fType == _UnsignedLong {
        return ed.checkTiffUnsignedLong( name, fType, fCount, nil )
    }
    return fmt.Errorf( "%s: invalid type (%s)\n", name, getTiffTString( fType ) )
}

func (ed *Desc) checkExifSensingMethod( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var smString string
        switch v {
        case 1 : smString = "Undefined"
        case 2 : smString = "One-chip color area sensor"
        case 3 : smString = "Two-chip color area sensor"
        case 4 : smString = "Three-chip color area sensor"
        case 5 : smString = "Color sequential area sensor"
        case 7 : smString = "Trilinear sensor"
        case 8 : smString = "Color sequential linear sensor"
        default:
            fmt.Printf( "Illegal sensing method (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", smString )
    }
    return ed.checkTiffUnsignedShort( "SensingMethod", fType, fCount, fmtv )
}

func (ed *Desc) checkExifFileSource( fType, fCount uint ) error {
    if fType != _Undefined {
        return fmt.Errorf( "FileSource: invalid type (%s)\n", getTiffTString( fType ) )
    }
    fmtv := func( v byte ) {       // expect byte
        if v != 3 {
            fmt.Printf( "Illegal file source (%d)\n", v )
            return
        }
        fmt.Printf( "Digital Still Camera (DSC)\n" )
    }
    return ed.checkTiffByte( "FileSource", _UnsignedByte, fCount, fmtv )
}

func (ed *Desc) checkExifSceneType( fType, fCount uint ) error {
    if fType != _Undefined {
        return fmt.Errorf( "SceneType: invalid type (%s)\n", getTiffTString( fType ) )
    }
    fmtv := func( v byte ) {       // expect byte
        var stString string
        switch v {
        case 1 : stString = "Directly photographed"
        default:
            fmt.Printf( "Illegal schene type (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", stString )
    }
    return ed.checkTiffByte( "SceneType", _UnsignedByte, fCount, fmtv )
}

func (ed *Desc) checkExifCFAPattern( fType, fCount uint ) error {
    if fType != _Undefined {
        return fmt.Errorf( "CFAPattern: invalid type (%s)\n", getTiffTString( fType ) )
    }
    // structure describing the color filter array (CFA)
    // 2 short words: horizontal repeat pixel unit (h), vertical repeat pixel unit (v)
    // followed by h*v bytes, each byte value indicating a color:
    // 0 RED, 1 GREEN, 2 BLUE, 3 CYAN, 4 MAGENTA, 5 YELLOW, 6 WHITE
    // Since the structure cannot fit in 4 bytes, its location is indicated by an offset
    offset := ed.getUnsignedLong( ed.offset )
    h := ed.getUnsignedShort( offset )
    v := ed.getUnsignedShort( offset + 2 )
    // however, it seems that older microsoft tools do not use the proper endianess,
    // so check here if the values are consistent with the total count:
    var swap bool
    if h * v != fCount - 4 { // if not try changing endianess
        h1 := ((h & 0xff) << 8) + (h >> 8)
        v1 := ((v & 0xff) << 8) + (v >> 8)
        if ( h1 *v1 != fCount - 4 ) {
            return fmt.Errorf( "CFAPattern: invalid repeat patterns(%d,%d)\n", h, v )
        }
        h = h1
        v = v1
        swap = true
    }
    if ed.print {
        fmt.Printf( "    CFAPattern:" )
    }
    offset += 4
    c := ed.getBytes(  offset, h * v )

    for i := uint(0); i < v; i++ {
        fmt.Printf("\n      Row %d:", i )
        for j := uint(0); j < h; j++ {
            var s string
            switch c[(i*h)+j] {
            case 0: s = "RED"
            case 1: s = "GREEN"
            case 2: s = "BLUE"
            case 3: s = "CYAN"
            case 4: s = "MAGENTA"
            case 5: s = "YELLOW"
            case 6: s = "WHITE"
            default:
                if ed.print {
                    fmt.Printf("\n")
                }
                return fmt.Errorf( "CFAPattern: invalid color (%d)\n", c[(i*h)+j] )
            }
            if ed.print {
                fmt.Printf( " %s", s )
            }
        }
    }
    if ed.print {
        fmt.Printf( "\n" )
    }
    if swap {
        fmt.Printf("      Warning: CFAPattern: incorrect endianess\n")
    }
    return nil
}

func (ed *Desc) checkExifCustomRendered( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var crString string
        switch v {
        case 0 : crString = "Normal process"
        case 1 : crString = "Custom process"
        default:
            fmt.Printf( "Illegal rendering process (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", crString )
    }
    return ed.checkTiffUnsignedShort( "CustomRendered", fType, fCount, fmtv )
}

func (ed *Desc) checkExifExposureMode( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var emString string
        switch v {
        case 0 : emString = "Auto exposure"
        case 1 : emString = "Manual exposure"
        case 3 : emString = "Auto bracket"
        default:
            fmt.Printf( "Illegal Exposure mode (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", emString )
    }
    return ed.checkTiffUnsignedShort( "ExposureMode", fType, fCount, fmtv )
}

func (ed *Desc) checkExifWhiteBalance( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var wbString string
        switch v {
        case 0 : wbString = "Auto white balance"
        case 1 : wbString = "Manual white balance"
        default:
            fmt.Printf( "Illegal white balance (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", wbString )
    }
    return ed.checkTiffUnsignedShort( "WhiteBalance", fType, fCount, fmtv )
}

func (ed *Desc) checkExifDigitalZoomRatio( fType, fCount uint ) error {
    fmv := func( v rational ) {
        if v.numerator == 0 {
            fmt.Printf( "not used\n" )
        } else if v.denominator == 0 {
            fmt.Printf( "invalid ratio denominator (0)\n" )
        } else {
            fmt.Printf( "%f\n", float32(v.numerator)/float32(v.denominator) )
        }
    }
    return ed.checkTiffUnsignedRational( "DigitalZoomRatio", fType, fCount, fmv )
}

func (ed *Desc) checkExifSceneCaptureType( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var sctString string
        switch v {
        case 0 : sctString = "Standard"
        case 1 : sctString = "Landscape"
        case 2 : sctString = "Portrait"
        case 3 : sctString = "Night scene"
        default:
            fmt.Printf( "Illegal scene capture type (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", sctString )
    }
    return ed.checkTiffUnsignedShort( "SceneCaptureType", fType, fCount, fmtv )
}

func (ed *Desc) checkExifGainControl( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var gcString string
        switch v {
        case 0 : gcString = "none"
        case 1 : gcString = "Low gain up"
        case 2 : gcString = "high gain up"
        case 3 : gcString = "low gain down"
        case 4 : gcString = "high gain down"
        default:
            fmt.Printf( "Illegal gain control (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", gcString )
    }
    return ed.checkTiffUnsignedShort( "GainControl", fType, fCount, fmtv )
}

func (ed *Desc) checkExifContrast( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var cString string
        switch v {
        case 0 : cString = "Normal"
        case 1 : cString = "Soft"
        case 2 : cString = "Hard"
        default:
            fmt.Printf( "Illegal contrast (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", cString )
    }
    return ed.checkTiffUnsignedShort( "Contrast", fType, fCount, fmtv )
}

func (ed *Desc) checkExifSaturation( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var sString string
        switch v {
        case 0 : sString = "Normal"
        case 1 : sString = "Low saturation"
        case 2 : sString = "High saturation"
        default:
            fmt.Printf( "Illegal Saturation (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", sString )
    }
    return ed.checkTiffUnsignedShort( "Saturation", fType, fCount, fmtv )
}

func (ed *Desc) checkExifSharpness( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var sString string
        switch v {
        case 0 : sString = "Normal"
        case 1 : sString = "Soft"
        case 2 : sString = "Hard"
        default:
            fmt.Printf( "Illegal Sharpness (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", sString )
    }
    return ed.checkTiffUnsignedShort( "Sharpness", fType, fCount, fmtv )
}

func (ed *Desc) checkExifDistanceRange( fType, fCount uint ) error {
    fmtv := func( v uint ) {
        var drString string
        switch v {
        case 0 : drString = "Unknown"
        case 1 : drString = "Macro"
        case 2 : drString = "Close View"
        case 3 : drString = "Distant View"
        default:
            fmt.Printf( "Illegal Distance Range (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", drString )
    }
    return ed.checkTiffUnsignedShort( "DistanceRange", fType, fCount, fmtv )
}

func (ed *Desc) checkExifLensSpecification( fType, fCount uint ) error {
// LensSpecification is an array of ordered rational values:
//  minimum focal length
//  maximum focal length
//  minimum F number in minimum focal length
//  maximum F number in maximum focal length
//  which are specification information for the lens that was used in photography.
//  When the minimum F number is unknown, the notation is 0/0.
    if fCount != 4 {
        return fmt.Errorf( "LensSpecification: invalid count (%d)\n", fCount )
    }
    if fType != _UnsignedRational {
        return fmt.Errorf( "LensSpecification: invalid type (%s)\n", getTiffTString( fType ) )
    }

    if ed.print {
        fmt.Printf( "    LensSpecification:\n" )
        offset := ed.getUnsignedLong( ed.offset )

        v := ed.getUnsignedRational( offset )
        fmt.Printf( "      minimum focal length: %d/%d=%f\n",
                    v.numerator, v.denominator,
                    float32(v.numerator)/float32(v.denominator) )
        offset += 8
        v = ed.getUnsignedRational( offset )
        fmt.Printf( "      maximum focal length: %d/%d=%f\n",
                    v.numerator, v.denominator,
                    float32(v.numerator)/float32(v.denominator) )
        offset += 8
        v = ed.getUnsignedRational( offset )
        fmt.Printf( "      minimum F number: %d/%d=%f\n",
                    v.numerator, v.denominator,
                    float32(v.numerator)/float32(v.denominator) )
        offset += 8
        v = ed.getUnsignedRational( offset )
        fmt.Printf( "      maximum F number: %d/%d=%f\n",
                    v.numerator, v.denominator,
                    float32(v.numerator)/float32(v.denominator) )
    }
    return nil
}

func (ed *Desc) checkExifTag( ifd, tag, fType, fCount uint ) error {
    switch tag {
    case _ExposureTime:
        return ed.checkExifExposureTime( fType, fCount )
    case _FNumber:
        return ed.checkTiffUnsignedRational( "FNumber", fType, fCount, nil )
    case _ExposureProgram:
        return ed.checkExifExposureProgram( fType, fCount )

    case _ISOSpeedRatings:
        return ed.checkTiffUnsignedShorts( "ISOSpeedRatings", fType, fCount )
    case _ExifVersion:
        return ed.checkExifVersion( fType, fCount )

    case _DateTimeOriginal:
        return ed.checkTiffAscii( "DateTimeOriginal", fType, fCount )
    case _DateTimeDigitized:
        return ed.checkTiffAscii( "DateTimeDigitized", fType, fCount )

    case _OffsetTime:
        return ed.checkTiffAscii( "OffsetTime", fType, fCount )
    case _OffsetTimeOriginal:
        return ed.checkTiffAscii( "OffsetTimeOriginal", fType, fCount )
    case _OffsetTimeDigitized:
        return ed.checkTiffAscii( "OffsetTimeDigitized", fType, fCount )

    case _ComponentsConfiguration:
        return ed.checkExifComponentsConfiguration( fType, fCount )
    case _CompressedBitsPerPixel:
        return ed.checkTiffUnsignedRational( "CompressedBitsPerPixel", fType, fCount, nil )
    case _ShutterSpeedValue:
        return ed.checkTiffSignedRational( "ShutterSpeedValue", fType, fCount, nil )
    case _ApertureValue:
        return ed.checkTiffUnsignedRational( "ApertureValue", fType, fCount, nil )
    case _BrightnessValue:
        return ed.checkTiffSignedRational( "BrightnessValue", fType, fCount, nil )
    case _ExposureBiasValue:
        return ed.checkTiffSignedRational( "ExposureBiasValue", fType, fCount, nil )
    case _MaxApertureValue:
        return ed.checkTiffUnsignedRational( "MaxApertureValue", fType, fCount, nil )
    case _SubjectDistance:
        return ed.checkExifSubjectDistance( fType, fCount )
    case _MeteringMode:
        return ed.checkExifMeteringMode( fType, fCount )
    case _LightSource:
        return ed.checkExifLightSource( fType, fCount )
    case _Flash:
        return ed.checkExifFlash( fType, fCount )
    case _FocalLength:
        return ed.checkTiffUnsignedRational( "FocalLength", fType, fCount, nil )
    case _SubjectArea:
        return ed.checkExifSubjectArea( fType, fCount )

    case _MakerNote:
        return ed.checkExifMakerNote( fType, fCount )
    case _UserComment:
        return ed.checkExifUserComment( fType, fCount )
    case _SubsecTime:
        return ed.checkTiffAscii( "SubsecTime", fType, fCount )
    case _SubsecTimeOriginal:
        return ed.checkTiffAscii( "SubsecTimeOriginal", fType, fCount )
    case _SubsecTimeDigitized:
        return ed.checkTiffAscii( "SubsecTimeDigitized", fType, fCount )
    case _FlashpixVersion:
        return ed.checkFlashpixVersion( fType, fCount )

    case _ColorSpace:
        return ed.checkExifColorSpace( fType, fCount )
    case _PixelXDimension:
        return ed.checkExifDimension( "PixelXDimension", fType, fCount )
    case _PixelYDimension:
        return ed.checkExifDimension( "PixelYDimension", fType, fCount )

    case _SensingMethod:
        return ed.checkExifSensingMethod( fType, fCount )
    case _FileSource:
        return ed.checkExifFileSource( fType, fCount )
    case _SceneType:
        return ed.checkExifSceneType( fType, fCount )
    case _CFAPattern:
        return ed.checkExifCFAPattern( fType, fCount )
    case _CustomRendered:
        return ed.checkExifCustomRendered( fType, fCount )
    case _ExposureMode:
        return ed.checkExifExposureMode( fType, fCount )
    case _WhiteBalance:
        return ed.checkExifWhiteBalance( fType, fCount )
    case _DigitalZoomRatio:
        return ed.checkExifDigitalZoomRatio( fType, fCount )
    case _FocalLengthIn35mmFilm:
        return ed.checkTiffUnsignedShort( "FocalLengthIn35mmFilm", fType, fCount, nil )
    case _SceneCaptureType:
        return ed.checkExifSceneCaptureType( fType, fCount )
    case _GainControl:
        return ed.checkExifGainControl( fType, fCount )
    case _Contrast:
        return ed.checkExifContrast( fType, fCount )
    case _Saturation:
        return ed.checkExifSaturation( fType, fCount )
    case _Sharpness:
        return ed.checkExifSharpness( fType, fCount )
    case _SubjectDistanceRange:
        return ed.checkExifDistanceRange( fType, fCount )
    case _ImageUniqueID:
        return ed.checkTiffAscii( "ImageUniqueID ", fType, fCount )
    case _LensSpecification:
        return ed.checkExifLensSpecification( fType, fCount )
    case _LensMake:
        return ed.checkTiffAscii( "LensMake", fType, fCount )
    case _LensModel:
        return ed.checkTiffAscii( "LensModel", fType, fCount )
    case _Padding:
        return ed.checkPadding( fType, fCount )
    }
    return fmt.Errorf( "checkExifTag: unknown or unsupported tag (%#02x) @offset %#04x type %s count %d\n",
                       tag, ed.offset-8, getTiffTString( fType ), fCount )
}

const (                                     // _GPS IFD specific tags
    _GPSVersionID           = 0x00
    _GPSLatitudeRef         = 0x01
    _GPSLatitude            = 0x02
    _GPSLongitudeRef        = 0x03
    _GPSLongitude           = 0x04
    _GPSAltitudeRef         = 0x05
    _GPSAltitude            = 0x06
    _GPSTimeStamp           = 0x07
    _GPSSatellites          = 0x08
    _GPSStatus              = 0x09
    _GPSMeasureMode         = 0x0a
    _GPSDOP                 = 0x0b
    _GPSSpeedRef            = 0x0c
    _GPSSpeed               = 0x0d
    _GPSTrackRef            = 0x0e
    _GPSTrack               = 0x0f
    _GPSImgDirectionRef     = 0x10
    _GPSImgDirection        = 0x11
    _GPSMapDatum            = 0x12
    _GPSDestLatitudeRef     = 0x13
    _GPSDestLatitude        = 0x14
    _GPSDestLongitudeRef    = 0x15
    _GPSDestLongitude       = 0x16
    _GPSDestBearingRef      = 0x17
    _GPSDestBearing         = 0x18
    _GPSDestDistanceRef     = 0x19
    _GPSDestDistance        = 0x1a
    _GPSProcessingMethod    = 0x1b
    _GPSAreaInformation     = 0x1c
    _GPSDateStamp           = 0x1d
    _GPSDifferential        = 0x1e
)

func (ed *Desc) checkGPSVersionID( fType, fCount uint ) error {
    if fCount != 4 {
        return fmt.Errorf( "GPSVersionID: invalid count (%d)\n", fCount )
    }
    if fType != _UnsignedByte {
        return fmt.Errorf( "GPSVersionID: invalid type (%s)\n", getTiffTString( fType ) )
    }
    if ed.print {
        slc := ed.getBytes( ed.offset, fCount )  // 4 bytes fit in directory entry
        fmt.Printf("    GPSVersionID: %d.%d.%d.%d\n", slc[0], slc[1], slc[2], slc[3] )
    }
    return nil
}

func (ed *Desc) checkGpsTag( ifd, tag, fType, fCount uint ) error {
    switch tag {
    case _GPSVersionID:
        return ed.checkGPSVersionID( fType, fCount )
    }
    return fmt.Errorf( "checkGpsTag: unknown or unsupported tag (%#02x) @offset %#04x type %s count %d\n",
                       tag, ed.offset-8, getTiffTString( fType ), fCount )
}

const (                                     // _IOP IFD tags
    _InteroperabilityIndex      = 0x01
    _InteroperabilityVersion    = 0x02
)

func (ed *Desc) checkInteroperabilityVersion( fType, fCount uint ) error {
    if fType != _Undefined {
        return fmt.Errorf( "InteroperabilityVersion: invalid type (%s)\n", getTiffTString( fType ) )
    }
    // assume bytes
    if ed.print {
        bs := ed.getBytesFromIFD( fCount )
        fmt.Printf( "    InteroperabilityVersion: %#02x, %#02x, %#02x, %#02x\n",
                    bs[0], bs[1], bs[2], bs[3] )
    }
    return nil
}

func (ed *Desc)checkIopTag( ifd, tag, fType, fCount uint ) error {
    switch tag {
    case _InteroperabilityIndex:
        return ed.checkTiffAscii( "Interoperability", fType, fCount )
    case _InteroperabilityVersion:
        return ed.checkInteroperabilityVersion( fType, fCount )
    default:
        if ed.print {
            fmt.Printf( "    unknown or unsupported tag (%#02x) @offset %#04x type %s count %d\n",
                        tag, ed.offset-8, getTiffTString( fType), fCount )
        }
    }
    return nil
}

func (ed *Desc) checkIFD( Ifd uint, tag1, tag2 int ) ( offset0, offset1,
                                                    offset2 uint, err error ) {

    offset1 = 0
    offset2 = 0
    err = nil

    var checkTags func( Ifd, tag, fType, fCount uint ) error
    switch Ifd {
    case _PRIMARY, _THUMBNAIL:  checkTags = ed.checkTiffTag
    case _EXIF:                 checkTags = ed.checkExifTag
    case _GPS:                  checkTags = ed.checkGpsTag
    case _IOP:                  checkTags = ed.checkIopTag
    case _APPLE:                checkTags = ed.checkApple
    }
    /*
        Image File Directory starts with the number of following directory entries (2 bytes)
        followed by that number of entries (12 bytes) and one extra offset to the next IFD
        (4 bytes)
    */
    nIfdEntries := ed.getUnsignedShort( ed.offset )
    if ed.print {
        fmt.Printf( "  IFD #%d %s @%#04x #entries %d\n", Ifd,
                    IfdNames[Ifd], ed.offset, nIfdEntries )
//        fmt.Printf( "  %s:\n", IfdNames[Ifd] )
    }

    ed.offset += 2
    for i := uint(0); i < nIfdEntries; i++ {
        fTag := ed.getUnsignedShort( ed.offset )
        fType := ed.getUnsignedShort( ed.offset + 2 )
        fCount := ed.getUnsignedLong( ed.offset + 4 )
        ed.offset += 8

        if tag1 != -1 && fTag == uint(tag1) {
            offset1 = ed.getUnsignedLong( ed.offset )
        } else if tag2 != -1 && fTag == uint(tag2) {
            offset2 = ed.getUnsignedLong( ed.offset )
        } else {
            err := checkTags( Ifd, fTag, fType, fCount )
            if err != nil {
                return 0, 0, 0, fmt.Errorf( "checkIFD: invalid field: %v\n", err )
            }
        }
        ed.offset += 4
    }
    offset0 = ed.getUnsignedLong( ed.offset )
    return
}

func getEndianess( data []byte ) ( size uint, lEndian bool, err error) {
    lEndian = false
    size = 2
    err = nil
    if bytes.Equal( data[:2], []byte( "II" ) ) {
        lEndian = true
    } else if ! bytes.Equal( data[:2], []byte( "MM" ) ) {
        size = 0
        err = fmt.Errorf( "exif: invalid TIFF header (unknown byte ordering: %v)\n", data[:2] )
    }
    return
}

func Parse( data []byte, start, dLen uint, ec *Control ) (*Desc, error) {
    if ! bytes.Equal( data[start:start+6], []byte( "Exif\x00\x00" ) ) {
        return nil, fmt.Errorf( "exif: invalid signature (%s)\n", string(data[0:6]) )
    }
    // Exif\0\0 is followed immediately by TIFF header
    ed := new( Desc )
    ed.data = data[start+6:start+dLen-6]    // origin is always TIFF header (0)
    ed.Control = *ec

    if ed.print {
        fmt.Printf( "APP1 (EXIF)\n" )
    }

    // TIFF header starts with 2 bytes indicating the byte ordering (little or big endian)
    var err error
    var offset uint
    offset, ed.lEndian, err = getEndianess( ed.data )
    if err != nil {
        return nil, err
    }

    validTiff := ed.getUnsignedShort( offset )
    if validTiff != 0x2a {
        return nil, fmt.Errorf( "exif: invalid TIFF header (invalid identifier: %#02x)\n", validTiff )
    }

    // first IFD is the primary image file directory 0
    IFDOffset := ed.getUnsignedLong( 4 )
    ed.offset = IFDOffset
    IFDOffset, exifIFDOffset, gpsIFDOffset, err :=
        ed.checkIFD( _PRIMARY, _ExifIFD, _GpsIFD )
    if err != nil { return nil, err }
//    fmt.Printf( "IFDOffset %#04x, exifIFDOffset %#04x, gpsIFDOffset %#04x\n",
//                IFDOffset, exifIFDOffset, gpsIFDOffset )

    if IFDOffset != 0 {
        ed.offset = IFDOffset
        _, thbOffset, thbLength, err := ed.checkIFD( _THUMBNAIL,
                                                     _JPEGInterchangeFormat,
                                                     _JPEGInterchangeFormatLength )
        if err != nil { return nil, err }

        // store tumbnail image information
        ed.tOffset = thbOffset
        ed.tLen = thbLength
    }

    var ioIFDopOffset uint
    if exifIFDOffset != 0 {
        ed.offset = exifIFDOffset
        _, ioIFDopOffset, _, err = ed.checkIFD( _EXIF, _InteroperabilityIFD, -1 )
        if err != nil { return nil, err }
    }

    if ioIFDopOffset != 0 {
        ed.offset = ioIFDopOffset
        _, _, _, err = ed.checkIFD( _IOP, -1, -1 )
        if err != nil { return nil, err }
    }

    if gpsIFDOffset != 0 {
        ed.offset = gpsIFDOffset
        _, _, _, err = ed.checkIFD( _GPS, -1, -1 )
        if err != nil { return nil, err }
    }
    return ed, nil
}

func (ed *Desc)getThumbnail() (uint, uint, Compression) {
    return ed.tOffset, ed.tLen, ed.tType
}

