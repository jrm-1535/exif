
package exif

import (
    "fmt"
    "bytes"
    "strings"
    "io"
)

const (                                     // PRIMARY & THUMBNAIL IFD tags
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

func (ifd *ifdd) storeTiffCompression( ) error {
/*
    Exif2-2: optional in Primary IFD and in thumbnail IFD
When a primary image is JPEG compressed, this designation is not necessary and is omitted.
When thumbnails use JPEG compression, this tag value is set to 6.
*/
    c, err := ifd.checkUnsignedShorts( 1 )
    if err == nil {
        var cType Compression
        switch( c[0] ) {
        case 1: cType = NotCompressed
        case 2: cType = CCITT_1D
        case 3: cType = CCITT_Group3
        case 4: cType = CCITT_Group4
        case 5: cType = LZW
        case 6: cType = JPEG
        case 7: cType = JPEG_Technote2
        case 8: cType = Deflate
        case 9: cType = RFC_2301_BW_JBIG
        case 10: cType = RFC_2301_Color_JBIG
        case 32773: cType = PackBits
        default:
            return fmt.Errorf( "Illegal compression (%d)\n", c[0] )
        }

        if ifd.id == PRIMARY {
            if ifd.desc.Warn {
                if cType != JPEG {
                    fmt.Printf("Warning: non-JPEG compression specified in a JPEG file\n" )
                } else {
                    fmt.Printf("Warning: Exif2-2 specifies that in case of JPEG picture compression be omited\n")
                }
            }
        } else {    // _THUMBNAIL
            ifd.desc.global["thumbType"] = cType // remember compression type
        }

        fmtCompression := func( w io.Writer, v interface{} ) {
            c := v.([]uint16)
            var cString string
            switch( c[0] ) {
            case 1: cString = "No compression"
            case 2: cString = "CCITT 1D modified Huffman RLE"
            case 3: cString = "CCITT Group 3 fax encoding"
            case 4: cString = "CCITT Group 4 fax encoding"
            case 5: cString = "LZW"
            case 6: cString = "JPEG"
            case 7: cString = "JPEG (Technote2)"
            case 8: cString = "Deflate"
            case 9: cString = "RFC 2301 (black and white JBIG)."
            case 10: cString = "RFC 2301 (color JBIG)."
            case 32773: cString = "PackBits compression (Macintosh RLE)"
            }
            fmt.Fprintf( w, "%s\n", cString )
        }
        ifd.storeValue( ifd.newUnsignedShortValue( "Compression", fmtCompression, c ) )
    }
    return err
}

func (ifd *ifdd) storeTiffOrientation( ) error {

    fmtv := func( w io.Writer, v interface{} ) {
        o := v.([]uint16)
        var oString string
        switch( o[0] ) {
        case 1: oString = "Row #0 Top, Col #0 Left"
        case 2: oString = "Row #0 Top, Col #0 Right"
        case 3: oString = "Row #0 Bottom, Col #0 Right"
        case 4: oString = "Row #0 Bottom, Col #0 Left"
        case 5: oString = "Row #0 Left, Col #0 Top"
        case 6: oString = "Row #0 Right, Col #0 Top"
        case 7: oString = "Row #0 Right, Col #0 Bottom"
        case 8: oString = "Row #0 Left, Col #0 Bottom"
        default:
            fmt.Fprintf( w, "Illegal orientation (%d)\n", o[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", oString )
    }

    return ifd.storeUnsignedShorts( "Orientation", 1, fmtv )
}

func (ifd *ifdd) storeTiffResolutionUnit( ) error {

    fmtv := func( w io.Writer, v interface{} ) {
        ru := v.([]uint16)
        var ruString string
        switch( ru[0] ) {
        case 1 : ruString = "Dots per Arbitrary unit"
        case 2 : ruString = "Dots per Inch"
        case 3 : ruString = "Dots per Cm"
        default:
            fmt.Fprintf( w, "Illegal resolution unit (%d)\n", ru[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", ruString )
    }
    return ifd.storeUnsignedShorts( "Resolution Unit", 1, fmtv )
}

func (ifd *ifdd) storeTiffYCbCrPositioning( ) error {

    fmtv := func( w io.Writer, v interface{} ) {
        pos := v.([]uint16)
        var posString string
        switch( pos[0] ) {
        case 1 : posString = "Centered"
        case 2 : posString = "Cosited"
        default:
            fmt.Fprintf( w, "Illegal positioning (%d)\n", pos[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", posString )
    }
    return ifd.storeUnsignedShorts( "YCbCr Positioning", 1, fmtv )
}

func (ifd *ifdd) storeJPEGInterchangeFormat( ) error {
    offset, err := ifd.checkUnsignedLongs( 1 )
    if err == nil {
        ifd.desc.global["thumbOffset"] = offset[0]
//        fmt.Printf( "JPEGInterchangeFormat: offset %#08x\n", offset[0] )
//        ifd.storeValue( ifd.newUnsignedLongValue( "", nil, offset ) )
    }
    return err
}

func (ifd *ifdd) storeJPEGInterchangeFormatLength( ) error {
    length, err := ifd.checkUnsignedLongs( 1 )
    if err == nil {
        offset := ifd.desc.global["thumbOffset"].(uint32)
        if offset == 0 {
            return fmt.Errorf("JPEGInterchangeFormatLength without JPEGInterchangeFormat\n")
        }
        ifd.desc.global["thumbLen"] = length[0]
        ifd.storeValue(
            ifd.newThumbnailValue( _JPEGInterchangeFormat,
                                   ifd.desc.data[offset:offset+length[0]] ) )
        ifd.storeValue( ifd.newUnsignedLongValue( "", nil, length ) )
    }
    return err
}

func storeTiffTags( ifd *ifdd ) error {
//    fmt.Printf( "storeTiffTags: tag (%#04x) @offset %#04x type %s count %d\n",
//                 ifd.fTag, ifd.sOffset-8, getTiffTString( ifd.fType ), ifd.fCount )
    switch ifd.fTag {
    case _Compression:
        return ifd.storeTiffCompression( )
    case _FillOrder:

    case _ImageDescription:
        return ifd.storeAsciiString( "ImageDescription" )
    case _Make:
        return ifd.storeAsciiString( "Make" )
    case _Model:
        return ifd.storeAsciiString( "Model" )
    case _Orientation:
        return ifd.storeTiffOrientation( )
    case _XResolution:
        return ifd.storeUnsignedRationals( "XResolution", 1, nil )
    case _YResolution:
        return ifd.storeUnsignedRationals( "YResolution", 1, nil )
    case _ResolutionUnit:
        return ifd.storeTiffResolutionUnit( )
    case _Software:
        return ifd.storeAsciiString( "Software" )
    case _DateTime:
        return ifd.storeAsciiString( "Date" )
    case _HostComputer:
        return ifd.storeAsciiString( "HostComputer" )

    case _JPEGInterchangeFormat:
        return ifd.storeJPEGInterchangeFormat( )
    case _JPEGInterchangeFormatLength:
        return ifd.storeJPEGInterchangeFormatLength( )

    case _YCbCrPositioning:
        return ifd.storeTiffYCbCrPositioning( )

    case _Copyright:
        return ifd.storeAsciiString( "Copyright" )

    case _ExifIFD:
        return ifd.storeEmbeddedIfd( "Exif IFD", EXIF, storeExifTags )
    case  _GpsIFD:
        return ifd.storeEmbeddedIfd( "GPS IFD", GPS, storeGpsTags )

    case _Padding:
        return ifd.processPadding( )
    default:
        return ifd.processUnknownTag( )
    }
    return nil
}

const (                                     // EXIF IFD specific tags
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

func (ifd *ifdd) storeExifVersion( ) error {
  // special case: tiff type is undefined, but it is actually ASCII
    if ifd.fType != _Undefined {
        return fmt.Errorf( "ExifVersion: invalid byte type (%s)\n", getTiffTString( ifd.fType ) )
    }
    text := ifd.getUnsignedBytes( )
    ifd.storeValue( ifd.newAsciiStringValue( "Exif Version", text ) )
    return nil
}

func (ifd *ifdd) storeExifExposureTime( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        et := v.([]unsignedRational)
        fmt.Fprintf( w, "%f seconds\n", float32(et[0].Numerator)/float32(et[0].Denominator) )
    }
    return ifd.storeUnsignedRationals( "Exposure Time", 1, fmtv )
}

func (ifd *ifdd) storeExifExposureProgram( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        ep := v.([]uint16)
        var epString string
        switch ep[0] {
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
            epString = fmt.Sprintf( "Illegal Exposure Program (%d)", ep[0] )
        }
        fmt.Fprintf( w, "%s\n", epString )
    }
    return ifd.storeUnsignedShorts( "Exposure Program", 1, fmtv )
}

func (ifd *ifdd) storeExifComponentsConfiguration( ) error {

    p := func( w io.Writer, v interface{} ) {
        bSlice := v.([]byte)
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
        fmt.Fprintf( w, "%s\n", config.String() )
    }

    return ifd.storeUndefinedAsUnsignedBytes( "Components Configuration", 4, p )
}

func (ifd *ifdd) storeExifSubjectDistance( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        sd := v.([]unsignedRational)
        if sd[0].Numerator == 0 {
            fmt.Fprintf( w, "Unknown\n" )
        } else if sd[0].Numerator == 0xffffffff {
            fmt.Fprintf( w, "Infinity\n" )
        } else {
            fmt.Fprintf( w, "%f meters\n",
                         float32(sd[0].Numerator)/float32(sd[0].Denominator) )
        }
    }
    return ifd.storeUnsignedRationals( "Subject Distance", 1, fmtv )
}

func (ifd *ifdd) storeExifMeteringMode( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        mm := v.([]uint16)
        var mmString string
        switch mm[0] {
        case 0 : mmString = "Unknown"
        case 1 : mmString = "Average"
        case 2 : mmString = "CenterWeightedAverage program"
        case 3 : mmString = "Spot"
        case 4 : mmString = "MultiSpot"
        case 5 : mmString = "Pattern"
        case 6 : mmString = "Partial"
        case 255: mmString = "Other"
        default:
            fmt.Fprintf( w, "Illegal Metering Mode (%d)\n", mm[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", mmString )
    }
    return ifd.storeUnsignedShorts( "Metering Mode", 1, fmtv )
}

func (ifd *ifdd) storeExifLightSource( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        ls := v.([]uint16)
        var lsString string
        switch ls[0] {
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
            fmt.Fprintf( w, "Illegal light source (%d)\n", ls[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", lsString )
    }
    return ifd.storeUnsignedShorts( "Light Source", 1, fmtv )
}

func (ifd *ifdd) storeExifFlash( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        f := v.([]uint16)
        var fString string
        switch f[0] {
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
            fmt.Fprintf( w, "Illegal Flash (%#02x)\n", f[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", fString )
    }
    return ifd.storeUnsignedShorts( "Flash", 1, fmtv )
}

func (ifd *ifdd) storeExifSubjectArea( ) error {
    if ifd.fCount < 2 && ifd.fCount > 4 {
        return fmt.Errorf( "Subject Area: invalid count (%d)\n", ifd.fCount )
    }

    fmsa := func( w io.Writer, v interface{} ) {
        loc := v.([]uint16)
        switch len(loc) {
        case 2:
            fmt.Fprintf( w, "Point x=%d, y=%d\n", loc[0], loc[1] )
        case 3:
            fmt.Fprintf( w, "Circle center x=%d, y=%d diameter=%d\n",
                         loc[0], loc[1], loc[2] )
        case 4:
            fmt.Fprintf( w, "Rectangle center x=%d, y=%d width=%d height=%d\n",
                         loc[0], loc[1], loc[2], loc[3] )
        }
    }
    return ifd.storeUnsignedShorts( "Subject Area", 0, fmsa )
}

func (ifd *ifdd) storeExifMakerNote( ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "MakerNote: invalid type (%s)\n", getTiffTString( ifd.fType ) )
    }
    if ifd.fCount > 4 {
        offset := ifd.desc.getUnsignedLong( ifd.sOffset )
        for _, mn := range makerNotes {
            p := mn.try( ifd, offset )
            if p != nil {
                return p( offset )
            }
        }
    }
    return fmt.Errorf( "storeExifMakerNote: unknown maker\n")
}

func (ifd *ifdd) storeExifUserComment( ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "UserComment: invalid type (%s)\n", getTiffTString( ifd.fType ) )
    }
    if ifd.fCount < 8 {
        return fmt.Errorf( "UserComment: invalid count (%s)\n", ifd.fCount )
    }
    //  first 8 Bytes are the encoding
    offset := ifd.desc.getUnsignedLong( ifd.sOffset )
    ud := ifd.desc.data[offset:offset+ifd.fCount]

    p := func( w io.Writer, v interface{} ) {
        ud := v.([]byte)
        encoding := ud[0:8]
        switch encoding[0] {
        case 0x41:  // ASCII?
            if bytes.Equal( encoding, []byte{ 'A', 'S', 'C', 'I', 'I', 0, 0, 0 } ) {
                fmt.Fprintf( w, " ITU-T T.50 IA5 (ASCII) [%s]\n", string(ud[8:]) )
            }
        case 0x4a: // JIS?
            if bytes.Equal( encoding, []byte{ 'J', 'I', 'S', 0, 0, 0, 0, 0 } ) {
                fmt.Fprintf( w, "JIS X208-1990 (JIS):" )
                dumpData( w, "    UserComment", "      ", ud[8:] )
            }
        case 0x55:  // UNICODE?
            if bytes.Equal( encoding, []byte{ 'U', 'N', 'I', 'C', 'O', 'D', 'E', 0 } ) {
                fmt.Fprintf( w, "Unicode Standard:" )
                dumpData( w, "    UserComment", "      ", ud[8:] )
            }
        case 0x00:  // Undefined
            if bytes.Equal( encoding, []byte{ 0, 0, 0, 0, 0, 0, 0, 0 } ) {
                fmt.Fprintf( w, "Undefined encoding:" )
                dumpData( w, "    UserComment", "      ", ud[8:] )
            }
        default:
            fmt.Fprintf( w, "Invalid encoding\n" )
        }
    }
    ifd.storeValue( ifd.newUnsignedByteValue( "User Comment", p, ud ) )
    return nil
}

func (ifd *ifdd) storeExifFlashpixVersion( ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "FlashpixVersion: invalid type (%s)\n", getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 4 {
        return fmt.Errorf( "FlashpixVersion: incorrect count (%d)\n", ifd.fCount )
    }
    text := ifd.getUnsignedBytes( )
    ifd.storeValue( ifd.newAsciiStringValue( "Flashpix Version", text ) )
    return nil
}

func (ifd *ifdd) storeExifColorSpace( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        cs := v.([]uint16)
        var csString string
        switch cs[0] {
        case 1 : csString = "sRGB"
        case 65535: csString = "Uncalibrated"
        default:
            fmt.Fprintf( w, "Illegal color space (%d)\n", cs[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", csString )
    }
    return ifd.storeUnsignedShorts( "Color Space", 1, fmtv )
}

func (ifd *ifdd) storeExifDimension( name string ) error {
    if ifd.fType == _UnsignedShort {
        return ifd.storeUnsignedShorts( name, 1, nil )
    } else if ifd.fType == _UnsignedLong {
        return ifd.storeUnsignedLongs( name, 1, nil )
    }
    return fmt.Errorf( "%s: invalid type (%s)\n", name, getTiffTString( ifd.fType ) )
}

func (ifd *ifdd) storeExifSensingMethod( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        sm := v.([]uint16)
        var smString string
        switch sm[0] {
        case 1 : smString = "Undefined"
        case 2 : smString = "One-chip color area sensor"
        case 3 : smString = "Two-chip color area sensor"
        case 4 : smString = "Three-chip color area sensor"
        case 5 : smString = "Color sequential area sensor"
        case 7 : smString = "Trilinear sensor"
        case 8 : smString = "Color sequential linear sensor"
        default:
            fmt.Fprintf( w, "Illegal sensing method (%d)\n", sm[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", smString )
    }
    return ifd.storeUnsignedShorts( "Sensing Method", 1, fmtv )
}

func (ifd *ifdd) storeExifFileSource( ) error {
    fmtv := func( w io.Writer, v interface{} ) {  // undfined but expect byte
        bs := v.([]byte)
        if bs[0] != 3 {
            fmt.Fprintf( w, "Illegal file source (%d)\n", bs[0] )
            return
        }
        fmt.Fprintf( w, "Digital Still Camera (DSC)\n" )
    }
    return ifd.storeUndefinedAsUnsignedBytes( "File Source", 1, fmtv )
}

func (ifd *ifdd) storeExifSceneType( ) error {
    fmtv := func( w io.Writer, v interface{} ) {  // undefined but expect byte
        bs := v.([]byte)
        var stString string
        switch bs[0] {
        case 1 : stString = "Directly photographed"
        default:
            fmt.Fprintf( w, "Illegal scene type (%d)\n", bs[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", stString )
    }
    return ifd.storeUndefinedAsUnsignedBytes( "Scene Type", 1, fmtv )
}

func (ifd *ifdd) storeExifCFAPattern( ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "CFAPattern: invalid type (%s)\n", getTiffTString( ifd.fType ) )
    }
    // Since the structure cannot fit in 4 bytes, its location is indicated by an offset
    offset := ifd.desc.getUnsignedLong( ifd.sOffset )
    bSlice := ifd.desc.getUnsignedBytes( offset, ifd.fCount )

    // structure describing the color filter array (CFA)
    // 2 short words: horizontal repeat pixel unit (h), vertical repeat pixel unit (v)
    // followed by h*v bytes, each byte value indicating a color:
    // 0 RED, 1 GREEN, 2 BLUE, 3 CYAN, 4 MAGENTA, 5 YELLOW, 6 WHITE

    hz := ifd.desc.getUnsignedShort( offset )
    vt := ifd.desc.getUnsignedShort( offset + 2 )
    // however, it seems that older microsoft tools do not use the proper endianess,
    // so check here if the values are consistent with the total count:
    if uint32(hz) * uint32(vt) != ifd.fCount - 4 { // if not try changing endianess
        h1 := ((hz & 0xff) << 8) + (hz >> 8)
        v1 := ((vt & 0xff) << 8) + (vt >> 8)
        if ( uint32(h1) * uint32(v1) != ifd.fCount - 4 ) {
            return fmt.Errorf( "CFAPattern: Invalid repeat patterns(%d,%d) @%#08x\n", hz, vt, ifd.sOffset )
        }
        hz, vt = h1, v1
        if ifd.desc.Warn {
            fmt.Printf("CFAPattern: Warning: incorrect endianess\n")
        }
    }

    // current h and v will still be accessible from f
    p := func( w io.Writer, v interface{} ) {
        c := v.([]byte)
        for i := uint16(4); i < vt; i++ {    // skip first 4 bytes 
            fmt.Fprintf( w, "\n      Row %d:", i )
            for j := uint16(0); j < hz; j++ {
                var s string
                switch c[(i*hz)+j] {
                case 0: s = "RED"
                case 1: s = "GREEN"
                case 2: s = "BLUE"
                case 3: s = "CYAN"
                case 4: s = "MAGENTA"
                case 5: s = "YELLOW"
                case 6: s = "WHITE"
                default:
                    fmt.Fprintf( w, "Invalid color (%d)\n", c[(i*hz)+j] )
                    return
                }
                fmt.Fprintf( w, " %s", s )
            }
        }
        fmt.Fprintf( w, "\n" )
    }
    ifd.storeValue( ifd.newUnsignedByteValue( "Color Filter Array Pattern", p, bSlice ) )
    return nil
}

func (ifd *ifdd) storeExifCustomRendered( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        cr := v.([]uint16)
        switch cr[0] {
        case 0 : fmt.Fprintf( w, "Normal process\n" )
        case 1 : fmt.Fprintf( w, "Custom process\n" )
        default: fmt.Fprintf( w, "Illegal rendering process (%d)\n", cr[0] )
        }
    }
    return ifd.storeUnsignedShorts( "Custom Rendered", 1, fmtv )
}

func (ifd *ifdd) storeExifExposureMode( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        em := v.([]uint16)
        var emString string
        switch em[0] {
        case 0 : emString = "Auto exposure"
        case 1 : emString = "Manual exposure"
        case 3 : emString = "Auto bracket"
        default:
            fmt.Fprintf( w, "Illegal Exposure mode (%d)\n", em[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", emString )
    }
    return ifd.storeUnsignedShorts( "Exposure Mode", 1, fmtv )
}

func (ifd *ifdd) storeExifWhiteBalance( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        wb := v.([]uint16)
        var wbString string
        switch wb[0] {
        case 0 : wbString = "Auto white balance"
        case 1 : wbString = "Manual white balance"
        default:
            fmt.Fprintf( w, "Illegal white balance (%d)\n", wb[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", wbString )
    }
    return ifd.storeUnsignedShorts( "White Balance", 1, fmtv )
}

func (ifd *ifdd) storeExifDigitalZoomRatio( ) error {
    fmv := func( w io.Writer, v interface{} ) {
        dzr := v.([]unsignedRational)
        if dzr[0].Numerator == 0 {
            fmt.Fprintf( w, "not used\n" )
        } else if dzr[0].Denominator == 0 {
            fmt.Fprintf( w, "invalid ratio Denominator (0)\n" )
        } else {
            fmt.Fprintf( w, "%f\n",
                         float32(dzr[0].Numerator)/float32(dzr[0].Denominator) )
        }
    }
    return ifd.storeUnsignedRationals( "Digital-Zoom Ratio", 1, fmv )
}

func (ifd *ifdd) storeExifSceneCaptureType( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        ct := v.([]uint16)
        var sctString string
        switch ct[0] {
        case 0 : sctString = "Standard"
        case 1 : sctString = "Landscape"
        case 2 : sctString = "Portrait"
        case 3 : sctString = "Night scene"
        default:
            fmt.Fprintf( w, "Illegal scene capture type (%d)\n", ct[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", sctString )
    }
    return ifd.storeUnsignedShorts( "Scene-Capture Type", 1, fmtv )
}

func (ifd *ifdd) storeExifGainControl( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        gc := v.([]uint16)
        var gcString string
        switch gc[0] {
        case 0 : gcString = "none"
        case 1 : gcString = "Low gain up"
        case 2 : gcString = "high gain up"
        case 3 : gcString = "low gain down"
        case 4 : gcString = "high gain down"
        default:
            fmt.Fprintf( w, "Illegal gain control (%d)\n", gc[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", gcString )
    }
    return ifd.storeUnsignedShorts( "Gain Control", 1, fmtv )
}

func (ifd *ifdd) storeExifContrast( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        c := v.([]uint16)
        var cString string
        switch c[0] {
        case 0 : cString = "Normal"
        case 1 : cString = "Soft"
        case 2 : cString = "Hard"
        default:
            fmt.Fprintf( w, "Illegal contrast (%d)\n", c[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", cString )
    }
    return ifd.storeUnsignedShorts( "Contrast", 1, fmtv )
}

func (ifd *ifdd) storeExifSaturation( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        s := v.([]uint16)
        var sString string
        switch s[0] {
        case 0 : sString = "Normal"
        case 1 : sString = "Low saturation"
        case 2 : sString = "High saturation"
        default:
            fmt.Fprintf( w, "Illegal Saturation (%d)\n", s[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", sString )
    }
    return ifd.storeUnsignedShorts( "Saturation", 1, fmtv )
}

func (ifd *ifdd) storeExifSharpness( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        s := v.([]uint16)
        var sString string
        switch s[0] {
        case 0 : sString = "Normal"
        case 1 : sString = "Soft"
        case 2 : sString = "Hard"
        default:
            fmt.Fprintf( w, "Illegal Sharpness (%d)\n", s[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", sString )
    }
    return ifd.storeUnsignedShorts( "Sharpness", 1, fmtv )
}

func (ifd *ifdd) storeExifDistanceRange( ) error {
    fmtv := func( w io.Writer, v interface{} ) {
        dr := v.([]uint16)
        var drString string
        switch dr[0] {
        case 0 : drString = "Unknown"
        case 1 : drString = "Macro"
        case 2 : drString = "Close View"
        case 3 : drString = "Distant View"
        default:
            fmt.Fprintf( w, "Illegal Distance Range (%d)\n", dr[0] )
            return
        }
        fmt.Fprintf( w, "%s\n", drString )
    }
    return ifd.storeUnsignedShorts( "Distance Range", 1, fmtv )
}

func (ifd *ifdd) storeExifLensSpecification( ) error {
// LensSpecification is an array of ordered unsignedRational values:
//  minimum focal length
//  maximum focal length
//  minimum F number in minimum focal length
//  maximum F number in maximum focal length
//  which are specification information for the lens that was used in photography.
//  When the minimum F number is unknown, the notation is 0/0.

    fmls := func( w io.Writer, v interface{} ) {
        ls := v.([]unsignedRational)

        fmt.Fprintf( w, "\n     minimum focal length: %d/%d=%f\n",
                    ls[0].Numerator, ls[0].Denominator,
                    float32(ls[0].Numerator)/float32(ls[0].Denominator) )
        fmt.Fprintf( w, "     maximum focal length: %d/%d=%f\n",
                    ls[1].Numerator, ls[1].Denominator,
                    float32(ls[1].Numerator)/float32(ls[1].Denominator) )
        fmt.Fprintf( w, "     minimum F number: %d/%d=%f\n",
                    ls[2].Numerator, ls[2].Denominator,
                    float32(ls[2].Numerator)/float32(ls[2].Denominator) )
        fmt.Fprintf( w, "     maximum F number: %d/%d=%f\n",
                    ls[3].Numerator, ls[3].Denominator,
                    float32(ls[3].Numerator)/float32(ls[3].Denominator) )
    }
    return ifd.storeUnsignedRationals( "Lens Specification", 4, fmls )
}

func storeExifTags( ifd *ifdd ) error {
//    fmt.Printf( "storeExifTags: tag (%#04x) @offset %#04x type %s count %d\n",
//                 ifd.fTag, ifd.sOffset-8, getTiffTString( ifd.fType ), ifd.fCount )
    switch ifd.fTag {
    case _ExposureTime:
        return ifd.storeExifExposureTime( )
    case _FNumber:
        return ifd.storeUnsignedRationals( "FNumber", 1, nil )
    case _ExposureProgram:
        return ifd.storeExifExposureProgram( )

    case _ISOSpeedRatings:
        return ifd.storeUnsignedShorts( "ISO Speed Ratings", 1, nil )
    case _ExifVersion:
        return ifd.storeExifVersion( )

    case _DateTimeOriginal:
        return ifd.storeAsciiString( "DateTime Original" )
    case _DateTimeDigitized:
        return ifd.storeAsciiString( "DateTime Digitized" )

    case _OffsetTime:
        return ifd.storeAsciiString( "Offset Time" )
    case _OffsetTimeOriginal:
        return ifd.storeAsciiString( "Offset Time Original" )
    case _OffsetTimeDigitized:
        return ifd.storeAsciiString( "Offset Time Digitized" )

    case _ComponentsConfiguration:
        return ifd.storeExifComponentsConfiguration( )
    case _CompressedBitsPerPixel:
        return ifd.storeUnsignedRationals( "Compressed Bits Per Pixel", 1, nil )
    case _ShutterSpeedValue:
        return ifd.storeSignedRationals( "Shutter Speed Value", 1, nil )
    case _ApertureValue:
        return ifd.storeUnsignedRationals( "Aperture Value", 1, nil )
    case _BrightnessValue:
        return ifd.storeSignedRationals( "Brightness Value", 1, nil )
    case _ExposureBiasValue:
        return ifd.storeSignedRationals( "Exposure Bias Value", 1, nil )
    case _MaxApertureValue:
        return ifd.storeUnsignedRationals( "Max Aperture Value", 1, nil )
    case _SubjectDistance:
        return ifd.storeExifSubjectDistance( )
    case _MeteringMode:
        return ifd.storeExifMeteringMode( )
    case _LightSource:
        return ifd.storeExifLightSource( )
    case _Flash:
        return ifd.storeExifFlash( )
    case _FocalLength:
        return ifd.storeUnsignedRationals( "Focal Length", 1, nil )
    case _SubjectArea:
        return ifd.storeExifSubjectArea( )

    case _MakerNote:
        return ifd.storeExifMakerNote( )
    case _UserComment:
        return ifd.storeExifUserComment( )
    case _SubsecTime:
        return ifd.storeAsciiString( "Subsec Time" )
    case _SubsecTimeOriginal:
        return ifd.storeAsciiString( "Subsec Time Original" )
    case _SubsecTimeDigitized:
        return ifd.storeAsciiString( "Subsec Time Digitized" )
    case _FlashpixVersion:
        return ifd.storeExifFlashpixVersion( )

    case _ColorSpace:
        return ifd.storeExifColorSpace( )
    case _PixelXDimension:
        return ifd.storeExifDimension( "PixelX Dimension" )
    case _PixelYDimension:
        return ifd.storeExifDimension( "PixelY Dimension" )

    case _SensingMethod:
        return ifd.storeExifSensingMethod( )
    case _FileSource:
        return ifd.storeExifFileSource( )
    case _SceneType:
        return ifd.storeExifSceneType( )
    case _CFAPattern:
        return ifd.storeExifCFAPattern( )
    case _CustomRendered:
        return ifd.storeExifCustomRendered( )
    case _ExposureMode:
        return ifd.storeExifExposureMode( )
    case _WhiteBalance:
        return ifd.storeExifWhiteBalance( )
    case _DigitalZoomRatio:
        return ifd.storeExifDigitalZoomRatio( )
    case _FocalLengthIn35mmFilm:
        return ifd.storeUnsignedShorts( "Focal Length In 35mm Film", 1, nil )
    case _SceneCaptureType:
        return ifd.storeExifSceneCaptureType( )
    case _GainControl:
        return ifd.storeExifGainControl( )
    case _Contrast:
        return ifd.storeExifContrast( )
    case _Saturation:
        return ifd.storeExifSaturation( )
    case _Sharpness:
        return ifd.storeExifSharpness( )
    case _SubjectDistanceRange:
        return ifd.storeExifDistanceRange( )
    case _ImageUniqueID:
        return ifd.storeAsciiString( "Image Unique ID " )
    case _LensSpecification:
        return ifd.storeExifLensSpecification( )
    case _LensMake:
        return ifd.storeAsciiString( "Lens Make" )
    case _LensModel:
        return ifd.storeAsciiString( "Lens Model" )

    case _InteroperabilityIFD:
        return ifd.storeEmbeddedIfd( "IOP IFD", IOP, storeIopTags )

    case _Padding:
        return ifd.processPadding( )
    default:
        return ifd.processUnknownTag( )
    }
    return nil
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

func (ifd *ifdd) storeGPSVersionID( ) error {
    p := func( w io.Writer, v interface{} ) {
        vid := v.([]byte)
        fmt.Fprintf( w, "%d.%d.%d.%d\n", vid[0], vid[1], vid[2], vid[3] )
    }
    return ifd.storeUnsignedBytes( "GPS Version ID", 4, p )
}

func storeGpsTags( ifd *ifdd ) error {
    switch ifd.fTag {
    case _GPSVersionID:
        return ifd.storeGPSVersionID( )
    default:
        return ifd.processUnknownTag( )
    }
}

const (                                     // _IOP IFD tags
    _InteroperabilityIndex      = 0x01
    _InteroperabilityVersion    = 0x02
)

func (ifd *ifdd) storeInteroperabilityVersion( ) error {
    p := func( w io.Writer, v interface{} ) {
        bs := v.([]byte)
        fmt.Fprintf( w, "%#02x, %#02x, %#02x, %#02x\n",
                     bs[0], bs[1], bs[2], bs[3] )
    }
    return ifd.storeUndefinedAsUnsignedBytes( "Interoperability Version", 4, p )
}

func storeIopTags( ifd *ifdd ) error {
    switch ifd.fTag {
    case _InteroperabilityIndex:
        return ifd.storeAsciiString( "Interoperability" )
    case _InteroperabilityVersion:
        return ifd.storeInteroperabilityVersion( )
    default:
        return ifd.processUnknownTag( )
    }
}

// storeIfd makes a new ifdd, checks all entries and store the corresponding
// values in the ifdd. It returns the offset of the next ifd in list (0 if
// none), the newly created ifdd and an error if it failed.
func (d *Desc) storeIFD( id IfdId, start uint32,
                         storeTags func(*ifdd) error ) ( uint32, *ifdd, error ) {

    /*
        Image File Directory starts with the number of following directory entries (2 bytes)
        followed by that number of entries (12 bytes) and one extra offset to the next IFD
        (4 bytes) and is followed by the IFD data area
    */
    ifd := new( ifdd )
    ifd.id = id
    ifd.desc = d

    nIfdEntries := d.getUnsignedShort( start )
    ifd.sOffset = start + _ShortSize
    ifd.values = make( []serializer, 0, nIfdEntries )

    if d.ParsDbg {
        fmt.Printf( "storeIFD %s IFD (%d): %d entries\n",
                    ifd.getIfdName(), id, nIfdEntries )
    }

    for i := uint16(0); i < nIfdEntries; i++ {
        ifd.fTag = tTag(d.getUnsignedShort( ifd.sOffset ))
        ifd.fType = tType(d.getUnsignedShort( ifd.sOffset + 2 ))
        ifd.fCount = d.getUnsignedLong( ifd.sOffset + 4 )

        if d.ParsDbg {
            fmt.Printf( "storeIFD %s IFD (%d): entry %d @%#08x: tag %#04x, %d %s\n",
                        ifd.getIfdName(), id, i, ifd.sOffset, ifd.fTag,
                        ifd.fCount, getTiffTString( ifd.fType ) )
        }

        ifd.sOffset += 8
        err := storeTags( ifd )
        if err != nil {
            return 0, nil, fmt.Errorf( "storeIFD: invalid field: %v", err )
        }
        ifd.sOffset += 4
    }
    d.ifds[id] = ifd                            // store in flat ifd array
    offset := d.getUnsignedLong( ifd.sOffset )  // next IFD offset in list
    if d.ParsDbg {
        if offset == 0 {
            fmt.Printf( "storeIFD %s IFD (%d): no next IFD in list\n",
                        ifd.getIfdName(), id )
        } else {
            fmt.Printf( "storeIFD %s IFD (%d): next ifd @offset %#08x\n",
                        ifd.getIfdName(), id, offset )
        }
    }
    return offset, ifd, nil
}

