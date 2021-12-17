
package exif

import (
    "fmt"
    "bytes"
    "strings"
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
            if cType != JPEG {
                fmt.Printf("    Warning: non-JPEG compression specified in a JPEG file\n" )
            } else {
                fmt.Printf("    Warning: Exif2-2 specifies that in case of JPEG picture compression be omited\n")
            }
        } else {    // _THUMBNAIL
            ifd.desc.tType = cType    // remember thumnail compression type
        }

        fmtCompression := func( v interface{} ) {
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
            fmt.Printf( "%s\n", cString )
        }
        ifd.storeValue( ifd.newUnsignedShortValue( "Compression", fmtCompression, c ) )
    }
    return err
}

func (ifd *ifdd) storeTiffOrientation( ) error {

    fmtv := func( v interface{} ) {
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
            fmt.Printf( "Illegal orientation (%d)\n", o[0] )
            return
        }
        fmt.Printf( "%s\n", oString )
    }

    return ifd.storeUnsignedShorts( "Orientation", 1, fmtv )
}

func (ifd *ifdd) storeTiffResolutionUnit( ) error {

    fmtv := func( v interface{} ) {
        ru := v.([]uint16)
        var ruString string
        switch( ru[0] ) {
        case 1 : ruString = "Dots per Arbitrary unit"
        case 2 : ruString = "Dots per Inch"
        case 3 : ruString = "Dots per Cm"
        default:
            fmt.Printf( "Illegal resolution unit (%d)\n", ru[0] )
            return
        }
        fmt.Printf( "%s\n", ruString )
    }
    return ifd.storeUnsignedShorts( "Resolution Unit", 1, fmtv )
}

func (ifd *ifdd) storeTiffYCbCrPositioning( ) error {

    fmtv := func( v interface{} ) {
        pos := v.([]uint16)
        var posString string
        switch( pos[0] ) {
        case 1 : posString = "Centered"
        case 2 : posString = "Cosited"
        default:
            fmt.Printf( "Illegal positioning (%d)\n", pos[0] )
            return
        }
        fmt.Printf( "%s\n", posString )
    }
    return ifd.storeUnsignedShorts( "YCbCr Positioning", 1, fmtv )
}

func (ifd *ifdd) storeJPEGInterchangeFormat( ) error {
    offset, err := ifd.checkUnsignedLongs( 1 )
    if err != nil {
        ifd.desc.tOffset = offset[0]
        ifd.storeValue( ifd.newUnsignedLongValue( "", nil, offset ) )
    }
    return err
}

func (ifd *ifdd) storeJPEGInterchangeFormatLength( ) error {
    offset, err := ifd.checkUnsignedLongs( 1 )
    if err != nil {
        ifd.desc.tLen = offset[0]
        ifd.storeValue( ifd.newUnsignedLongValue( "", nil, offset ) )
    }
    return err
}

func (ifd *ifdd) storeEmbeddedIfd( name string, id IfdId,
                                   checkTags func( ifd *ifdd) error ) error {
    offset, err := ifd.checkUnsignedLongs( 1 )
    if err == nil {
        // recusively process the embedded IFD here
        _, eIfd, err := ifd.desc.checkIFD( id, offset[0], checkTags )
        if err == nil {
            ifd.storeValue( ifd.newIfdValue( eIfd ) )
            ifd.desc.ifds[id] = ifd // store in flat ifd array
        }
    }
    return err
}

