
package exif

import (
    "fmt"
    "bytes"
    "strings"
    "encoding/binary"
    "os"        // temporarily
)

/*
    check functions are always called with a valid entry (fTag, fType, fCount
    and sOffset pointing at the value|offset of an IFD entry. They check for a
    valid type and count (if appropriate), and if no error was found, print
    name and value (if requested) and store the ifd value in the current ifd.
*/ 

func (ifd *ifdd) checkUndefinedAsByte( name string, f func( v byte) ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "%s: invalid type (%s)\n", name, getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 1 {
        return fmt.Errorf( "%s: incorrect count (%d)\n", name, ifd.fCount )
    }
    value := ifd.desc.getByte( ifd.sOffset )
    if ifd.desc.Print {
        if f == nil {
            fmt.Printf( "    %s: %d\n", name, value )
        } else {
            fmt.Printf( "    %s: ", name )
            f( value )
        }
    }
    ifd.storeValue( ifd.newUnsignedByteValue( []byte{ value } ) )
    return nil
}

func (ifd *ifdd) checkTiffAscii( name string ) error {
    if ifd.fType != _ASCIIString {
        return fmt.Errorf( "%s: invalid type (%s)\n",
                           name, getTiffTString( ifd.fType ) )
    }
    offset := ifd.sOffset
    if ifd.fCount > 4 {
        offset = ifd.desc.getUnsignedLong( ifd.sOffset )
    }
    text := ifd.desc.getASCIIString( offset, ifd.fCount )
    if ifd.desc.Print {
        fmt.Printf( "    %s: %s\n", name, text )
    }
    ifd.storeValue( ifd.newAsciiStringValue( text ) )
    return nil
}