func checkTiffTag( ifd *ifdd ) error {
//    fmt.Printf( "checkTiffTag: tag (%#04x) @offset %#04x type %s count %d\n",
//                 ifd.fTag, ifd.sOffset-8, getTiffTString( ifd.fType ), ifd.fCount )
    switch ifd.fTag {
    case _Compression:
        return ifd.storeTiffCompression( )
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
    case _YCbCrPositioning:
        return ifd.storeTiffYCbCrPositioning( )

    case _JPEGInterchangeFormat:
        return ifd.storeJPEGInterchangeFormat( )

    case _JPEGInterchangeFormatLength:
        return ifd.storeJPEGInterchangeFormatLength( )

    case _Copyright:
        return ifd.storeAsciiString( "Copyright" )

    case _ExifIFD:
        return ifd.storeEmbeddedIfd( "Exif IFD", EXIF, checkExifTag )
    case  _GpsIFD:
        return ifd.storeEmbeddedIfd( "GPS IFD", GPS, checkGpsTag )

    case _Padding:
        return nil
    }
    return fmt.Errorf( "checkTiffTag: unknown or unsupported tag (%#02x) @offset %#04x type %s count %d\n",
                       ifd.fTag, ifd.sOffset-8, getTiffTString( ifd.fType ), ifd.fCount )
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

func (ifd *ifdd) checkExifVersion( ) error {
  // special case: tiff type is undefined, but it is actually ASCII
    if ifd.fType != _Undefined {
        return fmt.Errorf( "ExifVersion: invalid byte type (%s)\n", getTiffTString( ifd.fType ) )
    }
    text := ifd.getUnsignedBytes( )
    ifd.storeValue( ifd.newAsciiStringValue( "Exif Version", text ) )
    return nil
}

func (ifd *ifdd) checkExifExposureTime( ) error {
    fmtv := func( v interface{} ) {
        et := v.([]unsignedRational)
        fmt.Printf( "%f seconds\n", float32(et[0].Numerator)/float32(et[0].Denominator) )
    }
    return ifd.storeUnsignedRationals( "Exposure Time", 1, fmtv )
}

func (ifd *ifdd) checkExifExposureProgram( ) error {
    fmtv := func( v interface{} ) {
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
            epString = fmt.Sprintf( "Illegal Exposure Program (%d)\n", ep[0] )
        }
        fmt.Printf( "%s\n", epString )
    }
    return ifd.storeUnsignedShorts( "Exposure Program", 1, fmtv )
}

func (ifd *ifdd) checkExifComponentsConfiguration( ) error {

    p := func( v interface{} ) {
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
        fmt.Printf( "%s\n", config.String() )
    }

    return ifd.storeUndefinedAsBytes( "Components Configuration", 4, p )
}

func (ifd *ifdd) checkExifSubjectDistance( ) error {
    fmtv := func( v interface{} ) {
        sd := v.([]unsignedRational)
        if sd[0].Numerator == 0 {
            fmt.Printf( "Unknown\n" )
        } else if sd[0].Numerator == 0xffffffff {
            fmt.Printf( "Infinity\n" )
        } else {
            fmt.Printf( "%f meters\n",
                        float32(sd[0].Numerator)/float32(sd[0].Denominator) )
        }
    }
    return ifd.storeUnsignedRationals( "Subject Distance", 1, fmtv )
}

func (ifd *ifdd) checkExifMeteringMode( ) error {
    fmtv := func( v interface{} ) {
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
            fmt.Printf( "Illegal Metering Mode (%d)\n", mm[0] )
            return
        }
        fmt.Printf( "%s\n", mmString )
    }
    return ifd.storeUnsignedShorts( "Metering Mode", 1, fmtv )
}

func (ifd *ifdd) checkExifLightSource( ) error {
    fmtv := func( v interface{} ) {
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
            fmt.Printf( "Illegal light source (%d)\n", ls[0] )
            return
        }
        fmt.Printf( "%s\n", lsString )
    }
    return ifd.storeUnsignedShorts( "Light Source", 1, fmtv )
}

func (ifd *ifdd) checkExifFlash( ) error {
    fmtv := func( v interface{} ) {
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
            fmt.Printf( "Illegal Flash (%#02x)\n", f[0] )
            return
        }
        fmt.Printf( "%s\n", fString )
    }
    return ifd.storeUnsignedShorts( "Flash", 1, fmtv )
}

func (ifd *ifdd) checkExifSubjectArea( ) error {
    if ifd.fCount < 2 && ifd.fCount > 4 {
        return fmt.Errorf( "Subject Area: invalid count (%d)\n", ifd.fCount )
    }

    fmsa := func( v interface{} ) {
        loc := v.([]uint16)
        switch len(loc) {
        case 2:
            fmt.Printf( "Point x=%d, y=%d\n", loc[0], loc[1] )
        case 3:
            fmt.Printf( "Circle center x=%d, y=%d diameter=%d\n",
                        loc[0], loc[1], loc[2] )
        case 4:
            fmt.Printf( "Rectangle center x=%d, y=%d width=%d height=%d\n",
                        loc[0], loc[1], loc[2], loc[3] )
        }
    }
    return ifd.storeUnsignedShorts( "Subject Area", 0, fmsa )
}

func (ifd *ifdd) checkExifMakerNote( ) error {
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
    ud := ifd.desc.data[offset:offset+ifd.fCount]

    p := func( v interface{} ) {
        ud := v.([]byte)
        encoding := ud[0:8]
        switch encoding[0] {
        case 0x41:  // ASCII?
            if bytes.Equal( encoding, []byte{ 'A', 'S', 'C', 'I', 'I', 0, 0, 0 } ) {
                if ifd.desc.Print {
                    fmt.Printf( " ITU-T T.50 IA5 (ASCII) [%s]\n", string(ud[8:]) )
                }
            }
        case 0x4a: // JIS?
            if bytes.Equal( encoding, []byte{ 'J', 'I', 'S', 0, 0, 0, 0, 0 } ) {
                if ifd.desc.Print {
                    fmt.Printf( "JIS X208-1990 (JIS):" )
                    dumpData( "    UserComment", "      ", ud[8:] )
                }
            }
        case 0x55:  // UNICODE?
            if bytes.Equal( encoding, []byte{ 'U', 'N', 'I', 'C', 'O', 'D', 'E', 0 } ) {
                if ifd.desc.Print {
                    fmt.Printf( "Unicode Standard:" )
                    dumpData( "    UserComment", "      ", ud[8:] )
                }
            }
        case 0x00:  // Undefined
            if bytes.Equal( encoding, []byte{ 0, 0, 0, 0, 0, 0, 0, 0 } ) {
                if ifd.desc.Print {
                    fmt.Printf( "Undefined encoding:" )
                    dumpData( "    UserComment", "      ", ud[8:] )
                }
            }
        default:
            fmt.Printf( "Invalid encoding\n" )
        }
    }
    ifd.storeValue( ifd.newUnsignedByteValue( "User Comment", p, ud ) )
    return nil
}

func (ifd *ifdd) checkFlashpixVersion( ) error {
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

func (ifd *ifdd) checkExifColorSpace( ) error {
    fmtv := func( v interface{} ) {
        cs := v.([]uint16)
        var csString string
        switch cs[0] {
        case 1 : csString = "sRGB"
        case 65535: csString = "Uncalibrated"
        default:
            fmt.Printf( "Illegal color space (%d)\n", cs[0] )
            return
        }
        fmt.Printf( "%s\n", csString )
    }
    return ifd.storeUnsignedShorts( "Color Space", 1, fmtv )
}

func (ifd *ifdd) checkExifDimension( name string ) error {
    if ifd.fType == _UnsignedShort {
        return ifd.storeUnsignedShorts( name, 1, nil )
    } else if ifd.fType == _UnsignedLong {
        return ifd.storeUnsignedLongs( name, 1, nil )
    }
    return fmt.Errorf( "%s: invalid type (%s)\n", name, getTiffTString( ifd.fType ) )
}

func (ifd *ifdd) checkExifSensingMethod( ) error {
    fmtv := func( v interface{} ) {
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
            fmt.Printf( "Illegal sensing method (%d)\n", sm[0] )
            return
        }
        fmt.Printf( "%s\n", smString )
    }
    return ifd.storeUnsignedShorts( "Sensing Method", 1, fmtv )
}

func (ifd *ifdd) checkExifFileSource( ) error {
    fmtv := func( v interface{} ) {  // undfined but expect byte
        bs := v.([]byte)
        if bs[0] != 3 {
            fmt.Printf( "Illegal file source (%d)\n", bs[0] )
            return
        }
        fmt.Printf( "Digital Still Camera (DSC)\n" )
    }
    return ifd.storeUndefinedAsBytes( "File Source", 1, fmtv )
}

func (ifd *ifdd) checkExifSceneType( ) error {
    fmtv := func( v interface{} ) {  // undefined but expect byte
        bs := v.([]byte)
        var stString string
        switch bs[0] {
        case 1 : stString = "Directly photographed"
        default:
            fmt.Printf( "Illegal scene type (%d)\n", bs[0] )
            return
        }
        fmt.Printf( "%s\n", stString )
    }
    return ifd.storeUndefinedAsBytes( "Scene Type", 1, fmtv )
}