func (ifd *ifdd) checkTiffUnsignedShort( name string, f func( v uint16) ) error {
    if ifd.fType != _UnsignedShort {
        return fmt.Errorf( "%s: invalid type (%s)\n", name, getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 1 {
        return fmt.Errorf( "%s: invalid count (%d)\n", name, ifd.fCount )
    }
    value := ifd.desc.getUnsignedShort( ifd.sOffset )
    if ifd.desc.Print {
        if f == nil {
            fmt.Printf( "    %s: %d\n", name, value )
        } else {
            fmt.Printf( "    %s: ", name )
            f( value )
        }        
    }
    ifd.storeValue( ifd.newUnsignedShortValue( []uint16{ value } ) )
    return nil
}

func (ifd *ifdd) checkTiffUnsignedShorts( name string ) error {
    if ifd.fType != _UnsignedShort {
        return fmt.Errorf( "%s: invalid type (%s)\n",
                            name, getTiffTString( ifd.fType ) )
    }
    values := ifd.getUnsignedShorts( )
    if ifd.desc.Print {
        fmt.Printf( "    %s:", name )
        for _, v := range values {
            fmt.Printf( " %d", v )
        }
        fmt.Printf( "\n");
    }
    ifd.storeValue( ifd.newUnsignedShortValue( values ) )
    return nil
}

func (ifd *ifdd) checkTiffUnsignedLong( name string, f func( v uint32) ) error {
    if ifd.fType != _UnsignedLong {
        return fmt.Errorf( "%s: invalid type (%s)\n", name, getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 1 {
        return fmt.Errorf( "%s: invalid count (%d)\n", name, ifd.fCount )
    }
    value := ifd.desc.getUnsignedLong( ifd.sOffset )
    if ifd.desc.Print {
        if f == nil {
            fmt.Printf( "    %s: %d\n", name, value )
        } else {
            fmt.Printf( "    %s: ", name )
            f( value )
        }
    }
    ifd.storeValue( ifd.newUnsignedLongValue( []uint32{ value } ) )
    return nil
}

func (ifd *ifdd) checkTiffSignedLong( name string, f func( v int32) ) error {
    if ifd.fType != _SignedLong {
        return fmt.Errorf( "%s: invalid type (%s)\n", name, getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 1 {
        return fmt.Errorf( "%s: invalid count (%d)\n", name, ifd.fCount )
    }
    value := ifd.desc.getSignedLong( ifd.sOffset )
    if ifd.desc.Print {
        if f == nil {
            fmt.Printf( "    %s: %d\n", name, value )
        } else {
            fmt.Printf( "    %s: ", name )
            f( value )
        }        
    }
    ifd.storeValue( ifd.newSignedLongValue( []int32{ value } ) )
    return nil
}

func (ifd *ifdd) checkTiffUnsignedRational( name string,
                                           f func( v unsignedRational ) ) error {
    if ifd.fType != _UnsignedRational {
        return fmt.Errorf( "%s: invalid type (%s)\n",
                            name, getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 1 {
        return fmt.Errorf( "%s: invalid count (%d)\n", name, ifd.fCount )
    }
    // a rational never fits directly in valOffset (requires more than 4 bytes)
    offset := ifd.desc.getUnsignedLong( ifd.sOffset )
    v := ifd.desc.getUnsignedRational( offset )
    if ifd.desc.Print {
        if f == nil {
            fmt.Printf( "    %s: %d/%d=%f\n", name, v.Numerator, v.Denominator,
                        float32(v.Numerator)/float32(v.Denominator) )
        } else {
            fmt.Printf( "    %s: ", name )
            f( v )
        }
    }
    ifd.storeValue( ifd.newUnsignedRationalValue( []unsignedRational{ v } ) )
    return nil
}

func (ifd *ifdd) checkTiffSignedRational( name string,
                                         f func( v signedRational ) ) error {
    if ifd.fType != _SignedRational {
        return fmt.Errorf( "%s: invalid type (%s)\n",
                            name, getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 1 {
        return fmt.Errorf( "%s: invalid count (%d)\n", name, ifd.fCount )
    }
    // a rational never fits directly in valOffset (requires more than 4 bytes)
    offset := ifd.desc.getUnsignedLong( ifd.sOffset )
    v := ifd.desc.getSignedRational( offset )
    if ifd.desc.Print {
        if f == nil {
            fmt.Printf( "    %s: %d/%d=%f\n", name, v.Numerator, v.Denominator,
                        float32(v.Numerator)/float32(v.Denominator) )
        } else {
            fmt.Printf( "    %s: ", name )
            f( v )
        }
    }
    ifd.storeValue( ifd.newSignedRationalValue( []signedRational{ v } ) )
    return nil
}

func (ifd *ifdd) checkTiffUnsignedRationals( name string, count uint32 ) error {
    if ifd.fType != _UnsignedRational {
        return fmt.Errorf( "%s: invalid type (%s)\n",
                            name, getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != count {
        return fmt.Errorf( "%s: invalid count (%d)\n", name, ifd.fCount )
    }
    rOffset := ifd.desc.getUnsignedLong( ifd.sOffset )
    values := ifd.desc.getUnsignedRationals( rOffset, ifd.fCount )
    fmt.Printf( "    %s:", name )
    if ifd.desc.Print {
        for _, v := range values {
            fmt.Printf( " %d/%d", v.Numerator, v.Denominator,
                        float32(v.Numerator)/float32(v.Denominator) )
        }
        fmt.Printf( "\n");
    }
    ifd.storeValue( ifd.newUnsignedRationalValue( values ) )
    return nil
}

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

func (ifd *ifdd) checkTiffCompression( ) error {
/*
    Exif2-2: optional in Primary IFD and in thumbnail IFD
When a primary image is JPEG compressed, this designation is not necessary and is omitted.
When thumbnails use JPEG compression, this tag value is set to 6.
*/
    fmtCompression := func( v uint16 ) {
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
        if ifd.id == _PRIMARY {
            if v != 6 {
                fmt.Printf("    Warning: non-JPEG compression specified in a JPEG file\n" )
            } else {
                fmt.Printf("    Warning: Exif2-2 specifies that in case of JPEG picture compression be omited\n")
            }
        } else {    // _THUMBNAIL
            ifd.desc.tType = cType    // remember thumnail compression type
        }
    }
    return ifd.checkTiffUnsignedShort( "Compression", fmtCompression ) 
}

func (ifd *ifdd) checkTiffOrientation( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "Orientation", fmtv )
}

func (ifd *ifdd) checkTiffResolutionUnit( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "ResolutionUnit", fmtv )
}

func (ifd *ifdd) checkTiffYCbCrPositioning( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "YCbCrPositioning", fmtv )
}

func (ifd *ifdd) checkJPEGInterchangeFormat( ) error {
    if ifd.fType != _UnsignedLong {
        return fmt.Errorf( "checkJPEGInterchangeFormat: invalid type (%s)\n",
                            getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 1 {
        return fmt.Errorf( "checkJPEGInterchangeFormat: invalid count (%d)\n",
                            ifd.fCount )
    }
    ifd.desc.tOffset = ifd.desc.getUnsignedLong( ifd.sOffset )
    ifd.storeValue( ifd.newUnsignedLongValue( []uint32{ ifd.desc.tOffset } ) )
    return nil
}

func (ifd *ifdd) checkJPEGInterchangeFormatLength( ) error {
    if ifd.fType != _UnsignedLong {
        return fmt.Errorf( "checkJPEGInterchangeFormatLength: invalid type (%s)\n",
                           getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 1 {
        return fmt.Errorf( "checkJPEGInterchangeFormatLength: invalid count (%d)\n",
                           ifd.fCount )
    }
    ifd.desc.tLen = ifd.desc.getUnsignedLong( ifd.sOffset )
    ifd.storeValue( ifd.newUnsignedLongValue( []uint32{ ifd.desc.tLen } ) )
    return nil
}

func (ifd *ifdd) checkEmbeddedIfd( name string, id ifdId,
                                   checkTags func( ifd *ifdd) error ) error {
    if ifd.fType != _UnsignedLong {
        return fmt.Errorf( "check %s: invalid type (%s)\n",
                           name, getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 1 {
        return fmt.Errorf( "check %d: invalid count (%d)\n",
                           name, ifd.fCount )
    }
    // recusively process the embedded IFD here
    offset := ifd.desc.getUnsignedLong( ifd.sOffset )
    if ifd.desc.Print {
        fmt.Printf( "  %s @%#04x\n", name, offset )
    }

    fmt.Printf("      ---------------------------------- %s ----------------------------------\n", name)
    _, eIfd, err := ifd.desc.checkIFD( id, offset, checkTags )
    fmt.Printf("      ----------------------------------------------------------------------------\n")
    if err != nil {
        return err
    }
    ifd.storeValue( ifd.newIfdValue( eIfd ) )
    return nil
}

func (ifd *ifdd) checkPadding( ) error {
    if ifd.desc.Print {
        fmt.Printf("    Padding: %d bytes - ignored\n", ifd.fCount )
    }
    return nil
}

func checkTiffTag( ifd *ifdd ) error {
    fmt.Printf( "checkTiffTag: tag (%#04x) @offset %#04x type %s count %d\n",
                 ifd.fTag, ifd.sOffset-8, getTiffTString( ifd.fType ), ifd.fCount )
    switch ifd.fTag {
    case _Compression:
        return ifd.checkTiffCompression( )
    case _ImageDescription:
        return ifd.checkTiffAscii( "ImageDescription" )
    case _Make:
        return ifd.checkTiffAscii( "Make" )
    case _Model:
        return ifd.checkTiffAscii( "Model" )
    case _Orientation:
        return ifd.checkTiffOrientation( )
    case _XResolution:
        return ifd.checkTiffUnsignedRational( "XResolution", nil )
    case _YResolution:
        return ifd.checkTiffUnsignedRational( "YResolution", nil )
    case _ResolutionUnit:
        return ifd.checkTiffResolutionUnit( )
    case _Software:
        return ifd.checkTiffAscii( "Software" )
    case _DateTime:
        return ifd.checkTiffAscii( "Date" )
    case _HostComputer:
        return ifd.checkTiffAscii( "HostComputer" )
    case _YCbCrPositioning:
        return ifd.checkTiffYCbCrPositioning( )

    case _JPEGInterchangeFormat:
        return ifd.checkJPEGInterchangeFormat( )

    case _JPEGInterchangeFormatLength:
        return ifd.checkJPEGInterchangeFormatLength( )

    case _Copyright:
        return ifd.checkTiffAscii( "Copyright" )

    case _ExifIFD:
        return ifd.checkEmbeddedIfd( "Exif IFD", _EXIF, checkExifTag )
    case  _GpsIFD:
        return ifd.checkEmbeddedIfd( "GPS IFD", _GPS, checkGpsTag )

    case _Padding:
        return ifd.checkPadding( )
    }
    return fmt.Errorf( "checkTiffTag: unknown or unsupported tag (%#02x) @offset %#04x type %s count %d\n",
                       ifd.fTag, ifd.sOffset-8, getTiffTString( ifd.fType ), ifd.fCount )
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

func (ifd *ifdd) checkExifVersion( ) error {
  // special case: tiff type is undefined, but it is actually ASCII
    if ifd.fType != _Undefined {
        return fmt.Errorf( "ExifVersion: invalid byte type (%s)\n", getTiffTString( ifd.fType ) )
    }
    text := ifd.getAsciiString( )
    if ifd.desc.Print {
        fmt.Printf( "    ExifVersion: %s\n", text )
    }
    ifd.storeValue( ifd.newAsciiStringValue( text ) )
    return nil
}

func (ifd *ifdd) checkExifExposureTime( ) error {
    fmtv := func( v unsignedRational ) {
        fmt.Printf( "%f seconds\n", float32(v.Numerator)/float32(v.Denominator) )
    }
    return ifd.checkTiffUnsignedRational( "ExposureTime", fmtv )
}

func (ifd *ifdd) checkExifExposureProgram( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "ExposureProgram", fmtv )
}

func (ifd *ifdd) checkExifComponentsConfiguration( ) error {
    if ifd.fType != _Undefined {  // special case: tiff type is undefined, but it is actually bytes
        return fmt.Errorf( "ComponentsConfiguration: invalid type (%s)\n", getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 4 {
        return fmt.Errorf( "ComponentsConfiguration: invalid byte count (%d)\n", ifd.fCount )
    }
    bSlice := ifd.getUnsignedBytes(  )
    if ifd.desc.Print {
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
    ifd.storeValue( ifd.newUnsignedByteValue( bSlice ) )
    return nil
}

func (ifd *ifdd) checkExifSubjectDistance( ) error {
    fmtv := func( v unsignedRational ) {
        if v.Numerator == 0 {
            fmt.Printf( "Unknown\n" )
        } else if v.Numerator == 0xffffffff {
            fmt.Printf( "Infinity\n" )
        } else {
            fmt.Printf( "%f meters\n", float32(v.Numerator)/float32(v.Denominator) )
        }
    }
    return ifd.checkTiffUnsignedRational( "SubjectDistance", fmtv )
}

func (ifd *ifdd) checkExifMeteringMode( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "MeteringMode", fmtv )
}

func (ifd *ifdd) checkExifLightSource( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "LightSource", fmtv )
}

func (ifd *ifdd) checkExifFlash( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "Flash", fmtv )
}

func (ifd *ifdd) checkExifSubjectArea( ) error {
    if ifd.fCount < 2 && ifd.fCount > 4 {
        return fmt.Errorf( "Subject Area: invalid count (%d)\n", ifd.fCount )
    }
    loc := ifd.getUnsignedShorts( )
    if ifd.desc.Print {
        switch ifd.fCount {
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
    ifd.storeValue( ifd.newUnsignedShortValue( loc ) )
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

func (ifd *ifdd) checkExifMakerNote( ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "MakerNote: invalid type (%s)\n", getTiffTString( ifd.fType ) )
    }
    if ifd.fCount < 4 {
        if ifd.desc.Print {
            dumpData( "    MakerNote", "      ", ifd.desc.data[ifd.sOffset:ifd.sOffset+ifd.fCount] )
        }
// FIXME: add generic makerNote value for unknown makers
//        ifd.storeEntry( interface{}( ifd.desc.data[ifd.sOffset:ifd.sOffset+ifd.fCount] ) )
        return nil
    } else {
        offset := ifd.desc.getUnsignedLong( ifd.sOffset )
        p := ifd.tryAppleMakerNote( offset )
        if p != nil {
            return p( offset )
        }
    }
    return fmt.Errorf( "checkExifMakerNote: unknown maker\n")
}

func (ifd *ifdd) checkExifUserComment( ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "UserComment: invalid type (%s)\n", getTiffTString( ifd.fType ) )
    }
    if ifd.fCount < 8 {
        return fmt.Errorf( "UserComment: invalid count (%s)\n", ifd.fCount )
    }
    //  first 8 Bytes are the encoding
    offset := ifd.desc.getUnsignedLong( ifd.sOffset )
    encoding := ifd.desc.getUnsignedBytes( offset, 8 )
    switch encoding[0] {
    case 0x41:  // ASCII?
        if bytes.Equal( encoding, []byte{ 'A', 'S', 'C', 'I', 'I', 0, 0, 0 } ) {
            if ifd.desc.Print {
                fmt.Printf( "    UserComment: ITU-T T.50 IA5 (ASCII) [%s]\n", 
                            string(ifd.desc.getUnsignedBytes( offset+8, ifd.fCount-8 )) )
            }
        }
    case 0x4a: // JIS?
        if bytes.Equal( encoding, []byte{ 'J', 'I', 'S', 0, 0, 0, 0, 0 } ) {
            if ifd.desc.Print {
                fmt.Printf( "    UserComment: JIS X208-1990 (JIS):" )
                dumpData( "    UserComment", "      ", ifd.desc.data[offset+8:offset+ifd.fCount] )
            }
        }
    case 0x55:  // UNICODE?
        if bytes.Equal( encoding, []byte{ 'U', 'N', 'I', 'C', 'O', 'D', 'E', 0 } ) {
            if ifd.desc.Print {
                fmt.Printf( "    UserComment: Unicode Standard:" )
                dumpData( "    UserComment", "      ", ifd.desc.data[offset+8:offset+ifd.fCount] )
            }
        }
    case 0x00:  // Undefined
        if bytes.Equal( encoding, []byte{ 0, 0, 0, 0, 0, 0, 0, 0 } ) {
            if ifd.desc.Print {
                fmt.Printf( "    UserComment: Undefined encoding:" )
                dumpData( "    UserComment", "      ", ifd.desc.data[offset+8:offset+ifd.fCount] )
            }
        }
    default:
        return fmt.Errorf( "UserComment: invalid encoding\n" )
    }
    ifd.storeValue( ifd.newUnsignedByteValue( ifd.desc.data[ifd.sOffset:ifd.sOffset+ifd.fCount] ) )
    return nil
}

func (ifd *ifdd) checkFlashpixVersion( ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "FlashpixVersion: invalid type (%s)\n", getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 4 {
        return fmt.Errorf( "FlashpixVersion: incorrect count (%d)\n", ifd.fCount )
    }
    text := ifd.getAsciiString( )
    if ifd.desc.Print {
        fmt.Printf( "    FlashpixVersion: %s\n", text )
    }
    ifd.storeValue( ifd.newAsciiStringValue( text ) )
    return nil
}

func (ifd *ifdd) checkExifColorSpace( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "ColorSpace", fmtv )
}

func (ifd *ifdd) checkExifDimension( name string ) error {
    if ifd.fType == _UnsignedShort {
        return ifd.checkTiffUnsignedShort( name, nil )
    } else if ifd.fType == _UnsignedLong {
        return ifd.checkTiffUnsignedLong( name, nil )
    }
    return fmt.Errorf( "%s: invalid type (%s)\n", name, getTiffTString( ifd.fType ) )
}

func (ifd *ifdd) checkExifSensingMethod( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "SensingMethod", fmtv )
}

func (ifd *ifdd) checkExifFileSource( ) error {
    fmtv := func( v byte ) {       // undfined but expect byte
        if v != 3 {
            fmt.Printf( "Illegal file source (%d)\n", v )
            return
        }
        fmt.Printf( "Digital Still Camera (DSC)\n" )
    }
    return ifd.checkUndefinedAsByte( "FileSource", fmtv )
}

func (ifd *ifdd) checkExifSceneType( ) error {
    fmtv := func( v byte ) {       // undefined but expect byte
        var stString string
        switch v {
        case 1 : stString = "Directly photographed"
        default:
            fmt.Printf( "Illegal scene type (%d)\n", v )
            return
        }
        fmt.Printf( "%s\n", stString )
    }
    return ifd.checkUndefinedAsByte( "SceneType", fmtv )
}

func (ifd *ifdd) checkExifCFAPattern( ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "CFAPattern: invalid type (%s)\n", getTiffTString( ifd.fType ) )
    }
    // structure describing the color filter array (CFA)
    // 2 short words: horizontal repeat pixel unit (h), vertical repeat pixel unit (v)
    // followed by h*v bytes, each byte value indicating a color:
    // 0 RED, 1 GREEN, 2 BLUE, 3 CYAN, 4 MAGENTA, 5 YELLOW, 6 WHITE
    // Since the structure cannot fit in 4 bytes, its location is indicated by an offset
    offset := ifd.desc.getUnsignedLong( ifd.sOffset )
    h := ifd.desc.getUnsignedShort( offset )
    v := ifd.desc.getUnsignedShort( offset + 2 )
    // however, it seems that older microsoft tools do not use the proper endianess,
    // so check here if the values are consistent with the total count:
    var swap bool
    if uint32(h) * uint32(v) != ifd.fCount - 4 { // if not try changing endianess
        h1 := ((h & 0xff) << 8) + (h >> 8)
        v1 := ((v & 0xff) << 8) + (v >> 8)
        if ( uint32(h1) * uint32(v1) != ifd.fCount - 4 ) {
            return fmt.Errorf( "CFAPattern: invalid repeat patterns(%d,%d)\n", h, v )
        }
        h = h1
        v = v1
        swap = true
    }
    if ifd.desc.Print {
        fmt.Printf( "    CFAPattern:" )
    }
    offset += 4
    c := ifd.desc.getUnsignedBytes( offset, uint32(h) * uint32(v) )

    for i := uint16(0); i < v; i++ {
        fmt.Printf("\n      Row %d:", i )
        for j := uint16(0); j < h; j++ {
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
                if ifd.desc.Print {
                    fmt.Printf("\n")
                }
                return fmt.Errorf( "CFAPattern: invalid color (%d)\n", c[(i*h)+j] )
            }
            if ifd.desc.Print {
                fmt.Printf( " %s", s )
            }
        }
    }
    if ifd.desc.Print {
        fmt.Printf( "\n" )
    }
    if swap {
        fmt.Printf("      Warning: CFAPattern: incorrect endianess\n")
    }
    ifd.storeValue( ifd.newUnsignedByteValue( ifd.desc.data[ifd.sOffset:ifd.sOffset+ifd.fCount] ) )
    return nil
}

func (ifd *ifdd) checkExifCustomRendered( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "CustomRendered", fmtv )
}

func (ifd *ifdd) checkExifExposureMode( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "ExposureMode", fmtv )
}

func (ifd *ifdd) checkExifWhiteBalance( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "WhiteBalance", fmtv )
}

func (ifd *ifdd) checkExifDigitalZoomRatio( ) error {
    fmv := func( v unsignedRational ) {
        if v.Numerator == 0 {
            fmt.Printf( "not used\n" )
        } else if v.Denominator == 0 {
            fmt.Printf( "invalid ratio Denominator (0)\n" )
        } else {
            fmt.Printf( "%f\n", float32(v.Numerator)/float32(v.Denominator) )
        }
    }
    return ifd.checkTiffUnsignedRational( "DigitalZoomRatio", fmv )
}

func (ifd *ifdd) checkExifSceneCaptureType( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "SceneCaptureType", fmtv )
}

func (ifd *ifdd) checkExifGainControl( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "GainControl", fmtv )
}

func (ifd *ifdd) checkExifContrast( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "Contrast", fmtv )
}

func (ifd *ifdd) checkExifSaturation( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "Saturation", fmtv )
}

func (ifd *ifdd) checkExifSharpness( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "Sharpness", fmtv )
}

func (ifd *ifdd) checkExifDistanceRange( ) error {
    fmtv := func( v uint16 ) {
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
    return ifd.checkTiffUnsignedShort( "DistanceRange", fmtv )
}

func (ifd *ifdd) checkExifLensSpecification( ) error {
// LensSpecification is an array of ordered unsignedRational values:
//  minimum focal length
//  maximum focal length
//  minimum F number in minimum focal length
//  maximum F number in maximum focal length
//  which are specification information for the lens that was used in photography.
//  When the minimum F number is unknown, the notation is 0/0.
    if ifd.fCount != 4 {
        return fmt.Errorf( "LensSpecification: invalid count (%d)\n", ifd.fCount )
    }
    if ifd.fType != _UnsignedRational {
        return fmt.Errorf( "LensSpecification: invalid type (%s)\n", getTiffTString( ifd.fType ) )
    }

    offset := ifd.desc.getUnsignedLong( ifd.sOffset )
    specs :=  ifd.desc.getUnsignedRationals( offset, ifd.fCount )
    if ifd.desc.Print {
        fmt.Printf( "    LensSpecification:\n" )
        fmt.Printf( "      minimum focal length: %d/%d=%f\n",
                    specs[0].Numerator, specs[0].Denominator,
                    float32(specs[0].Numerator)/float32(specs[0].Denominator) )
        fmt.Printf( "      maximum focal length: %d/%d=%f\n",
                    specs[1].Numerator, specs[1].Denominator,
                    float32(specs[1].Numerator)/float32(specs[1].Denominator) )
        fmt.Printf( "      minimum F number: %d/%d=%f\n",
                    specs[2].Numerator, specs[2].Denominator,
                    float32(specs[2].Numerator)/float32(specs[2].Denominator) )
        fmt.Printf( "      maximum F number: %d/%d=%f\n",
                    specs[3].Numerator, specs[3].Denominator,
                    float32(specs[3].Numerator)/float32(specs[3].Denominator) )
    }
    ifd.storeValue( ifd.newUnsignedRationalValue( specs ) )
    return nil
}

func checkExifTag( ifd *ifdd ) error {
    fmt.Printf( "checkExifTag: tag (%#04x) @offset %#04x type %s count %d\n",
                 ifd.fTag, ifd.sOffset-8, getTiffTString( ifd.fType ), ifd.fCount )
    switch ifd.fTag {
    case _ExposureTime:
        return ifd.checkExifExposureTime( )
    case _FNumber:
        return ifd.checkTiffUnsignedRational( "FNumber", nil )
    case _ExposureProgram:
        return ifd.checkExifExposureProgram( )

    case _ISOSpeedRatings:
        return ifd.checkTiffUnsignedShorts( "ISOSpeedRatings" )
    case _ExifVersion:
        return ifd.checkExifVersion( )

    case _DateTimeOriginal:
        return ifd.checkTiffAscii( "DateTimeOriginal" )
    case _DateTimeDigitized:
        return ifd.checkTiffAscii( "DateTimeDigitized" )

    case _OffsetTime:
        return ifd.checkTiffAscii( "OffsetTime" )
    case _OffsetTimeOriginal:
        return ifd.checkTiffAscii( "OffsetTimeOriginal" )
    case _OffsetTimeDigitized:
        return ifd.checkTiffAscii( "OffsetTimeDigitized" )

    case _ComponentsConfiguration:
        return ifd.checkExifComponentsConfiguration( )
    case _CompressedBitsPerPixel:
        return ifd.checkTiffUnsignedRational( "CompressedBitsPerPixel", nil )
    case _ShutterSpeedValue:
        return ifd.checkTiffSignedRational( "ShutterSpeedValue", nil )
    case _ApertureValue:
        return ifd.checkTiffUnsignedRational( "ApertureValue", nil )
    case _BrightnessValue:
        return ifd.checkTiffSignedRational( "BrightnessValue", nil )
    case _ExposureBiasValue:
        return ifd.checkTiffSignedRational( "ExposureBiasValue", nil )
    case _MaxApertureValue:
        return ifd.checkTiffUnsignedRational( "MaxApertureValue", nil )
    case _SubjectDistance:
        return ifd.checkExifSubjectDistance( )
    case _MeteringMode:
        return ifd.checkExifMeteringMode( )
    case _LightSource:
        return ifd.checkExifLightSource( )
    case _Flash:
        return ifd.checkExifFlash( )
    case _FocalLength:
        return ifd.checkTiffUnsignedRational( "FocalLength", nil )
    case _SubjectArea:
        return ifd.checkExifSubjectArea( )

    case _MakerNote:
        return ifd.checkExifMakerNote( )
    case _UserComment:
        return ifd.checkExifUserComment( )
    case _SubsecTime:
        return ifd.checkTiffAscii( "SubsecTime" )
    case _SubsecTimeOriginal:
        return ifd.checkTiffAscii( "SubsecTimeOriginal" )
    case _SubsecTimeDigitized:
        return ifd.checkTiffAscii( "SubsecTimeDigitized" )
    case _FlashpixVersion:
        return ifd.checkFlashpixVersion( )

    case _ColorSpace:
        return ifd.checkExifColorSpace( )
    case _PixelXDimension:
        return ifd.checkExifDimension( "PixelXDimension" )
    case _PixelYDimension:
        return ifd.checkExifDimension( "PixelYDimension" )

    case _SensingMethod:
        return ifd.checkExifSensingMethod( )
    case _FileSource:
        return ifd.checkExifFileSource( )
    case _SceneType:
        return ifd.checkExifSceneType( )
    case _CFAPattern:
        return ifd.checkExifCFAPattern( )
    case _CustomRendered:
        return ifd.checkExifCustomRendered( )
    case _ExposureMode:
        return ifd.checkExifExposureMode( )
    case _WhiteBalance:
        return ifd.checkExifWhiteBalance( )
    case _DigitalZoomRatio:
        return ifd.checkExifDigitalZoomRatio( )
    case _FocalLengthIn35mmFilm:
        return ifd.checkTiffUnsignedShort( "FocalLengthIn35mmFilm", nil )
    case _SceneCaptureType:
        return ifd.checkExifSceneCaptureType( )
    case _GainControl:
        return ifd.checkExifGainControl( )
    case _Contrast:
        return ifd.checkExifContrast( )
    case _Saturation:
        return ifd.checkExifSaturation( )
    case _Sharpness:
        return ifd.checkExifSharpness( )
    case _SubjectDistanceRange:
        return ifd.checkExifDistanceRange( )
    case _ImageUniqueID:
        return ifd.checkTiffAscii( "ImageUniqueID " )
    case _LensSpecification:
        return ifd.checkExifLensSpecification( )
    case _LensMake:
        return ifd.checkTiffAscii( "LensMake" )
    case _LensModel:
        return ifd.checkTiffAscii( "LensModel" )

    case _InteroperabilityIFD:
        return ifd.checkEmbeddedIfd( "IOP IFD", _IOP, checkIopTag )

    case _Padding:
        return ifd.checkPadding( )
    }
    return fmt.Errorf( "checkExifTag: unknown or unsupported tag (%#02x) @offset %#04x type %s count %d\n",
                       ifd.fTag, ifd.sOffset-8, getTiffTString( ifd.fType ), ifd.fCount )
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

func (ifd *ifdd) checkGPSVersionID( ) error {
    if ifd.fCount != 4 {
        return fmt.Errorf( "GPSVersionID: invalid count (%d)\n", ifd.fCount )
    }
    if ifd.fType != _UnsignedByte {
        return fmt.Errorf( "GPSVersionID: invalid type (%s)\n", getTiffTString( ifd.fType ) )
    }
    slc := ifd.desc.getUnsignedBytes( ifd.sOffset, ifd.fCount )  // 4 bytes fit in directory entry
    if ifd.desc.Print {
        fmt.Printf("    GPSVersionID: %d.%d.%d.%d\n", slc[0], slc[1], slc[2], slc[3] )
    }
    ifd.storeValue( ifd.newUnsignedByteValue( slc ) )
    return nil
}

func checkGpsTag( ifd *ifdd ) error {
    switch ifd.fTag {
    case _GPSVersionID:
        return ifd.checkGPSVersionID( )
    }
    return fmt.Errorf( "checkGpsTag: unknown or unsupported tag (%#02x) @offset %#04x type %s count %d\n",
                       ifd.fTag, ifd.sOffset-8, getTiffTString( ifd.fType ), ifd.fCount )
}

const (                                     // _IOP IFD tags
    _InteroperabilityIndex      = 0x01
    _InteroperabilityVersion    = 0x02
)

func (ifd *ifdd) checkInteroperabilityVersion( ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "InteroperabilityVersion: invalid type (%s)\n", getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 4 {
        return fmt.Errorf( "InteroperabilityVersion: invalid count (%d)\n", ifd.fCount )
    }
    // assume bytes
    bs := ifd.getUnsignedBytes( )
    if ifd.desc.Print {
        fmt.Printf( "    InteroperabilityVersion: %#02x, %#02x, %#02x, %#02x\n",
                    bs[0], bs[1], bs[2], bs[3] )
    }
    ifd.storeValue( ifd.newUnsignedByteValue( bs ) )
    return nil
}

func checkIopTag( ifd *ifdd ) error {
    switch ifd.fTag {
    case _InteroperabilityIndex:
        return ifd.checkTiffAscii( "Interoperability" )
    case _InteroperabilityVersion:
        return ifd.checkInteroperabilityVersion( )
    default:
        if ifd.desc.Print {
            fmt.Printf( "    unknown or unsupported tag (%#02x) @offset %#04x type %s count %d\n",
                        ifd.fTag, ifd.sOffset-8, getTiffTString( ifd.fType), ifd.fCount )
        }
    }
    return nil
}

// checkIfd makes a new ifdd, check all entries and store the corresponding
// values in the ifdd. It returns the offset of the next ifd in list (0 if
// none), the newly created ifdd and an error if it failed.
func (ed *Desc) checkIFD( id ifdId, start uint32,
                          checkTags func(*ifdd) error ) ( uint32, *ifdd, error ) {

    /*
        Image File Directory starts with the number of following directory entries (2 bytes)
        followed by that number of entries (12 bytes) and one extra offset to the next IFD
        (4 bytes) and is followed by the IFD data area
    */
    ifd := new( ifdd )
    ifd.id = id
    ifd.desc = ed

    nIfdEntries := ed.getUnsignedShort( start )
    ifd.sOffset = start + _ShortSize
    ifd.values = make( []serializer, 0, nIfdEntries )

    fmt.Printf( "New IFD ID=%d: n entries %d first entry @ offset %#08x\n",
                ifd.id, nIfdEntries, ifd.sOffset )

    for i := uint16(0); i < nIfdEntries; i++ {
        ifd.fTag = tTag(ed.getUnsignedShort( ifd.sOffset ))
        ifd.fType = tType(ed.getUnsignedShort( ifd.sOffset + 2 ))
        ifd.fCount = ed.getUnsignedLong( ifd.sOffset + 4 )
        ifd.sOffset += 8

        err := checkTags( ifd )
        if err != nil {
            return 0, nil, fmt.Errorf( "checkIFD: invalid field: %v\n", err )
        }
        ifd.sOffset += 4
    }
    offset := ed.getUnsignedLong( ifd.sOffset )  // next IFD offset in list
    return offset, ifd, nil
}

func getEndianess( data []byte ) ( size uint32, endian binary.ByteOrder, err error) {
    size = 2
    err = nil
    endian = binary.BigEndian

    if bytes.Equal( data[:2], []byte( "II" ) ) {
        endian = binary.LittleEndian
    } else if ! bytes.Equal( data[:2], []byte( "MM" ) ) {
        err = fmt.Errorf( "exif: invalid TIFF header (unknown byte ordering: %v)\n", data[:2] )
    }
    return
}

func Parse( data []byte, start, dLen uint, ec *Control ) (*Desc, error) {
    if ! bytes.Equal( data[start:start+6], []byte( "Exif\x00\x00" ) ) {
        return nil, fmt.Errorf( "exif: invalid signature (%s)\n", string(data[0:6]) )
    }

    // temporary, save exif data into file - exif-src.txt
    {
	    f, err := os.OpenFile( "exif-src.bin", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
        if err != nil { return nil, err }
        _, err = f.Write( data[start:start+dLen] )
        if err != nil { return nil, err }
        if err = f.Close( ); err != nil { return nil, err }
        if err != nil { return nil, err }
    }
    // end of temporary stuff

    ed := new( Desc )   // Exif\0\0 is followed immediately by TIFF header
    ed.data = data[start+_originOffset:start+dLen-_originOffset] // starts @TIFF header
    ed.Control = *ec

    if ed.Print {
        fmt.Printf( "APP1 (EXIF)\n" )
    }

    // TIFF header starts with 2 bytes indicating the byte ordering (little or big endian)
    var err error
    var offset uint32
    offset, ed.endian, err = getEndianess( ed.data )
    if err != nil {
        return nil, err
    }
    // followed by 2-byte 0x002a (according to the endianess)
    validTiff := ed.getUnsignedShort( offset )
    if validTiff != 0x2a {
        return nil, fmt.Errorf( "exif: invalid TIFF header (invalid identifier: %#02x)\n", validTiff )
    }

    // followed by Primary Image File directory (IFD) offset
    offset = ed.getUnsignedLong( offset + 2 )
//    ed.origin = offset

    if ed.Print {
        fmt.Printf( "  Primary Image metadata @%#04x\n", offset )
        fmt.Printf("      ---------------------------------- IFD0 -----------------------------------\n")
    }
    offset, ed.root, err = ed.checkIFD( _PRIMARY, offset, checkTiffTag )
    if err != nil { return nil, err }
    if ed.Print {
        fmt.Printf("      ----------------------------------------------------------------------------\n")
    }

    if offset != 0 {
        if ed.Print {
            fmt.Printf( "  Thumbnail Image metadata @%#04x\n", offset )
            fmt.Printf("      -------------------------------- IFD1 --------------------------------------\n")
        }
        _, ed.root.next, err = ed.checkIFD( _THUMBNAIL, offset, checkTiffTag )
        if err != nil { return nil, err }
        if ed.Print {
            fmt.Printf("      ----------------------------------------------------------------------------\n")
        }
    }

    // temporary, save exif data into file - exif-dst.bin
    {
	    f, err := os.OpenFile( "exif-dst.bin", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
        if err != nil { return nil, err }
        n, err := ed.Write( f )
        if err != nil { return nil, err }
        fmt.Printf( "wrote %d bytes to exif-dst.bin\n", n )
        if err = f.Close( ); err != nil { return nil, err }
        if err != nil { return nil, err }
    }
    return ed, nil
}

func (ed *Desc)GetThumbnail() (uint32, uint32, Compression) {
    offset := ed.tOffset
    if offset != 0 {
        offset += _originOffset
    }
    return offset, ed.tLen, ed.tType
}