func (ifd *ifdd) checkExifCFAPattern( ) error {
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
            return fmt.Errorf( "CFAPattern: Invalid repeat patterns(%d,%d)\n", hz, vt )
        }
        hz, vt = h1, v1
        fmt.Printf("CFAPattern: Warning: incorrect endianess\n")
    }

    // current h and v will still be accessible from f
    p := func( v interface{} ) {
        c := v.([]byte)
        for i := uint16(4); i < vt; i++ {    // skip first 4 bytes 
            fmt.Printf("\n      Row %d:", i )
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
                    fmt.Printf( "Invalid color (%d)\n", c[(i*hz)+j] )
                    return
                }
                fmt.Printf( " %s", s )
            }
        }
        fmt.Printf( "\n" )
    }
    ifd.storeValue( ifd.newUnsignedByteValue( "Color Filter Array Pattern", p, bSlice ) )
    return nil
}

func (ifd *ifdd) checkExifCustomRendered( ) error {
    fmtv := func( v interface{} ) {
        cr := v.([]uint16)
        switch cr[0] {
        case 0 : fmt.Printf( "Normal process\n" )
        case 1 : fmt.Printf( "Custom process\n" )
        default: fmt.Printf( "Illegal rendering process (%d)\n", cr[0] )
        }
    }
    return ifd.storeUnsignedShorts( "Custom Rendered", 1, fmtv )
}

func (ifd *ifdd) checkExifExposureMode( ) error {
    fmtv := func( v interface{} ) {
        em := v.([]uint16)
        var emString string
        switch em[0] {
        case 0 : emString = "Auto exposure"
        case 1 : emString = "Manual exposure"
        case 3 : emString = "Auto bracket"
        default:
            fmt.Printf( "Illegal Exposure mode (%d)\n", em[0] )
            return
        }
        fmt.Printf( "%s\n", emString )
    }
    return ifd.storeUnsignedShorts( "Exposure Mode", 1, fmtv )
}

func (ifd *ifdd) checkExifWhiteBalance( ) error {
    fmtv := func( v interface{} ) {
        wb := v.([]uint16)
        var wbString string
        switch wb[0] {
        case 0 : wbString = "Auto white balance"
        case 1 : wbString = "Manual white balance"
        default:
            fmt.Printf( "Illegal white balance (%d)\n", wb[0] )
            return
        }
        fmt.Printf( "%s\n", wbString )
    }
    return ifd.storeUnsignedShorts( "White Balance", 1, fmtv )
}

func (ifd *ifdd) checkExifDigitalZoomRatio( ) error {
    fmv := func( v interface{} ) {
        dzr := v.([]unsignedRational)
        if dzr[0].Numerator == 0 {
            fmt.Printf( "not used\n" )
        } else if dzr[0].Denominator == 0 {
            fmt.Printf( "invalid ratio Denominator (0)\n" )
        } else {
            fmt.Printf( "%f\n",
                        float32(dzr[0].Numerator)/float32(dzr[0].Denominator) )
        }
    }
    return ifd.storeUnsignedRationals( "Digital-Zoom Ratio", 1, fmv )
}

func (ifd *ifdd) checkExifSceneCaptureType( ) error {
    fmtv := func( v interface{} ) {
        ct := v.([]uint16)
        var sctString string
        switch ct[0] {
        case 0 : sctString = "Standard"
        case 1 : sctString = "Landscape"
        case 2 : sctString = "Portrait"
        case 3 : sctString = "Night scene"
        default:
            fmt.Printf( "Illegal scene capture type (%d)\n", ct[0] )
            return
        }
        fmt.Printf( "%s\n", sctString )
    }
    return ifd.storeUnsignedShorts( "Scene-Capture Type", 1, fmtv )
}

func (ifd *ifdd) checkExifGainControl( ) error {
    fmtv := func( v interface{} ) {
        gc := v.([]uint16)
        var gcString string
        switch gc[0] {
        case 0 : gcString = "none"
        case 1 : gcString = "Low gain up"
        case 2 : gcString = "high gain up"
        case 3 : gcString = "low gain down"
        case 4 : gcString = "high gain down"
        default:
            fmt.Printf( "Illegal gain control (%d)\n", gc[0] )
            return
        }
        fmt.Printf( "%s\n", gcString )
    }
    return ifd.storeUnsignedShorts( "Gain Control", 1, fmtv )
}

func (ifd *ifdd) checkExifContrast( ) error {
    fmtv := func( v interface{} ) {
        c := v.([]uint16)
        var cString string
        switch c[0] {
        case 0 : cString = "Normal"
        case 1 : cString = "Soft"
        case 2 : cString = "Hard"
        default:
            fmt.Printf( "Illegal contrast (%d)\n", c[0] )
            return
        }
        fmt.Printf( "%s\n", cString )
    }
    return ifd.storeUnsignedShorts( "Contrast", 1, fmtv )
}

func (ifd *ifdd) checkExifSaturation( ) error {
    fmtv := func( v interface{} ) {
        s := v.([]uint16)
        var sString string
        switch s[0] {
        case 0 : sString = "Normal"
        case 1 : sString = "Low saturation"
        case 2 : sString = "High saturation"
        default:
            fmt.Printf( "Illegal Saturation (%d)\n", s[0] )
            return
        }
        fmt.Printf( "%s\n", sString )
    }
    return ifd.storeUnsignedShorts( "Saturation", 1, fmtv )
}

func (ifd *ifdd) checkExifSharpness( ) error {
    fmtv := func( v interface{} ) {
        s := v.([]uint16)
        var sString string
        switch s[0] {
        case 0 : sString = "Normal"
        case 1 : sString = "Soft"
        case 2 : sString = "Hard"
        default:
            fmt.Printf( "Illegal Sharpness (%d)\n", s[0] )
            return
        }
        fmt.Printf( "%s\n", sString )
    }
    return ifd.storeUnsignedShorts( "Sharpness", 1, fmtv )
}

func (ifd *ifdd) checkExifDistanceRange( ) error {
    fmtv := func( v interface{} ) {
        dr := v.([]uint16)
        var drString string
        switch dr[0] {
        case 0 : drString = "Unknown"
        case 1 : drString = "Macro"
        case 2 : drString = "Close View"
        case 3 : drString = "Distant View"
        default:
            fmt.Printf( "Illegal Distance Range (%d)\n", dr[0] )
            return
        }
        fmt.Printf( "%s\n", drString )
    }
    return ifd.storeUnsignedShorts( "Distance Range", 1, fmtv )
}

func (ifd *ifdd) checkExifLensSpecification( ) error {
// LensSpecification is an array of ordered unsignedRational values:
//  minimum focal length
//  maximum focal length
//  minimum F number in minimum focal length
//  maximum F number in maximum focal length
//  which are specification information for the lens that was used in photography.
//  When the minimum F number is unknown, the notation is 0/0.

    fmls := func( v interface{} ) {
        ls := v.([]unsignedRational)

        fmt.Printf( "\n     minimum focal length: %d/%d=%f\n",
                    ls[0].Numerator, ls[0].Denominator,
                    float32(ls[0].Numerator)/float32(ls[0].Denominator) )
        fmt.Printf( "     maximum focal length: %d/%d=%f\n",
                    ls[1].Numerator, ls[1].Denominator,
                    float32(ls[1].Numerator)/float32(ls[1].Denominator) )
        fmt.Printf( "     minimum F number: %d/%d=%f\n",
                    ls[2].Numerator, ls[2].Denominator,
                    float32(ls[2].Numerator)/float32(ls[2].Denominator) )
        fmt.Printf( "     maximum F number: %d/%d=%f\n",
                    ls[3].Numerator, ls[3].Denominator,
                    float32(ls[3].Numerator)/float32(ls[3].Denominator) )
    }
    return ifd.storeUnsignedRationals( "Lens Specification", 4, fmls )
}

func checkExifTag( ifd *ifdd ) error {
//    fmt.Printf( "checkExifTag: tag (%#04x) @offset %#04x type %s count %d\n",
//                 ifd.fTag, ifd.sOffset-8, getTiffTString( ifd.fType ), ifd.fCount )
    switch ifd.fTag {
    case _ExposureTime:
        return ifd.checkExifExposureTime( )
    case _FNumber:
        return ifd.storeUnsignedRationals( "FNumber", 1, nil )
    case _ExposureProgram:
        return ifd.checkExifExposureProgram( )

    case _ISOSpeedRatings:
        return ifd.storeUnsignedShorts( "ISO Speed Ratings", 1, nil )
    case _ExifVersion:
        return ifd.checkExifVersion( )

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
        return ifd.checkExifComponentsConfiguration( )
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
        return ifd.checkExifSubjectDistance( )
    case _MeteringMode:
        return ifd.checkExifMeteringMode( )
    case _LightSource:
        return ifd.checkExifLightSource( )
    case _Flash:
        return ifd.checkExifFlash( )
    case _FocalLength:
        return ifd.storeUnsignedRationals( "Focal Length", 1, nil )
    case _SubjectArea:
        return ifd.checkExifSubjectArea( )

    case _MakerNote:
        return ifd.checkExifMakerNote( )
    case _UserComment:
        return ifd.checkExifUserComment( )
    case _SubsecTime:
        return ifd.storeAsciiString( "Subsec Time" )
    case _SubsecTimeOriginal:
        return ifd.storeAsciiString( "Subsec Time Original" )
    case _SubsecTimeDigitized:
        return ifd.storeAsciiString( "Subsec Time Digitized" )
    case _FlashpixVersion:
        return ifd.checkFlashpixVersion( )

    case _ColorSpace:
        return ifd.checkExifColorSpace( )
    case _PixelXDimension:
        return ifd.checkExifDimension( "PixelX Dimension" )
    case _PixelYDimension:
        return ifd.checkExifDimension( "PixelY Dimension" )

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
        return ifd.storeUnsignedShorts( "Focal Length In 35mm Film", 1, nil )
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
        return ifd.storeAsciiString( "Image Unique ID " )
    case _LensSpecification:
        return ifd.checkExifLensSpecification( )
    case _LensMake:
        return ifd.storeAsciiString( "Lens Make" )
    case _LensModel:
        return ifd.storeAsciiString( "Lens Model" )

    case _InteroperabilityIFD:
        return ifd.storeEmbeddedIfd( "IOP IFD", IOP, checkIopTag )

    case _Padding:
        return nil
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
    f := func( v interface{} ) {
        slc := v.([]byte)
        fmt.Printf("%d.%d.%d.%d\n", slc[0], slc[1], slc[2], slc[3] )
    }
    ifd.storeValue( ifd.newUnsignedByteValue( "GPS Version ID", f, slc ) )
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
        return fmt.Errorf( "InteroperabilityVersion: invalid type (%s)\n",
                            getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 4 {
        return fmt.Errorf( "InteroperabilityVersion: invalid count (%d)\n",
                           ifd.fCount )
    }
    bs := ifd.getUnsignedBytes( )    // assuming bytes
    f := func( v interface{} ) {
        bs := v.([]byte)
        fmt.Printf( "%#02x, %#02x, %#02x, %#02x\n", bs[0], bs[1], bs[2], bs[3] )
    }
    ifd.storeValue( ifd.newUnsignedByteValue( "Interoperability Version", f, bs ) )
    return nil
}

func checkIopTag( ifd *ifdd ) error {
    switch ifd.fTag {
    case _InteroperabilityIndex:
        return ifd.storeAsciiString( "Interoperability" )
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
func (d *Desc) checkIFD( id IfdId, start uint32,
                         checkTags func(*ifdd) error ) ( uint32, *ifdd, error ) {

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

//    fmt.Printf( "New IFD ID=%d: n entries %d first entry @ offset %#08x\n",
//                ifd.id, nIfdEntries, ifd.sOffset )

    for i := uint16(0); i < nIfdEntries; i++ {
        ifd.fTag = tTag(d.getUnsignedShort( ifd.sOffset ))
        ifd.fType = tType(d.getUnsignedShort( ifd.sOffset + 2 ))
        ifd.fCount = d.getUnsignedLong( ifd.sOffset + 4 )
        ifd.sOffset += 8

        err := checkTags( ifd )
        if err != nil {
            return 0, nil, fmt.Errorf( "checkIFD: invalid field: %v\n", err )
        }
        ifd.sOffset += 4
    }
    offset := d.getUnsignedLong( ifd.sOffset )  // next IFD offset in list
    return offset, ifd, nil
}

