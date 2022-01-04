package exif

// support for Nikon Maker notes

import (
    "fmt"
    "bytes"
    "strings"
    "strconv"
    "math"
    "encoding/binary"
)

// Nikon Preview re-uses some standard TIFF tags
func checkNikon3Preview( ifd *ifdd ) error {

//    fmt.Printf( "checkNikon3Embedded: tag %#04x @offset %#04x type %s count %d\n",
//                 ifd.fTag, ifd.sOffset-8, getTiffTString( ifd.fType ), ifd.fCount )
    switch( ifd.fTag ) {
    case _Compression:
        return ifd.storeTiffCompression( )
    case _XResolution:
        return ifd.storeUnsignedRationals( "XResolution", 1, nil )
    case _YResolution:
        return ifd.storeUnsignedRationals( "YResolution", 1, nil )
    case _ResolutionUnit:
        return ifd.storeTiffResolutionUnit( )
    case _JPEGInterchangeFormat:
        return ifd.storeJPEGInterchangeFormat( )
    case _JPEGInterchangeFormatLength:
        return ifd.storeJPEGInterchangeFormatLength( )
    case _YCbCrPositioning:
        return ifd.storeTiffYCbCrPositioning( )
    default:
        return ifd.processUnknownTag( )
    }
    return nil
}

const (             // Nikon Type 3 Maker note tags
    _Nikon3Version                  = 0x0001  // 1 special: uint or string
    _Nikon3ISOSpeed                  = 0x0002  // 2 _UnsignedShort (0, speed)
    _Nikon3ColorMode                 = 0x0003  // _ASCIIString
    _Nikon3Quality                   = 0x0004  // _ASCIIString
    _Nikon3WhiteBalance              = 0x0005  // _ASCIIString
    _Nikon3Sharpness                 = 0x0006  // _ASCIIString
    _Nikon3FocusMode                 = 0x0007  // _ASCIIString
    _Nikon3FlashSetting              = 0x0008  // _ASCIIString
    _Nikon3FlashType                 = 0x0009  // _ASCIIString
    _Nikon300a                               = 0x00a  // unknown
    _Nikon3WhiteBalanceBias          = 0x000b  // 2 _SignedShort
    _Nikon3WhiteBalanceRBLevels      = 0x000c  // 4 _UnsignedRational
    _Nikon3ProgramShift              = 0x000d  // 4 _Undefined (bytes)
    _Nikon3ExposureDiff              = 0x000e  // 4 _Undefined (bytes)
    _Nikon3ISOSelection                      = 0x000f  // _ASCIIString
    _Nikon3DataDump                          = 0x0010  // n _Undefined (byte)
    _Nikon3Preview                   = 0x0011  // 1 _undefined -> embedded IFD
    _Nikon3FlashExposureCompensation = 0x0012  // 1 _SignedByte (-128,+127)
    _Nikon3ISOSpeedRequested         = 0x0013  // 2 _SignedShort (0, speed)
//    _Nikon3ColorBalance                      = 0x0014  // conflict
    _Nikon3015                               = 0x0015  // string "AUTO"?
    _Nikon3ImageBoundary             = 0x0016  // 4 _UnsignedShort
    _Nikon3ExtFlashExposureComp      = 0x0017  // 4 __Undefined (bytes)
    _Nikon3AEBracketCompensation     = 0x0018  // 4 _Undefined (bytes)
    _Nikon3ExposureBracketValue      = 0x0019  // 1 _SignedRational
    _Nikon3ImageProcessing                   = 0x001a  // string
    _Nikon3CropHiSpeed               = 0x001b  // 7 _UnsignedShort
    _Nikon3ExposureTuning            = 0x001c  // 3 _Undefined (bytes)
    _Nikon3SerialNumber              = 0x001d  // string
    _Nikon3ColorSpace                = 0x001e  // 1 _UnsignedShort
    _Nikon3VRInfo                    = 0x001f  // 8 _Undefined (bytes)

    _Nikon3ActiveDLighting           = 0x0022  // 1 _UnsignedShort
    _Nikon3PictureControlData        = 0x0023  // 58 _Undefined
    _Nikon3WorldTime                 = 0x0024  // 4 _Undefined
    _Nikon3ISOInfo                   = 0x0025  // 14 _Undefined

    _Nikon3DistortInfo               = 0x002b  // 16 _Undefined
    _Nikon302c                       = 0x002c  // 94 _Undefined

    _Nikon3ImageAdjustment                   = 0x0080  // _ASCIIString
    _Nikon3ToneCompensation                  = 0x0081  // _ASCIIString
    _Nikon3AuxillaryLens                     = 0x0082  // _ASCIIString
    _Nikon3LensType                  = 0x0083  // _UnsignedByte
    _Nikon3LensInfo                  = 0x0084  // 1 _UnsignedRational 
    _Nikon3ManualFocusDistance               = 0x0085  // 1 _SignedByte
    _Nikon3DigitalZoomFactor                 = 0x0086  // 1 _SignedByte
    _Nikon3FlashMode                 = 0x0087  // 1 _UnsignedByte
    _Nikon3AutoFocusArea                     = 0x0088  // 3 _SignedByte
    _Nikon3ShootingMode              = 0x0089  // 1 _UnsignedShort
    _Nikon308a                       = 0x008a  // 1 _UnsignedShort
    _Nikon3LensFStops                = 0x008b  // 4 _Undefined
    _Nikon3ContrastCurve                     = 0x008c  // ? _Undefined
    _Nikon3ColorHue                          = 0x008d  // string

    _Nikon3SceneMode                         = 0x008f  // 1 _UnsignedByte
    _Nikon3LightSource                       = 0x0090  // 1 string
    _Nikon3ShotInfo                  = 0x0091  // n _Undefined
    _Nikon3HueAdjustment                     = 0x0092  // 1 _UnsignedByte
    _Nikon3NEFCompression                    = 0x0093  // 1 _UnsignedShort
    _Nikon3Saturation                        = 0x0094  // 1 _SignedByte
    _Nikon3NoiseReduction            = 0x0095  // _ASCIIString
    _NikonLinearizationTable                 = 0x0096  // ? _Undefined
    _Nikon3ColorBalance              = 0x0097  // 1302 _Undefined
    _Nikon3LensData                  = 0x0098  // 33 _Undefined

    _Nikon3DateStampMode             = 0x009d  // 1 _UnsignedShort
    _Nikon3RetouchHistory            = 0x009e  // 10 _UnsignedShort

    _Nikon3ImageSize                  = 0x00a2 // 1 _UnsignedLong
    _Nikon30a3                        = 0x00a3 // 1 _UnsignedByte

    _Nikon3ShutterCount              = 0x00a7  // 1 _UnsignedLong
    _Nikon3FlashInfo                 = 0x00a8  // 2 _undefined
    _Nikon3ImageOptimization                 = 0x00a9  // string

    _Nikon3Saturation2                       = 0x00aa  // string
    _Nikon3DigitalVariProgram        = 0x00ab  // _ASCIIString

    _Nikon3MultiExposure             = 0x00b0  // 16 _Undefined
    _Nikon3HighISONoiseReduction     = 0x00b1  // 1 _UnsignedShort

    _Nikon3PowerUpTime               = 0x00b6  // 8 _Undefined
    _Nikon3AFInfo2                   = 0x00b7  // 30 _Undefined
    _Nikon3FileInfo                  = 0x00b8  // 172 _Undefined

    _Nikon3RetouchInfo                = 0x00bb  // 6 _Undefined
)

func (ifd *ifdd)storeNikon3Version( ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "storeNikon3Version: incorrect type (%s)\n",
                            getTiffTString( ifd.fType ) )
    }
    if ifd.fCount != 4 {
        return fmt.Errorf( "storeNikon3Version: incorrect count (%s)\n",
                           ifd.fCount )
    }
    text := ifd.getUnsignedBytes()
    ifd.storeValue( ifd.newAsciiStringValue( "Nikon maker note type 3 version", text ) )
    return nil
}

func (ifd *ifdd) storeNikon3ISOSpeed( name string ) error {
    fnis := func ( v interface{} ) {
        is := v.([]uint16)
        var hi string
        if is[0] == 1 {
            hi = " (Hi ISO mode)"
        }
        fmt.Printf( "%d%s\n", is[1], hi )
    }
    return ifd.storeUnsignedShorts( name, 2, fnis )
}

func (ifd *ifdd) storeNikon3UndefinedFraction( name string, count uint32,
                                               suffix string ) error {
    if count < 3 {
        panic("storeNikon3UndefinedFraction: too few bytes\n")
    }
    ff := func( v interface{} ) {
        f := v.([]int8)
//        fmt.Printf( "%d, %d, %d, %d\n", f[0], f[1], f[2], f[3] )
        if f[2] == 0 {
            fmt.Printf( "0%s\n", suffix )
        } else {
            fmt.Printf( "%f%s\n",
                        float32(f[0]) * (float32(f[1])/float32(f[2])),
                        suffix )
        }
    }
    return ifd.storeUndefinedAsSignedBytes( name, count, ff )
}

func (ifd *ifdd) storeNikon3ImageBoundary( ) error {
    fib := func( v interface{} ) {
        ib := v.([]uint16)
        fmt.Printf( "top-left: %d,%d bottom-right %d,%d\n",
                    ib[0], ib[1], ib[2], ib[3] )
    }
    return ifd.storeUnsignedShorts( "Image Boundary", 4, fib )
}

var cropCodes = [...]string{
            "off", "1.3x Crop", "DX Crop (1.5x)", "5:4 Crop",
            "3:2 Crop (1.2x)", "", "16:9 Crop", "",
            "2.7x Crop", "DX Movie Crop", "1.4x Movie Crop", "FX Uncropped",
            "DX Uncropped", "", "", "1.5x Movie Crop",
            "", "1:1 Crop" }

func (ifd *ifdd) storeNikon3CropHiSpeed( ) error {
    fchs := func( v interface{} ) {
        chs := v.([]uint16)
        if chs[0] < 18 {
            code := cropCodes[chs[0]]
            if code != "" {
                fmt.Printf( "%s\n", code )
                return
            }
        }
        fmt.Printf( "%dx%d cropped to %dx%d at pixel %d,%d\n",
                    chs[0], chs[1], chs[2], chs[3], chs[4], chs[5], chs[6] )
    }
    return ifd.storeUnsignedShorts( "Crop High Speed", 7, fchs )
}

func (ifd *ifdd) storeNikon3ColorSpace( ) error {
    fcs := func( v interface{} ) {
        cs := v.([]uint16)
        var csString string
        switch cs[0] {
        case 1:     csString = "sRGB"
        case 2:     csString = "Adobe RGB"
        default:    csString = "Unknown"
        }
        fmt.Printf("%s\n", csString )
    }
    return ifd.storeUnsignedShorts( "Color Space", 1, fcs )
}

func (ifd *ifdd) storeNikon3VRInfo( ) error {
    fvr := func( v interface{} ) {
        cs := v.([]uint8)
//        fmt.Printf("%d, %d, %d, %d, %d, %d, %d, %d\n",
//                   cs[0], cs[1], cs[2], cs[3], cs[4], cs[5], cs[6], cs[7] )
        version := string(cs[0:4])
        var state string
        switch cs[4] {
        case 0: state = "n/a"
        case 1: state = "On"
        case 2: state = "Off"
        default: state = "Undefined"
        }
        var mode string
        switch cs[6] {
        case 0: mode = "normal"
        case 2: mode = "active"
        case 3: mode = "sport"
        default: mode = "undefined"
        }
        fmt.Printf("%s Version %s Mode %s\n", state, version, mode )
    }
    return ifd.storeUndefinedAsUnsignedBytes( "Vibration Reduction", 8, fvr )
}

func (ifd *ifdd) storeNikon3ActiveSLighting( ) error {
    fal := func( v interface{} ) {
        al := v.([]uint16)
        var als string
        switch al[0] {
        case 0: als = "Off"
        case 1: als = "Low"
        case 3: als = "Normal"
        case 5: als = "High"
        case 7: als = "Extra High"
        case 8: als = "Extra High 1"
        case 9: als = "Extra High 2"
        case 10: als = "Extra High 3"
        case 11: als = "Extra High 4"
        case 0xffff: als = "Auto"
        default: als = "undefined"
        }
        fmt.Printf( "%s\n", als )
    }
    return ifd.storeUnsignedShorts( "Active D-Lighting", 1, fal )
}

func (ifd *ifdd) storeNikon3PictureControlData( ) error {
    fpcd := func( v interface{} ) {
        pcd := v.([]uint8)
//        dumpData( "Picture Control Data", "     ", pcd )
        var version = string(pcd[0:4])
        var name = string(pcd[4:24])
//        var base = string(pcd[24:44])
        var adjust string
        switch pcd[48] {
        case 0: adjust = "Default Settings"
        case 1: adjust = "Quick Adjust"
        case 3: adjust = "Full Control"
        }
        fmt.Printf( "Version %s %s %s\n", version, name, adjust )

        ppcv := func( name string, v int, norm string, fs string, div float32 ) {
            var t string
            switch v {
            case 0:
                t = norm
            case 127:
                t = "n/a"
            case -128:
                t = "Auto"
            default:
                fmt.Printf( "                       %s: ", name )
                fmt.Printf( fs, float32(v)/div)
                return
            }
            fmt.Printf( "                       %s: %s\n", name, t )
        }

        ppcv( " Quick Adjust", int(pcd[49]-0x80), "Normal", "%d\n", 1 )
        ppcv( " Sharpness", int(pcd[51]-0x80), "None", "%.2f\n", 4 )
        ppcv( " Clarity", int(pcd[53]-0x80), "None", "%.2f\n", 4 )
        ppcv( " Contrast", int(pcd[55]-0x80), "None", "%.2f\n", 4 )
        ppcv( " Brightness", int(pcd[57]-0x80), "None", "%.2f", 4 )
    }
    return ifd.storeUndefinedAsUnsignedBytes( "Picture Control Data", 0, fpcd )
}

func (ifd *ifdd) storeNikon3WorldTime( ) error {
    fwt := func( v interface{} ) {
        wt := v.([]uint8)
        tz := int16(ifd.desc.endian.Uint16( wt ))
        var sign string
        if tz < 0 {
            sign = "-"
            tz = -tz 
        } else {
            sign = "+"
        }
//        fmt.Printf( "tz=0=%#04x ", tz )
        hours := tz / 60
        fmt.Printf( "Time zone %s%dH", sign, hours )
        minutes := tz - (hours * 60)
        if minutes != 0 {
            fmt.Printf( " %02d M", minutes)
        }
        var dls string
        switch wt[2] {
        case 0: dls = "No"
        case 1: dls = "Yes"
        default: dls = "Undefined"
        }
        var df string
        switch wt[3] {
        case 0: df = "Y/M/D"
        case 1: df = "M/D/Y"
        case 2: df = "D/M/Y"
        default: df = "Undefined"
        }
        fmt.Printf( " Daylight savings %s, Date format %s\n", dls, df )
    }
    return ifd.storeUndefinedAsUnsignedBytes( "World Time", 4, fwt )
}

func (ifd *ifdd) storeNikon3ISOInfo( ) error {
    fiso := func( v interface{} ) {
        iso := v.([]uint8)
//        dumpData( "ISO", "     ", iso )
        val := 100  * (1 << ((uint16(iso[0])/12)-5))
        var isoex string
//        exp := conv2U8to1S16( iso[4:], ifd.desc.endian )
        exp := int16(ifd.desc.endian.Uint16( iso[4:] ))
        switch exp {
        case 0: isoex = "off"
        case 0x101: isoex = "Hi 0.3"
        case 0x102: isoex = "Hi 0.5"
        case 0x103: isoex = "Hi 0.7"
        case 0x104: isoex = "Hi 1.0"
        case 0x105: isoex = "Hi 1.3"
        case 0x106: isoex = "Hi 1.5"
        case 0x107: isoex = "Hi 1.7"
        case 0x108: isoex = "Hi 2.0"
        case 0x109: isoex = "Hi 2.3"
        case 0x10a: isoex = "Hi 2.5"
        case 0x10b: isoex = "Hi 2.7"
        case 0x10c: isoex = "Hi 3.0"
        case 0x10d: isoex = "Hi 3.3"
        case 0x10e: isoex = "Hi 3.5"
        case 0x10f: isoex = "Hi 3.7"
        case 0x110: isoex = "Hi 4.0"
        case 0x111: isoex = "Hi 4.3"
        case 0x112: isoex = "Hi 4.5"
        case 0x113: isoex = "Hi 4.7"
        case 0x114: isoex = "Hi 5.0"
        case 0x201: isoex = "Lo 0.3"
        case 0x202: isoex = "Lo 0.5"
        case 0x203: isoex = "Lo 0.7"
        case 0x204: isoex = "Lo 1.0"
        default: isoex = "Undefined"
        }
        v2 := 100 * (1 << ((uint16(iso[6])/12)-5))
        var isoex2 string
//        exp = conv2U8to1S16( iso[10:], ifd.desc.endian )
        exp = int16(ifd.desc.endian.Uint16( iso[10:] ))
        switch exp {
        case 0: isoex2 = "off"
        case 0x101: isoex2 = "Hi 0.3"
        case 0x102: isoex2 = "Hi 0.5"
        case 0x103: isoex2 = "Hi 0.7"
        case 0x104: isoex2 = "Hi 1.0"
        case 0x105: isoex2 = "Hi 1.3"
        case 0x106: isoex2 = "Hi 1.5"
        case 0x107: isoex2 = "Hi 1.7"
        case 0x108: isoex2 = "Hi 2.0"
        case 0x201: isoex2 = "Lo 0.3"
        case 0x202: isoex2 = "Lo 0.5"
        case 0x203: isoex2 = "Lo 0.7"
        case 0x204: isoex2 = "Lo 1.0"
        default: isoex2 = "Undefined"
        }
        fmt.Printf( "%d expansion %s iso2 %d expansion %s\n",
                    val, isoex, v2, isoex2 )
    }
    return ifd.storeUndefinedAsUnsignedBytes( "ISO Info", 14, fiso )
}

func (ifd *ifdd) storeNikon3DistortInfo( ) error {
    fdi := func( v interface{} ) {
        di := v.([]uint8)
//        dumpData( "Distortion", "     ", di )
        version := string(di[:4])
        var control string
        switch di[4] {
        case 0: control = "Off"
        case 1: control = "On"
        case 2: control = "On (underwater)"
        }
        fmt.Printf( "%s version %s\n", control, version )
    }
    return ifd.storeUndefinedAsUnsignedBytes( "Distortion information", 16, fdi )
}

func (ifd *ifdd) storeUndefinedInfo( name string ) error {
    fu := func( v interface{} ) {
        d := v.([]uint8)
        dumpData( "Unknown - Raw data", "     ", d )
    }
    return ifd.storeUndefinedAsUnsignedBytes( name, 0, fu )
}

func makeStringFromBits( v uint16, sa[]string ) string {
    var b strings.Builder
    for i:= 0; i < len(sa); i++ {
        if v & (1<<i) != 0 {
            b.WriteString(sa[i])
        }
    }
    return b.String()
}

func (ifd *ifdd) storeNikon3LensType( ) error {
    flt := func( v interface{} ) {
        lt := v.([]uint8)
        s := makeStringFromBits( uint16(lt[0]),
                                 []string{ "MF ", "D ", "G ", "VR ", 
                                           "1 ", "FT-1 ", "E ", "AF-P " } )
        fmt.Printf( "%s\n", s )
    }
    return ifd.storeUnsignedBytes( "Lens Type", 1, flt )
}

func (ifd *ifdd) storeNikon3LensInfo( ) error {
    fli := func( v interface{} ) {
        li := v.([]unsignedRational)
        fmt.Printf( "Hex %v\n", li )

    }
    return ifd.storeUnsignedRationals( "Lens Info", 4, fli )
}

func (ifd *ifdd) storeNikon3FlashMode() error {
    ffm := func( v interface{} ) {
        fm := v.([]uint8)
        var m string
        switch fm[0] {
        case 0: m = "Did not fire"
        case 1: m = "Fired, manual"
        case 3: m = "Not Ready"
        case 7: m = "Fired, external"
        case 8: m = "Fired, Commander mode"
        case 9: m = "Fired, TTL mode"
        case 18: m = "LED Light"
        default: m = "Unknown"
        }
        fmt.Printf("%s\n", m )
    }
    return ifd.storeUnsignedBytes( "Flash Mode", 1, ffm )
}

func (ifd *ifdd) storeNikon3ShootingMode() error {
    fsm := func( v interface{} ) {
        sm := v.([]uint16)
        //fmt.Printf( "%#04x\n", sm[0] )
        var sf string
        if sm[0] & 0x87 == 0 {
            if sm[0] == 0 {
                fmt.Printf( "Single-Frame\n" )
                return
            }
            sf = "Single-Frame "
        }
        s := makeStringFromBits( sm[0],
                []string{ "Continuous ", "Delay ", "PC Control ",
                          "Self-timer ", "Exposure Bracketing ", "Auto ISO ",
                          "white-Balance Bracketing ", "IR Control",
                          "D-Lighting Braketing" } )
        fmt.Printf( "%s%s\n", sf, s )
    }
    return ifd.storeUnsignedShorts( "Shooting Mode", 1, fsm )
}

func (ifd *ifdd) storeUnknownEntry( name string ) error {
    fu := func( v interface{} ) {
//        u := v.([]uint8)
        fmt.Printf( "Unknown %v\n", v )
    }
    return ifd.storeAnyNonUndefined( name, fu )
}

var xlat0 = [256]byte {
0xc1,0xbf,0x6d,0x0d,0x59,0xc5,0x13,0x9d,0x83,0x61,0x6b,0x4f,0xc7,0x7f,0x3d,0x3d,
0x53,0x59,0xe3,0xc7,0xe9,0x2f,0x95,0xa7,0x95,0x1f,0xdf,0x7f,0x2b,0x29,0xc7,0x0d,
0xdf,0x07,0xef,0x71,0x89,0x3d,0x13,0x3d,0x3b,0x13,0xfb,0x0d,0x89,0xc1,0x65,0x1f,
0xb3,0x0d,0x6b,0x29,0xe3,0xfb,0xef,0xa3,0x6b,0x47,0x7f,0x95,0x35,0xa7,0x47,0x4f,
0xc7,0xf1,0x59,0x95,0x35,0x11,0x29,0x61,0xf1,0x3d,0xb3,0x2b,0x0d,0x43,0x89,0xc1,
0x9d,0x9d,0x89,0x65,0xf1,0xe9,0xdf,0xbf,0x3d,0x7f,0x53,0x97,0xe5,0xe9,0x95,0x17,
0x1d,0x3d,0x8b,0xfb,0xc7,0xe3,0x67,0xa7,0x07,0xf1,0x71,0xa7,0x53,0xb5,0x29,0x89,
0xe5,0x2b,0xa7,0x17,0x29,0xe9,0x4f,0xc5,0x65,0x6d,0x6b,0xef,0x0d,0x89,0x49,0x2f,
0xb3,0x43,0x53,0x65,0x1d,0x49,0xa3,0x13,0x89,0x59,0xef,0x6b,0xef,0x65,0x1d,0x0b,
0x59,0x13,0xe3,0x4f,0x9d,0xb3,0x29,0x43,0x2b,0x07,0x1d,0x95,0x59,0x59,0x47,0xfb,
0xe5,0xe9,0x61,0x47,0x2f,0x35,0x7f,0x17,0x7f,0xef,0x7f,0x95,0x95,0x71,0xd3,0xa3,
0x0b,0x71,0xa3,0xad,0x0b,0x3b,0xb5,0xfb,0xa3,0xbf,0x4f,0x83,0x1d,0xad,0xe9,0x2f,
0x71,0x65,0xa3,0xe5,0x07,0x35,0x3d,0x0d,0xb5,0xe9,0xe5,0x47,0x3b,0x9d,0xef,0x35,
0xa3,0xbf,0xb3,0xdf,0x53,0xd3,0x97,0x53,0x49,0x71,0x07,0x35,0x61,0x71,0x2f,0x43,
0x2f,0x11,0xdf,0x17,0x97,0xfb,0x95,0x3b,0x7f,0x6b,0xd3,0x25,0xbf,0xad,0xc7,0xc5,
0xc5,0xb5,0x8b,0xef,0x2f,0xd3,0x07,0x6b,0x25,0x49,0x95,0x25,0x49,0x6d,0x71,0xc7 }

var xlat1 = [256]byte {
0xa7,0xbc,0xc9,0xad,0x91,0xdf,0x85,0xe5,0xd4,0x78,0xd5,0x17,0x46,0x7c,0x29,0x4c,
0x4d,0x03,0xe9,0x25,0x68,0x11,0x86,0xb3,0xbd,0xf7,0x6f,0x61,0x22,0xa2,0x26,0x34,
0x2a,0xbe,0x1e,0x46,0x14,0x68,0x9d,0x44,0x18,0xc2,0x40,0xf4,0x7e,0x5f,0x1b,0xad,
0x0b,0x94,0xb6,0x67,0xb4,0x0b,0xe1,0xea,0x95,0x9c,0x66,0xdc,0xe7,0x5d,0x6c,0x05,
0xda,0xd5,0xdf,0x7a,0xef,0xf6,0xdb,0x1f,0x82,0x4c,0xc0,0x68,0x47,0xa1,0xbd,0xee,
0x39,0x50,0x56,0x4a,0xdd,0xdf,0xa5,0xf8,0xc6,0xda,0xca,0x90,0xca,0x01,0x42,0x9d,
0x8b,0x0c,0x73,0x43,0x75,0x05,0x94,0xde,0x24,0xb3,0x80,0x34,0xe5,0x2c,0xdc,0x9b,
0x3f,0xca,0x33,0x45,0xd0,0xdb,0x5f,0xf5,0x52,0xc3,0x21,0xda,0xe2,0x22,0x72,0x6b,
0x3e,0xd0,0x5b,0xa8,0x87,0x8c,0x06,0x5d,0x0f,0xdd,0x09,0x19,0x93,0xd0,0xb9,0xfc,
0x8b,0x0f,0x84,0x60,0x33,0x1c,0x9b,0x45,0xf1,0xf0,0xa3,0x94,0x3a,0x12,0x77,0x33,
0x4d,0x44,0x78,0x28,0x3c,0x9e,0xfd,0x65,0x57,0x16,0x94,0x6b,0xfb,0x59,0xd0,0xc8,
0x22,0x36,0xdb,0xd2,0x63,0x98,0x43,0xa1,0x04,0x87,0x86,0xf7,0xa6,0x26,0xbb,0xd6,
0x59,0x4d,0xbf,0x6a,0x2e,0xaa,0x2b,0xef,0xe6,0x78,0xb6,0x4e,0xe0,0x2f,0xdc,0x7c,
0xbe,0x57,0x19,0x32,0x7e,0x2a,0xd0,0xb8,0xba,0x29,0x00,0x3c,0x52,0x7d,0xa8,0x49,
0x3b,0x2d,0xeb,0x25,0x49,0xfa,0xa3,0xaa,0x39,0xa7,0xc5,0xa7,0x50,0x11,0x36,0xfb,
0xc6,0x67,0x4a,0xf5,0xa5,0x12,0x65,0x7e,0xb0,0xdf,0xaf,0x4e,0xb3,0x61,0x7f,0x2f }

// descramble creates a descrambled copy of the received data: it does not modify
// the original data, which can then be stored. This is appropriate since those
// data can be preserved or removed but not modified after parsing. If we were to
// allow modifying the scrambled data we would need a function to re-scramble them
// after modification. This is doable but not necessary in this version.
func (ifd *ifdd) descramble( data []byte ) ([]byte, error) {
    serial, ok := ifd.desc.global["serialKey"].(uint32)
    if ! ok {
        return []byte{}, fmt.Errorf( "descramble: missing serial key\n" )
    }
    count, ok := ifd.desc.global["countKey"].(uint32)
    if ! ok {
        return []byte{}, fmt.Errorf( "descramble: missing count key\n" )
    }
//fmt.Printf("Serial: %#08x count: %#08x\n", serial, count )
    sKey := byte(serial & 0xff)
    cKey := byte(0)
    for i := 0; i < 4; i++ {
        cKey ^= byte((count >> (i*8)))
    }
//fmt.Printf( "\nsKey: %#02x cKey: %#02x\n", sKey, cKey )
    ci := xlat0[sKey]
    cj := xlat1[cKey]
    ck := byte(0x60)
//fmt.Printf( "initial ci: %#02x cj: %#02x ck: %#02x\n", ci, cj, ck )

    dLen := uint32(len(data))
    dsc := make( []byte, dLen )
    for i := uint32(0); i < dLen; i++ {
        cj = cj + ci * ck
        ck ++
        dsc[i] = data[i] ^ cj
    }

    return dsc, nil
}

func (ifd *ifdd) storeNikon3ShotInfo( ) error {
    fu := func( v interface{} ) {
        d := v.([]uint8)
        if string(d[0:4]) == "0215" && len(d) == 6745 {
            dsc, err := ifd.descramble( d[4:0x39a-4] )
            if err == nil {
                fmt.Printf( "Version: %s (%s) Firmware: %s\n",
                        string(d[0:4]), "D5000", string(dsc[0:5]) )
                return
            }
        }
        fmt.Printf( "Version %s\n", string(d[0:4]) )
    }
    return ifd.storeUndefinedAsUnsignedBytes( "Shot Info", 0, fu )
}

func (ifd *ifdd) storeNikon3ColorBalance( ) error {
    fu := func( v interface{} ) {
        d := v.([]uint8)
        if string(d[0:4]) == "0211" {
            dsc, err := ifd.descramble( d[284:284+24] )
            if err == nil {
                wbL0 := ifd.desc.endian.Uint16(dsc[16:])
                wbL1 := ifd.desc.endian.Uint16(dsc[18:])
                wbL2 := ifd.desc.endian.Uint16(dsc[20:])
                wbL3 := ifd.desc.endian.Uint16(dsc[22:])
                fmt.Printf( "Version: %s (%s) WB_GRBGLevels: %d %d %d %d\n",
                        string(d[0:4]), "D5000", wbL0, wbL1, wbL2, wbL3 )
                return
            }
        }
        fmt.Printf( "Version %s\n", string(d[0:4]) )
    }
    return ifd.storeUndefinedAsUnsignedBytes( "Color Balance", 0, fu )
}

func getNikonAperture( v uint8 ) float64 { return math.Exp2(float64(v)/24) }
func getNikonFocalLen( v uint8 ) float64 { return 5 * math.Exp2(float64(v)/24) }
func getNikonExitPupilPosition( v uint8 ) float64 {
    if v == 0 {
        return 0.0
    }
    return 2048.0 / float64(v)
}
func getNikonFocusDistanceString( v uint8 ) string {
    if v == 0 {
        return "inf"
    }
    return fmt.Sprintf( "%0.2f m", 0.01 * math.Pow( 10.0, float64(v)/40 ) )
}

func (ifd *ifdd) storeNikon3LensData( ) error {
    fld := func( v interface{} ) {
        d := v.([]uint8)
        if string(d[0:4]) == "0204" {
            dsc, err := ifd.descramble( d[4:] )
            if err == nil {
                var ids =[8]uint8{
                 dsc[8],dsc[9],dsc[10],dsc[11],dsc[12],dsc[13],dsc[14],dsc[16]}
                m := getLensModel( ids )
                fmt.Printf("Model %s\n", m )
                fmt.Printf( "        Exit pupil position %.1f mm\n",
                            getNikonExitPupilPosition(dsc[0]) )
                fmt.Printf( "        AF Aperture %.1f\n",
                            getNikonAperture( dsc[1] ) )
                fmt.Printf( "        Focus Position %#04x\n", dsc[4] )
                fmt.Printf( "        Focus Distance %s\n",
                            getNikonFocusDistanceString( dsc[6] ) )
                fmt.Printf( "        Focal Length %.1f mm\n",
                            getNikonFocalLen( dsc[7] ) )
                fmt.Printf( "        Min Focal Length %.1f mm\n",
                            getNikonFocalLen( dsc[0x0a] ) )
                fmt.Printf( "        Max Focal Length %.1f mm\n",
                            getNikonFocalLen( dsc[0x0b] ) )
                fmt.Printf( "        Max Aperture At Min Focal Length %.1f\n",
                            getNikonAperture( dsc[0x0c] ) )
                fmt.Printf( "        Max Aperture At Max Focal Length %.1f\n",
                            getNikonAperture( dsc[0x0d] ) )
                fmt.Printf( "        Effective Max Aperture %.1f\n",
                            getNikonAperture( dsc[0x0f] ) )
                fmt.Printf( "        Lens FStops %.2f\n",
                            float32(dsc[9])/12 )
                fmt.Printf( "        MCU Version %d\n", dsc[0xe] )

            }
        }
    }
    return ifd.storeUndefinedAsUnsignedBytes( "Lens", 0, fld )
}

func (ifd *ifdd) storeNikon3DateStampMode() error {
    fds := func( v interface{} ) {
        ds := v.([]uint16)
        var dss string
        switch ds[0] {
        case 0: dss = "Off"
        case 1: dss = "Date & Time"
        case 2: dss = "Date"
        case 3: dss = "Date Counter"
        }
        fmt.Printf( "%s\n", dss )
    }
    return ifd.storeUnsignedShorts( "Date Stamp Mode", 1, fds )
}

func getNikonRetouchString( codes []uint16 ) (rs string) {
    var b strings.Builder
    for _, c := range codes {
        if c < 3 || c > 54 {
            continue
        }
        s := nikonRetouchValues[c-3]
        if s != "" {
            b.WriteString( s )
            b.WriteByte( ' ' )
        }
    }
    rs = b.String()
    if len(rs) == 0 {
        rs = "None"
    }
    return
}

var nikonRetouchValues = [...]string{
//  3                       4                   5                       6
 "B & W",               "Sepia",            "Trim",                 "Small Picture",
//  7                       8                   9                       10
 "D-Lighting",          "Red Eye",          "Cyanotype",            "Sky Light",
//  11                      12                  13                      14
 "Warm Tone",           "Color Custom",     "Image Overlay",        "Red Intensifier",
//  15                      16                  17                      18
 "Green Intensifier",   "Blue Intensifier", "Cross Screen",         "Quick Retouch",
//  19                      20                  21                      22
 "NEF Processing",      "",                 "",                     "",
//  23                      24                  25                      26
 "Distortion Control",  "",                 "Fisheye",              "Straighten",
//  27                      28                  29                      30
 "",                    "",                 "Perspective Control",  "Color Outline",
//  31                      32                  33                      34
 "Soft Filter",         "Resize",           "Miniature Effect",     "Skin Softening",
//  35                      36                  37                      38
 "Selected Frame",      "",                 "Color Sketch",         "Selective Color",
//  39                      40                  41                      42
 "Glamour",             "Drawing",          "",                     "",
//  43                      44                  45                      46
 "",                    "Pop",              "Toy Camera Effect 1",  "Toy Camera Effect 2",
//  47                      48                  49                      50
 "Cross Process (red)", "Cross Process (blue)", "Cross Process (green)", "Cross Process (yellow)",
//  51                      52                      53                  54
 "Super Vivid",         "High-contrast Monochrome", "High Key",     "Low Key" }

func (ifd *ifdd) storeNikon3RetouchHistory() error {
    frh := func( v interface{} ) {
        rh := v.([]uint16)
        fmt.Printf( "%s\n", getNikonRetouchString( rh ) )
    }
    return ifd.storeUnsignedShorts( "Retouch History", 10, frh )
}

func (ifd *ifdd) storeNikon3ImageSize() error {
    fis := func( v interface{} ) {
        is := v.([]uint32)
        fmt.Printf( "%d\n", is[0] )
    }
    return ifd.storeUnsignedLongs( "Compressed Image Size", 1, fis )
}

func (ifd *ifdd) storeNikon3ShutterCount() error {
    fsc := func( v interface{} ) {
        sc := v.([]uint32)
        fmt.Printf( "%d\n", sc[0] )
    }
    return ifd.storeUnsignedLongs( "Shutter Count", 1, fsc )
}

type nikonFlashConv struct {
    ids [2]byte
    name string
}

var nikon3FlashFirmware = [...]nikonFlashConv{
    {[2]byte{0, 0}, "n/a"},
    {[2]byte{1, 1}, "1.01 (SB-800 or Metz 58 AF-1)"},
    {[2]byte{1, 3}, "1.03 (SB-800)"},
    {[2]byte{2, 1}, "2.01 (SB-800)"},
    {[2]byte{2, 4}, "2.04 (SB-600)"},
    {[2]byte{2, 5}, "2.05 (SB-600)"},
    {[2]byte{3, 1}, "3.01 (SU-800 Remote Commander)"},
    {[2]byte{4, 1}, "4.01 (SB-400)"},
    {[2]byte{4, 2}, "4.02 (SB-400)"},
    {[2]byte{4, 4}, "4.04 (SB-400)"},
    {[2]byte{5, 1}, "5.01 (SB-900)"},
    {[2]byte{5, 2}, "5.02 (SB-900)"},
    {[2]byte{6, 1}, "6.01 (SB-700)"},
    {[2]byte{7, 1}, "7.01 (SB-910)"} }

func getExternalFlashFirmware( ids []byte ) string {
    v := [2]byte{ ids[0], ids[1] }
    for _, e := range nikon3FlashFirmware {
        if v == e.ids {
            return e.name
        }
    }
    return fmt.Sprintf("Unknown (%d %d)", ids[0], ids[1] )
}

func getFlashSource( code uint8 ) (s string) {
    switch code {
    case 0: s = "None"
    case 1: s = "External"
    case 2: s = "Internal"
    default: s = "Unknown"
    }
    return
}

func getFlashControlMode( code uint8 ) (m string) {
    switch code {
    case 0: m = "Off"
    case 1: m = "iTTL-BL"
    case 2: m = "iTTL"
    case 3: m = "Auto Aperture"
    case 4: m = "Automatic"
    case 5: m = "GN (distance priority)"
    case 6: m = "Manual"
    case 7: m = "Repeating Flash"
    default: m = "Unknown"
    }
    return
}

func (ifd *ifdd) storeNikon3FlashInfo() error {
    ffi := func( v interface{} ) {
        fi := v.([]uint8)
//        dumpData( "Raw data", "     ", fi )
        fmt.Printf("Version %s", string(fi[0:4]) )
        fmt.Printf(" Source %s", getFlashSource( fi[4] ) )
        if fi[4] == 1 { // external source
            fmt.Printf(" Firmware %s", getExternalFlashFirmware( fi[6:8] ) )
            flags := makeStringFromBits( uint16(fi[8]),
                            []string{ "Fired ", "Bounce Flash ",
                                      "Wide Flash Adapter", "Dome Diffuser" } )
            fmt.Printf(" %s", flags )
        }
        if fi[4] != 0 {
            if fi[9] & 0x80 != 0 {
                fmt.Printf( "Commander Mode On" )
            }
            fcm := fi[9] & 0x7f
            fmt.Printf( " Mode %s", getFlashControlMode( fcm ) )
// TODO: complete decoding later
        }
        fmt.Printf( "\n" )
    }
    return ifd.storeUndefinedAsUnsignedBytes( "Flash Info", 0, ffi )
}

func getNikonMultiExposureMode( mode uint32 ) (mem string) {

    switch mode {
    case 0: mem = "Off"
    case 1: mem = "Multiple exposure"
    case 2: mem = "Image Overlay"
    case 3: mem = "HDR"
    default: mem = "Unknown"
    }
    return
}

func getNikonOnOff( b bool ) string {
    if b {
        return "On"
    }
    return "Off"
}

func (ifd *ifdd) storeNikon3MultiExposure() error {
    fme := func( v interface{} ) {
        me := v.([]uint8)
//        dumpData( "Raw data", "     ", me )
        fmt.Printf("Version %s", string(me[0:4]) )
        var endian binary.ByteOrder
        if me[3] == 0x31 {
            endian = binary.LittleEndian
        } else {
            endian = ifd.desc.endian
        }
        shots := endian.Uint32(me[8:12])
        fmt.Printf( " Mode %s (%d shots)",
                    getNikonMultiExposureMode(endian.Uint32(me[4:8])), shots )
        fmt.Printf( " Auto gain %s\n",
                    getNikonOnOff( 0 != endian.Uint32(me[12:16] ) ))
    }
    return ifd.storeUndefinedAsUnsignedBytes( "Exposure mode", 0, fme )
}

func getNikon3HignISONoiseReduction( hnr uint16 ) (s string) {
    switch hnr {
    case 0: s = "Off"
    case 1: s = "Minimal"
    case 2: s = "Low"
    case 3: s = "Medium Low"
    case 4: s = "Normal"
    case 5: s = "Medium High"
    case 6: s = "High"
    default: s = "Unknown"
    }
    return
}

func (ifd *ifdd) storeNikon3HighISONoiseReduction() error {
    fhnr := func( v interface{} ) {
        hnr := v.([]uint16)
        fmt.Printf( "%s\n", getNikon3HignISONoiseReduction( hnr[0] ) )
    }
    return ifd.storeUnsignedShorts( "High ISO Noise Reduction", 1, fhnr )
}

func (ifd *ifdd) storeNikon3PowerUpTime() error {
    fpu := func( v interface{} ) {
        pu := v.([]uint8)
//        dumpData( "Raw data", "     ", pu )
// 0x0000: 07 e5 06 0d 0e 1a 31 00 
        year := ifd.desc.endian.Uint16(pu[0:2])
        fmt.Printf( "%d/%d/%d %d:%d:%d\n",
                    year, pu[2], pu[3], pu[4], pu[5], pu[6] )
    }
    return ifd.storeUndefinedAsUnsignedBytes( "Power Up", 0, fpu )
}

func getNikon3AFAreaMode( v uint8 ) (m string) {
    switch v {
    case 0: m = "Single Area"
    case 1: m = "Dynamic Area"
    case 2: m = "Dynamic Area (closest subject)"
    case 3: m = "Group Dynamic"
    case 4: m = "Dynamic Area (9 points)"
    case 5: m = "Dynamic Area (21 points)"
    case 6: m = "Dynamic Area (51 points)"
    case 7: m = "Dynamic Area (51 points, 3D-tracking)"
    case 8: m = "Auto-area"
    case 9: m = "Dynamic Area (3D-tracking)"
    case 10: m = "Single Area (wide)"
    case 11: m = "Dynamic Area (wide)"
    case 12: m = "Dynamic Area (wide, 3D-tracking)"
    case 13: m = "Group Area"
    case 14: m = "Dynamic Area (25 points)"
    case 15: m = "Dynamic Area (72 points)"
    case 16: m = "Group Area (HL)"
    case 17: m = "Group Area (VL)"
    case 18: m = "Dynamic Area (49 points)"
    case 128: m = "Single"
    case 129: m = "Auto (41 points)"
    case 130: m = "Subject Tracking (41 points)"
    case 131: m = "Face Priority (41 points)"
    case 192: m = "Pinpoint"
    case 193: m = "Single"
    case 195: m = "Wide (S)"
    case 196: m = "Wide (L)"
    case 197: m = "Auto"
    default: m = "Unknown"
    }
    return
}

func getNikon3ContrastDetectArea( v uint8 ) (m string) {
    switch v {
    case 0: m = "Contrast-detect"
    case 1: m = "Contrast-detect (normal area)"
    case 2: m = "Contrast-detect (wide area)"
    case 3: m = "Contrast-detect (face priority)"
    case 4: m = "Contrast-detect (subject tracking)"
    case 128: m = "Single"
    case 129: m = "Auto (41 points)"
    case 130: m = "Subject Tracking (41 points)"
    case 131: m = "Face Priority (41 points)"
    case 192: m = "Pinpoint"
    case 193: m = "Single"
    case 194: m = "Dynamic"
    case 195: m = "Wide (S)"
    case 196: m = "Wide (L)"
    case 197: m = "Auto"
    case 198: m = "Auto (People)"
    case 199: m = "Auto (Animal)"
    case 200: m = "Normal-area AF"
    case 201: m = "Wide-area AF"
    case 202: m = "Face-priority AF"
    case 203: m = "Subject-tracking AF"
    }
    return
}

func getNikon3PhaseDetectPoints( v uint8 ) (m string) {
    switch v {
    case 0: m = "Off"
    case 1: m = "On (51-point)"
    case 2: m = "On (11-point)"
    case 3: m = "On (39-point)"
    case 4: m = "On (73-point)"
    case 5: m = "On (5)"
    case 6: m = "On (105-point)"
    case 7: m = "On (153-point)"
    case 8: m = "On (81-point)"
    case 9: m = "On (105-point)"
    }
    return
}

func getNikon3Point( v uint8 ) (m string ) {
    switch( v ) {
    case 0: m = "(none)"
    case 1: m = "Center"
    case 2: m = "Top"
    case 3: m = "Bottom"
    case 4: m = "Mid-left"
    case 5: m = "Upper-left"
    case 6: m = "Lower-left"
    case 7: m = "Far Left"
    case 8: m = "Mid-right"
    case 9: m = "Upper-right"
    case 10: m = "Lower-right"
    case 11: m = "Far Right"
    }
    return
}

func getNikon3AFPointsUsed( e binary.ByteOrder, v []uint8 ) string {
    u := e.Uint16( v )
    if u == 0x7f {
        return "All 11 Points"
    }
    return makeStringFromBits( u,
                            []string{ "Center ", "Top ", "Bottom ", "Mid-left",
                                      "Upper-left", "Lower-left", "Far Left",
                                      "Mid-right", "Upper-right", "Lower-left",
                                      "Far Right" } )
}

func (ifd *ifdd) storeNikon3AFInfo2() error {
    fafi := func( v interface{} ) {
        afi := v.([]uint8)
//        dumpData( "Raw data", "     ", afi )
// 0x0000: 30 31 30 30 00 00 02 0b 00 04 00 00 00 00 00 00 0100............
// 0x0010: 00 00 00 00 00 00 00 00 00 00 00 00 00 00       ..............
        fmt.Printf("Version %s Contrast Detect %s ",
                   string(afi[0:4]), getNikonOnOff( 0 != afi[4] ) )
        if 0 == afi[4] { // contrast detect off
            fmt.Printf( "Area Mode %s\n", getNikon3AFAreaMode( afi[5] ) )
        } else {
            fmt.Printf( "Area Mode %s\n", getNikon3ContrastDetectArea( afi[5] ) )
        }
        fmt.Printf( "           Phase Detect AF %s Primary AF Point %s\n",
                    getNikon3PhaseDetectPoints( afi[6] ),
                    getNikon3Point( afi[7] ) )
        if 2 == afi[6] {
            fmt.Printf( "           AF Points Used %s\n",
                        getNikon3AFPointsUsed( ifd.desc.endian, afi[8:10]) )
        }
        if 0 != afi[4] { // contrast detect off
            fmt.Printf( "           AF Image Height %d Width %d X position %d Y position %d\n",
                        ifd.desc.endian.Uint16(afi[10:11]),
                        ifd.desc.endian.Uint16(afi[12:13]),
                        ifd.desc.endian.Uint16(afi[14:15]),
                        ifd.desc.endian.Uint16(afi[16:17]) )
        }
// TODO: add more
    }
    return ifd.storeUndefinedAsUnsignedBytes( "AF Info", 0, fafi )
}

func (ifd *ifdd) storeNikon3FileInfo() error {
    ffi := func( v interface{} ) {
        fi := v.([]uint8)
        fmt.Printf("Version %s Memory card %d Directory # %d file # %d\n",
                   string(fi[0:4]), ifd.desc.endian.Uint16(fi[4:6]),
                   ifd.desc.endian.Uint16(fi[6:8]),
                   ifd.desc.endian.Uint16(fi[8:10]) )
    }
    return ifd.storeUndefinedAsUnsignedBytes( "File Info", 0, ffi )
}

func (ifd *ifdd) storeNikon3RetouchInfo() error {
    fri := func( v interface{} ) {
        ri := v.([]uint8)
//        dumpData( "Raw data", "     ", ri )
//      0x0000: 30 31 30 30 ff 00                               0100..
        fmt.Printf( "Version %s", ri[0:4] )
        if string(ri[0:2]) == "02" {
            fmt.Printf( " NEF Processing %s\n", getNikonOnOff( 1 == ri[5]) )
        } else {
            fmt.Printf( "\n" )
        }
    }
    return ifd.storeUndefinedAsUnsignedBytes( "Retouch Info", 0, fri )
}

func getRationalString( v unsignedRational) string {
    if v.Denominator == 0 {
        return "Inf"
    }
    return fmt.Sprintf( "%.3f", float32(v.Numerator)/float32(v.Denominator) )
}

func (ifd *ifdd) storeNikom3WhiteBalanceRBLevels() error {
    fwb := func( v interface{} ) {
        wb := v.([]unsignedRational)
        fmt.Printf( "%s %s %s %s\n",
                    getRationalString(wb[0]), getRationalString(wb[1]),
                    getRationalString(wb[2]), getRationalString(wb[3]) )
    }
    return ifd.storeUnsignedRationals( "Nikon White Balance Levels", 4, fwb )
}

func storeNikon3Tags( ifd *ifdd ) error {
//    fmt.Printf( "storeNikon3Tags: tag %#04x @offset %#04x type %s count %d\n",
//                 ifd.fTag, ifd.sOffset-8, getTiffTString( ifd.fType ), ifd.fCount )
    switch ifd.fTag {
    case _Nikon3Version:
        return ifd.storeNikon3Version()
    case _Nikon3ISOSpeed:
        return ifd.storeNikon3ISOSpeed( "Nikon ISO Speed" )
    case _Nikon3ColorMode:
        return ifd.storeAsciiString( "Nikon Color Mode" )
    case _Nikon3Quality:
        return ifd.storeAsciiString( "Nikon Quality" )
    case _Nikon3WhiteBalance:
        return ifd.storeAsciiString( "Nikon White Balance" )
    case _Nikon3Sharpness:
        return ifd.storeAsciiString( "Nikon Sharpness" )
    case _Nikon3FocusMode:
        return ifd.storeAsciiString( "Nikon Focus Mode" )
    case _Nikon3FlashSetting:
        return ifd.storeAsciiString( "Nikon Flash Setting" )
    case _Nikon3FlashType:
        return ifd.storeAsciiString( "Nikon Flash Device" )
    case _Nikon3WhiteBalanceBias:
        return ifd.storeSignedShorts( "Nikon White Balance Bias", 2, nil )
    case _Nikon3WhiteBalanceRBLevels:
        return ifd.storeNikom3WhiteBalanceRBLevels( )
    case _Nikon3ProgramShift:
        return ifd.storeNikon3UndefinedFraction( "Nikon Program Shift", 4, "" )
    case _Nikon3ExposureDiff:
        return ifd.storeNikon3UndefinedFraction( "Nikon Exposure Difference", 4, "" )
    case _Nikon3Preview:
        return ifd.storeEmbeddedIfd( "Nikon Preview", EMBEDDED, checkNikon3Preview )
    case _Nikon3FlashExposureCompensation:
        return ifd.storeNikon3UndefinedFraction( "Nikon Flash Exposure Compensation",
                                           4, " EV" )
    case _Nikon3ISOSpeedRequested:
        return ifd.storeNikon3ISOSpeed( "Nikon ISO Speed Requested" )
    case _Nikon3ImageBoundary:
        return ifd.storeNikon3ImageBoundary( )
    case _Nikon3ExtFlashExposureComp:
        return ifd.storeNikon3UndefinedFraction(
                        "Nikon External Flash Exposure Compensation", 4, " EV")
    case _Nikon3AEBracketCompensation:
        return ifd.storeNikon3UndefinedFraction( "Nikon AE Bracket Compensation", 4, " EV" )
    case _Nikon3ExposureBracketValue:
        return ifd.storeSignedRationals( "Nikon Exposure Bracket Value", 1, nil )
    case _Nikon3CropHiSpeed:
        return ifd.storeNikon3CropHiSpeed( )
    case _Nikon3ExposureTuning:
        return ifd.storeNikon3UndefinedFraction( "Nikon Exposure Tuning", 3, "" )
    case _Nikon3SerialNumber:
        return ifd.storeAsciiString( "Serial Number" )
    case _Nikon3ColorSpace:
        return ifd.storeNikon3ColorSpace( )
    case _Nikon3VRInfo:
        return ifd.storeNikon3VRInfo( )
    case _Nikon3ActiveDLighting:
        return ifd.storeNikon3ActiveSLighting( )
    case _Nikon3PictureControlData:
        return ifd.storeNikon3PictureControlData( )
    case _Nikon3WorldTime:
        return ifd.storeNikon3WorldTime( )
    case _Nikon3ISOInfo:
        return ifd.storeNikon3ISOInfo( )
    case _Nikon3DistortInfo:
        return ifd.storeNikon3DistortInfo( )
//    case _Nikon302c:
//        return ifd.storeUndefinedInfo( "Nikon 0x002c" )
    case _Nikon3LensType:
        return ifd.storeNikon3LensType( )
    case _Nikon3LensInfo:
        return ifd.storeExifLensSpecification( )
    case _Nikon3FlashMode:
        return ifd.storeNikon3FlashMode( )
    case _Nikon3ShootingMode:
        return ifd.storeNikon3ShootingMode( )
//    case _Nikon308a:    // 1 _UnsignedShort
//        return ifd.storeUnknownEntry( "Nikon 0x008a" )
    case _Nikon3LensFStops:
        return ifd.storeNikon3UndefinedFraction( "Lens F Stops", 4, "" )
    case _Nikon3ShotInfo:
        return ifd.storeNikon3ShotInfo( )
    case _Nikon3NoiseReduction:
        return ifd.storeAsciiString( "Noise Reduction" )
    case _Nikon3ColorBalance:
        return ifd.storeNikon3ColorBalance( )
    case _Nikon3LensData:
        return ifd.storeNikon3LensData( )
    case _Nikon3DateStampMode:
        return ifd.storeNikon3DateStampMode( )
    case _Nikon3RetouchHistory:
        return ifd.storeNikon3RetouchHistory( )
    case _Nikon3ImageSize:
        return ifd.storeNikon3ImageSize( )
//    case _Nikon30a3:
//        return ifd.storeAnyUnknownSilently( )
    case _Nikon3ShutterCount:
        return ifd.storeNikon3ShutterCount( )
    case _Nikon3FlashInfo:
        return ifd.storeNikon3FlashInfo( )
    case _Nikon3DigitalVariProgram:
        return ifd.storeAsciiString( "Digital VariProgram" )
    case _Nikon3MultiExposure:
        return ifd.storeNikon3MultiExposure( )
    case _Nikon3HighISONoiseReduction:
        return ifd.storeNikon3HighISONoiseReduction( )
    case _Nikon3PowerUpTime:
        return ifd.storeNikon3PowerUpTime( )
    case _Nikon3AFInfo2:
        return ifd.storeNikon3AFInfo2( )
    case _Nikon3FileInfo:
        return ifd.storeNikon3FileInfo( )
    case _Nikon3RetouchInfo:
        return ifd.storeNikon3RetouchInfo( )
    default:
        return ifd.processUnknownTag( )
    }

    return nil
}

func preProcessNikon3Tags( ifd *ifdd ) error {
    switch ifd.fTag {
    case _Nikon3SerialNumber:
        text, err := ifd.checkTiffAsciiString( )
        if err == nil {
            var n int
            n, err = strconv.Atoi( string(text[:7]) )
            if err == nil {
                ifd.desc.global["serialKey"] = uint32(n)
            }
        }
        return err
    case _Nikon3ShutterCount:
        count, err := ifd.checkUnsignedLongs( 1 )
//        fmt.Printf("found _Nikon3ShutterCount: %d, error %v\n", count, err )
        if err == nil {
            ifd.desc.global["countKey"] = count[0]
        }
        return err
    }
    return nil
}

const (
    _NIKON_MAKER_SIGNATURE_1 = "Nikon\x00\x01\x00"
    _NIKON_MAKER_SIGNATURE_1_SIZE = 8

    _NIKON_MAKER_SIGNATURE_3 = "Nikon\x00\x02\x10\x00\x00"
    _NIKON_MAKER_SIGNATURE_3_SIZE = 10

    _NIKON_MAKER_SIGNATURE_4 = "Nikon\x00\x02\x00\x00\x00"
    _NIKON_MAKER_SIGNATURE_4_SIZE = 10

    _NIKON_TIFF_HEADER = "MM\x00\x2a\x00\x00\x00\x08"
    _NIKON_TIFF_HEADER_SIZE = 8
)

func (ifd *ifdd)processNikonMakerNote3( offset uint32 ) error {
//    fmt.Printf( "Offset %#08x Count %d\n", offset, ifd.fCount )
//    dumpData( "Nikon Maker Note Type 3", "   ", ifd.desc.data[offset:offset+ifd.fCount] )

    // Nikon maker notes type 3 looks like an Exif metadata file with its own
    // endianess and own reference (not the same origin as the exif descriptor).
    // It starts with 10-byte identifier: "Nikon\x00\x02\x10\x00\x00", followed
    // by the 2-byte endian idendifier: "MM" for big endian and the 0x2a code,
    // plus the 0x00000008 offset to the regular IFD structure: 2-byte number
    // of entries followed by the regular IFD entries and IFD data, but no
    // next IFD offset at the end.
    offset += _NIKON_MAKER_SIGNATURE_3_SIZE
    count := ifd.fCount - _NIKON_MAKER_SIGNATURE_3_SIZE

//    mknd := new(Desc)
//    mknd.data = ifd.desc.data[offset:offset+count] // starts @TIFF header

    mknd := newDesc( ifd.desc.data[offset:offset+count], &ifd.desc.Control )

    var err error
    mknd.endian, err = getEndianess( mknd.data )
    if err != nil {
        return err
    }
    mknd.Control = ifd.desc.Control     // propagate original control
    offset, err = mknd.checkValidTiff( )
    if err != nil {
        return err
    }


    // collect decryption keys first
    if mknd.ParsDbg {
        fmt.Printf( "processNikonMakerNote3: First pass to collect SerialNumber and ShutterCount\n" )
    }
    _, _, err = mknd.storeIFD( MAKER, offset, preProcessNikon3Tags )
    if err != nil {
        return err
    }
    if mknd.ParsDbg {
        fmt.Printf( "processNikonMakerNote3: Serial %d count %d\n",
                     mknd.global["serialKey"], mknd.global["countKey"] )
        fmt.Printf( "processNikonMakerNote3: Second pass to process all tags\n")
    }
    var nikon *ifdd
    _, nikon, err = mknd.storeIFD( MAKER, offset, storeNikon3Tags )
    if err != nil {
        fmt.Printf("Nikon maker note error: %v", err )
        return err
    }

    // transfer EMBEDDED IFD info to the parent ifd desc 
    ifd.desc.ifds[EMBEDDED] = mknd.ifds[EMBEDDED]
    // Do not transfer thumbnail (i.e. Nikon Preview) to parent ifd desc
//    ifd.desc.global["thumbType"] = mknd.global["thumbType"]
//    ifd.desc.global["thumbOffset"] = mknd.global["thumbOffset"]
//    ifd.desc.global["thumbLen"] = mknd.global["thumbLen"]

    // Note that the ifd end without a next ifd offset
    // TODO: add a parameter to prevent checkIFD to read the next ifd offset?
    mknd.root = nikon
    // TODO: check the endianess for \x00\x2a\x00\x00\x00\x08
    ifd.storeValue( ifd.newDescValue( mknd,
                _NIKON_MAKER_SIGNATURE_3+_NIKON_TIFF_HEADER,
                _NIKON_TIFF_HEADER_SIZE ) )
    ifd.desc.ifds[MAKER] = nikon

//    panic( "Debug" )
    return err
}

func tryNikonMakerNote( ifd *ifdd, offset uint32 ) ( func( uint32 ) error ) {
    if bytes.Equal( ifd.desc.data[offset:offset+_NIKON_MAKER_SIGNATURE_1_SIZE],
                    []byte( _NIKON_MAKER_SIGNATURE_1 ) ) {
        fmt.Printf("    MakerNote: Nikon type 1\n" )
//        return ifd.processNikonMakerNote1
    }
    if bytes.Equal( ifd.desc.data[offset:offset+_NIKON_MAKER_SIGNATURE_3_SIZE],
                    []byte( _NIKON_MAKER_SIGNATURE_3 ) ) {
//        fmt.Printf("    MakerNote: Nikon type 3\n" )
        return ifd.processNikonMakerNote3
    }
    if bytes.Equal( ifd.desc.data[offset:offset+_NIKON_MAKER_SIGNATURE_4_SIZE],
                    []byte( _NIKON_MAKER_SIGNATURE_4 ) ) {
        fmt.Printf("    MakerNote: Nikon type 4\n" )
//        return ifd.processNikonMakerNote3 // common to type 3 & 4
    }
    return nil
}

type lensIdConv struct {
    ids  [8]uint8
    model string
}

// linear search - returns the lendIdConv string or "Unknown" if ids are not found
func getLensModel( ids [8]uint8 ) string {
    for _, c := range lensIDs {
        if c.ids == ids {
            return c.model
        }
    }
    return "Unknown"
}

var lensIDs = [...]lensIdConv{
{[8]uint8{0x01,0x58,0x50,0x50,0x14,0x14,0x02,0x00}, "AF Nikkor 50mm f/1.8"},
{[8]uint8{0x01,0x58,0x50,0x50,0x14,0x14,0x05,0x00}, "AF Nikkor 50mm f/1.8"},
{[8]uint8{0x02,0x42,0x44,0x5C,0x2A,0x34,0x02,0x00}, "AF Zoom-Nikkor 35-70mm f/3.3-4.5"},
{[8]uint8{0x02,0x42,0x44,0x5C,0x2A,0x34,0x08,0x00}, "AF Zoom-Nikkor 35-70mm f/3.3-4.5"},
{[8]uint8{0x03,0x48,0x5C,0x81,0x30,0x30,0x02,0x00}, "AF Zoom-Nikkor 70-210mm f/4"},
{[8]uint8{0x04,0x48,0x3C,0x3C,0x24,0x24,0x03,0x00}, "AF Nikkor 28mm f/2.8"},
{[8]uint8{0x05,0x54,0x50,0x50,0x0C,0x0C,0x04,0x00}, "AF Nikkor 50mm f/1.4"},
{[8]uint8{0x06,0x54,0x53,0x53,0x24,0x24,0x06,0x00}, "AF Micro-Nikkor 55mm f/2.8"},
{[8]uint8{0x07,0x40,0x3C,0x62,0x2C,0x34,0x03,0x00}, "AF Zoom-Nikkor 28-85mm f/3.5-4.5"},
{[8]uint8{0x08,0x40,0x44,0x6A,0x2C,0x34,0x04,0x00}, "AF Zoom-Nikkor 35-105mm f/3.5-4.5"},
{[8]uint8{0x09,0x48,0x37,0x37,0x24,0x24,0x04,0x00}, "AF Nikkor 24mm f/2.8"},
{[8]uint8{0x0A,0x48,0x8E,0x8E,0x24,0x24,0x03,0x00}, "AF Nikkor 300mm f/2.8 IF-ED"},
{[8]uint8{0x0A,0x48,0x8E,0x8E,0x24,0x24,0x05,0x00}, "AF Nikkor 300mm f/2.8 IF-ED N"},
{[8]uint8{0x0B,0x48,0x7C,0x7C,0x24,0x24,0x05,0x00}, "AF Nikkor 180mm f/2.8 IF-ED"},
{[8]uint8{0x0D,0x40,0x44,0x72,0x2C,0x34,0x07,0x00}, "AF Zoom-Nikkor 35-135mm f/3.5-4.5"},
{[8]uint8{0x0E,0x48,0x5C,0x81,0x30,0x30,0x05,0x00}, "AF Zoom-Nikkor 70-210mm f/4"},
{[8]uint8{0x0F,0x58,0x50,0x50,0x14,0x14,0x05,0x00}, "AF Nikkor 50mm f/1.8 N"},
{[8]uint8{0x10,0x48,0x8E,0x8E,0x30,0x30,0x08,0x00}, "AF Nikkor 300mm f/4 IF-ED"},
{[8]uint8{0x11,0x48,0x44,0x5C,0x24,0x24,0x08,0x00}, "AF Zoom-Nikkor 35-70mm f/2.8"},
{[8]uint8{0x11,0x48,0x44,0x5C,0x24,0x24,0x15,0x00}, "AF Zoom-Nikkor 35-70mm f/2.8"},
{[8]uint8{0x12,0x48,0x5C,0x81,0x30,0x3C,0x09,0x00}, "AF Nikkor 70-210mm f/4-5.6"},
{[8]uint8{0x13,0x42,0x37,0x50,0x2A,0x34,0x0B,0x00}, "AF Zoom-Nikkor 24-50mm f/3.3-4.5"},
{[8]uint8{0x14,0x48,0x60,0x80,0x24,0x24,0x0B,0x00}, "AF Zoom-Nikkor 80-200mm f/2.8 ED"},
{[8]uint8{0x15,0x4C,0x62,0x62,0x14,0x14,0x0C,0x00}, "AF Nikkor 85mm f/1.8"},
{[8]uint8{0x17,0x3C,0xA0,0xA0,0x30,0x30,0x0F,0x00}, "Nikkor 500mm f/4 P ED IF"},
{[8]uint8{0x17,0x3C,0xA0,0xA0,0x30,0x30,0x11,0x00}, "Nikkor 500mm f/4 P ED IF"},
{[8]uint8{0x18,0x40,0x44,0x72,0x2C,0x34,0x0E,0x00}, "AF Zoom-Nikkor 35-135mm f/3.5-4.5 N"},
{[8]uint8{0x1A,0x54,0x44,0x44,0x18,0x18,0x11,0x00}, "AF Nikkor 35mm f/2"},
{[8]uint8{0x1B,0x44,0x5E,0x8E,0x34,0x3C,0x10,0x00}, "AF Zoom-Nikkor 75-300mm f/4.5-5.6"},
{[8]uint8{0x1C,0x48,0x30,0x30,0x24,0x24,0x12,0x00}, "AF Nikkor 20mm f/2.8"},
{[8]uint8{0x1D,0x42,0x44,0x5C,0x2A,0x34,0x12,0x00}, "AF Zoom-Nikkor 35-70mm f/3.3-4.5 N"},
{[8]uint8{0x1E,0x54,0x56,0x56,0x24,0x24,0x13,0x00}, "AF Micro-Nikkor 60mm f/2.8"},
{[8]uint8{0x1F,0x54,0x6A,0x6A,0x24,0x24,0x14,0x00}, "AF Micro-Nikkor 105mm f/2.8"},
{[8]uint8{0x20,0x48,0x60,0x80,0x24,0x24,0x15,0x00}, "AF Zoom-Nikkor 80-200mm f/2.8 ED"},
{[8]uint8{0x21,0x40,0x3C,0x5C,0x2C,0x34,0x16,0x00}, "AF Zoom-Nikkor 28-70mm f/3.5-4.5"},
{[8]uint8{0x22,0x48,0x72,0x72,0x18,0x18,0x16,0x00}, "AF DC-Nikkor 135mm f/2"},
{[8]uint8{0x23,0x30,0xBE,0xCA,0x3C,0x48,0x17,0x00}, "Zoom-Nikkor 1200-1700mm f/5.6-8 P ED IF"},
{[8]uint8{0x24,0x48,0x60,0x80,0x24,0x24,0x1A,0x02}, "AF Zoom-Nikkor 80-200mm f/2.8D ED"},
{[8]uint8{0x25,0x48,0x44,0x5C,0x24,0x24,0x1B,0x02}, "AF Zoom-Nikkor 35-70mm f/2.8D"},
{[8]uint8{0x25,0x48,0x44,0x5C,0x24,0x24,0x3A,0x02}, "AF Zoom-Nikkor 35-70mm f/2.8D"},
{[8]uint8{0x25,0x48,0x44,0x5C,0x24,0x24,0x52,0x02}, "AF Zoom-Nikkor 35-70mm f/2.8D"},
{[8]uint8{0x26,0x40,0x3C,0x5C,0x2C,0x34,0x1C,0x02}, "AF Zoom-Nikkor 28-70mm f/3.5-4.5D"},
{[8]uint8{0x27,0x48,0x8E,0x8E,0x24,0x24,0x1D,0x02}, "AF-I Nikkor 300mm f/2.8D IF-ED"},
{[8]uint8{0x27,0x48,0x8E,0x8E,0x24,0x24,0xF1,0x02}, "AF-I Nikkor 300mm f/2.8D IF-ED + TC-14E"},
{[8]uint8{0x27,0x48,0x8E,0x8E,0x24,0x24,0xE1,0x02}, "AF-I Nikkor 300mm f/2.8D IF-ED + TC-17E"},
{[8]uint8{0x27,0x48,0x8E,0x8E,0x24,0x24,0xF2,0x02}, "AF-I Nikkor 300mm f/2.8D IF-ED + TC-20E"},
{[8]uint8{0x28,0x3C,0xA6,0xA6,0x30,0x30,0x1D,0x02}, "AF-I Nikkor 600mm f/4D IF-ED"},
{[8]uint8{0x28,0x3C,0xA6,0xA6,0x30,0x30,0xF1,0x02}, "AF-I Nikkor 600mm f/4D IF-ED + TC-14E"},
{[8]uint8{0x28,0x3C,0xA6,0xA6,0x30,0x30,0xE1,0x02}, "AF-I Nikkor 600mm f/4D IF-ED + TC-17E"},
{[8]uint8{0x28,0x3C,0xA6,0xA6,0x30,0x30,0xF2,0x02}, "AF-I Nikkor 600mm f/4D IF-ED + TC-20E"},
{[8]uint8{0x2A,0x54,0x3C,0x3C,0x0C,0x0C,0x26,0x02}, "AF Nikkor 28mm f/1.4D"},
{[8]uint8{0x2B,0x3C,0x44,0x60,0x30,0x3C,0x1F,0x02}, "AF Zoom-Nikkor 35-80mm f/4-5.6D"},
{[8]uint8{0x2C,0x48,0x6A,0x6A,0x18,0x18,0x27,0x02}, "AF DC-Nikkor 105mm f/2D"},
{[8]uint8{0x2D,0x48,0x80,0x80,0x30,0x30,0x21,0x02}, "AF Micro-Nikkor 200mm f/4D IF-ED"},
{[8]uint8{0x2E,0x48,0x5C,0x82,0x30,0x3C,0x22,0x02}, "AF Nikkor 70-210mm f/4-5.6D"},
{[8]uint8{0x2E,0x48,0x5C,0x82,0x30,0x3C,0x28,0x02}, "AF Nikkor 70-210mm f/4-5.6D"},
{[8]uint8{0x2F,0x48,0x30,0x44,0x24,0x24,0x29,0x02}, "AF Zoom-Nikkor 20-35mm f/2.8D IF"},
{[8]uint8{0x30,0x48,0x98,0x98,0x24,0x24,0x24,0x02}, "AF-I Nikkor 400mm f/2.8D IF-ED"},
{[8]uint8{0x30,0x48,0x98,0x98,0x24,0x24,0xF1,0x02}, "AF-I Nikkor 400mm f/2.8D IF-ED + TC-14E"},
{[8]uint8{0x30,0x48,0x98,0x98,0x24,0x24,0xE1,0x02}, "AF-I Nikkor 400mm f/2.8D IF-ED + TC-17E"},
{[8]uint8{0x30,0x48,0x98,0x98,0x24,0x24,0xF2,0x02}, "AF-I Nikkor 400mm f/2.8D IF-ED + TC-20E"},
{[8]uint8{0x31,0x54,0x56,0x56,0x24,0x24,0x25,0x02}, "AF Micro-Nikkor 60mm f/2.8D"},
{[8]uint8{0x32,0x54,0x6A,0x6A,0x24,0x24,0x35,0x02}, "AF Micro-Nikkor 105mm f/2.8D"},
{[8]uint8{0x33,0x48,0x2D,0x2D,0x24,0x24,0x31,0x02}, "AF Nikkor 18mm f/2.8D"},
{[8]uint8{0x34,0x48,0x29,0x29,0x24,0x24,0x32,0x02}, "AF Fisheye Nikkor 16mm f/2.8D"},
{[8]uint8{0x35,0x3C,0xA0,0xA0,0x30,0x30,0x33,0x02}, "AF-I Nikkor 500mm f/4D IF-ED"},
{[8]uint8{0x35,0x3C,0xA0,0xA0,0x30,0x30,0xF1,0x02}, "AF-I Nikkor 500mm f/4D IF-ED + TC-14E"},
{[8]uint8{0x35,0x3C,0xA0,0xA0,0x30,0x30,0xE1,0x02}, "AF-I Nikkor 500mm f/4D IF-ED + TC-17E"},
{[8]uint8{0x35,0x3C,0xA0,0xA0,0x30,0x30,0xF2,0x02}, "AF-I Nikkor 500mm f/4D IF-ED + TC-20E"},
{[8]uint8{0x36,0x48,0x37,0x37,0x24,0x24,0x34,0x02}, "AF Nikkor 24mm f/2.8D"},
{[8]uint8{0x37,0x48,0x30,0x30,0x24,0x24,0x36,0x02}, "AF Nikkor 20mm f/2.8D"},
{[8]uint8{0x38,0x4C,0x62,0x62,0x14,0x14,0x37,0x02}, "AF Nikkor 85mm f/1.8D"},
{[8]uint8{0x3A,0x40,0x3C,0x5C,0x2C,0x34,0x39,0x02}, "AF Zoom-Nikkor 28-70mm f/3.5-4.5D"},
{[8]uint8{0x3B,0x48,0x44,0x5C,0x24,0x24,0x3A,0x02}, "AF Zoom-Nikkor 35-70mm f/2.8D N"},
{[8]uint8{0x3C,0x48,0x60,0x80,0x24,0x24,0x3B,0x02}, "AF Zoom-Nikkor 80-200mm f/2.8D ED"},
{[8]uint8{0x3D,0x3C,0x44,0x60,0x30,0x3C,0x3E,0x02}, "AF Zoom-Nikkor 35-80mm f/4-5.6D"},
{[8]uint8{0x3E,0x48,0x3C,0x3C,0x24,0x24,0x3D,0x02}, "AF Nikkor 28mm f/2.8D"},
{[8]uint8{0x3F,0x40,0x44,0x6A,0x2C,0x34,0x45,0x02}, "AF Zoom-Nikkor 35-105mm f/3.5-4.5D"},
{[8]uint8{0x41,0x48,0x7C,0x7C,0x24,0x24,0x43,0x02}, "AF Nikkor 180mm f/2.8D IF-ED"},
{[8]uint8{0x42,0x54,0x44,0x44,0x18,0x18,0x44,0x02}, "AF Nikkor 35mm f/2D"},
{[8]uint8{0x43,0x54,0x50,0x50,0x0C,0x0C,0x46,0x02}, "AF Nikkor 50mm f/1.4D"},
{[8]uint8{0x44,0x44,0x60,0x80,0x34,0x3C,0x47,0x02}, "AF,0xZoom-Nikkor,0x80-200mm,0xf/4.5-5.6D"},
{[8]uint8{0x45,0x40,0x3C,0x60,0x2C,0x3C,0x48,0x02}, "AF Zoom-Nikkor 28-80mm f/3.5-5.6D"},
{[8]uint8{0x46,0x3C,0x44,0x60,0x30,0x3C,0x49,0x02}, "AF Zoom-Nikkor 35-80mm f/4-5.6D N"},
{[8]uint8{0x47,0x42,0x37,0x50,0x2A,0x34,0x4A,0x02}, "AF Zoom-Nikkor 24-50mm f/3.3-4.5D"},
{[8]uint8{0x48,0x48,0x8E,0x8E,0x24,0x24,0x4B,0x02}, "AF-S Nikkor 300mm f/2.8D IF-ED"},
{[8]uint8{0x48,0x48,0x8E,0x8E,0x24,0x24,0xF1,0x02}, "AF-S Nikkor 300mm f/2.8D IF-ED + TC-14E"},
{[8]uint8{0x48,0x48,0x8E,0x8E,0x24,0x24,0xE1,0x02}, "AF-S Nikkor 300mm f/2.8D IF-ED + TC-17E"},
{[8]uint8{0x48,0x48,0x8E,0x8E,0x24,0x24,0xF2,0x02}, "AF-S Nikkor 300mm f/2.8D IF-ED + TC-20E"},
{[8]uint8{0x49,0x3C,0xA6,0xA6,0x30,0x30,0x4C,0x02}, "AF-S Nikkor 600mm f/4D IF-ED"},
{[8]uint8{0x49,0x3C,0xA6,0xA6,0x30,0x30,0xF1,0x02}, "AF-S Nikkor 600mm f/4D IF-ED + TC-14E"},
{[8]uint8{0x49,0x3C,0xA6,0xA6,0x30,0x30,0xE1,0x02}, "AF-S Nikkor 600mm f/4D IF-ED + TC-17E"},
{[8]uint8{0x49,0x3C,0xA6,0xA6,0x30,0x30,0xF2,0x02}, "AF-S Nikkor 600mm f/4D IF-ED + TC-20E"},
{[8]uint8{0x4A,0x54,0x62,0x62,0x0C,0x0C,0x4D,0x02}, "AF Nikkor 85mm f/1.4D IF"},
{[8]uint8{0x4B,0x3C,0xA0,0xA0,0x30,0x30,0x4E,0x02}, "AF-S Nikkor 500mm f/4D IF-ED"},
{[8]uint8{0x4B,0x3C,0xA0,0xA0,0x30,0x30,0xF1,0x02}, "AF-S Nikkor 500mm f/4D IF-ED + TC-14E"},
{[8]uint8{0x4B,0x3C,0xA0,0xA0,0x30,0x30,0xE1,0x02}, "AF-S Nikkor 500mm f/4D IF-ED + TC-17E"},
{[8]uint8{0x4B,0x3C,0xA0,0xA0,0x30,0x30,0xF2,0x02}, "AF-S Nikkor 500mm f/4D IF-ED + TC-20E"},
{[8]uint8{0x4C,0x40,0x37,0x6E,0x2C,0x3C,0x4F,0x02}, "AF Zoom-Nikkor 24-120mm f/3.5-5.6D IF"},
{[8]uint8{0x4D,0x40,0x3C,0x80,0x2C,0x3C,0x62,0x02}, "AF Zoom-Nikkor 28-200mm f/3.5-5.6D IF"},
{[8]uint8{0x4E,0x48,0x72,0x72,0x18,0x18,0x51,0x02}, "AF DC-Nikkor 135mm f/2D"},
{[8]uint8{0x4F,0x40,0x37,0x5C,0x2C,0x3C,0x53,0x06}, "IX-Nikkor 24-70mm f/3.5-5.6"},
{[8]uint8{0x50,0x48,0x56,0x7C,0x30,0x3C,0x54,0x06}, "IX-Nikkor 60-180mm f/4-5.6"},
{[8]uint8{0x53,0x48,0x60,0x80,0x24,0x24,0x57,0x02}, "AF Zoom-Nikkor 80-200mm f/2.8D ED"},
{[8]uint8{0x53,0x48,0x60,0x80,0x24,0x24,0x60,0x02}, "AF Zoom-Nikkor 80-200mm f/2.8D ED"},
{[8]uint8{0x54,0x44,0x5C,0x7C,0x34,0x3C,0x58,0x02}, "AF Zoom-Micro Nikkor 70-180mm f/4.5-5.6D ED"},
{[8]uint8{0x54,0x44,0x5C,0x7C,0x34,0x3C,0x61,0x02}, "AF Zoom-Micro Nikkor 70-180mm f/4.5-5.6D ED"},
{[8]uint8{0x56,0x48,0x5C,0x8E,0x30,0x3C,0x5A,0x02}, "AF Zoom-Nikkor 70-300mm f/4-5.6D ED"},
{[8]uint8{0x59,0x48,0x98,0x98,0x24,0x24,0x5D,0x02}, "AF-S Nikkor 400mm f/2.8D IF-ED"},
{[8]uint8{0x59,0x48,0x98,0x98,0x24,0x24,0xF1,0x02}, "AF-S Nikkor 400mm f/2.8D IF-ED + TC-14E"},
{[8]uint8{0x59,0x48,0x98,0x98,0x24,0x24,0xE1,0x02}, "AF-S Nikkor 400mm f/2.8D IF-ED + TC-17E"},
{[8]uint8{0x59,0x48,0x98,0x98,0x24,0x24,0xF2,0x02}, "AF-S Nikkor 400mm f/2.8D IF-ED + TC-20E"},
{[8]uint8{0x5A,0x3C,0x3E,0x56,0x30,0x3C,0x5E,0x06}, "IX-Nikkor 30-60mm f/4-5.6"},
{[8]uint8{0x5B,0x44,0x56,0x7C,0x34,0x3C,0x5F,0x06}, "IX-Nikkor 60-180mm f/4.5-5.6"},
{[8]uint8{0x5D,0x48,0x3C,0x5C,0x24,0x24,0x63,0x02}, "AF-S Zoom-Nikkor 28-70mm f/2.8D IF-ED"},
{[8]uint8{0x5E,0x48,0x60,0x80,0x24,0x24,0x64,0x02}, "AF-S Zoom-Nikkor 80-200mm f/2.8D IF-ED"},
{[8]uint8{0x5F,0x40,0x3C,0x6A,0x2C,0x34,0x65,0x02}, "AF Zoom-Nikkor 28-105mm f/3.5-4.5D IF"},
{[8]uint8{0x60,0x40,0x3C,0x60,0x2C,0x3C,0x66,0x02}, "AF Zoom-Nikkor 28-80mm f/3.5-5.6D"},
{[8]uint8{0x61,0x44,0x5E,0x86,0x34,0x3C,0x67,0x02}, "AF Zoom-Nikkor 75-240mm f/4.5-5.6D"},
{[8]uint8{0x63,0x48,0x2B,0x44,0x24,0x24,0x68,0x02}, "AF-S Nikkor 17-35mm f/2.8D IF-ED"},
{[8]uint8{0x64,0x00,0x62,0x62,0x24,0x24,0x6A,0x02}, "PC Micro-Nikkor 85mm f/2.8D"},
{[8]uint8{0x65,0x44,0x60,0x98,0x34,0x3C,0x6B,0x0A}, "AF VR Zoom-Nikkor 80-400mm f/4.5-5.6D ED"},
{[8]uint8{0x66,0x40,0x2D,0x44,0x2C,0x34,0x6C,0x02}, "AF Zoom-Nikkor 18-35mm f/3.5-4.5D IF-ED"},
{[8]uint8{0x67,0x48,0x37,0x62,0x24,0x30,0x6D,0x02}, "AF Zoom-Nikkor 24-85mm f/2.8-4D IF"},
{[8]uint8{0x68,0x42,0x3C,0x60,0x2A,0x3C,0x6E,0x06}, "AF Zoom-Nikkor 28-80mm f/3.3-5.6G"},
{[8]uint8{0x69,0x48,0x5C,0x8E,0x30,0x3C,0x6F,0x06}, "AF Zoom-Nikkor 70-300mm f/4-5.6G"},
{[8]uint8{0x6A,0x48,0x8E,0x8E,0x30,0x30,0x70,0x02}, "AF-S Nikkor 300mm f/4D IF-ED"},
{[8]uint8{0x6B,0x48,0x24,0x24,0x24,0x24,0x71,0x02}, "AF Nikkor ED 14mm f/2.8D"},
{[8]uint8{0x6D,0x48,0x8E,0x8E,0x24,0x24,0x73,0x02}, "AF-S Nikkor 300mm f/2.8D IF-ED II"},
{[8]uint8{0x6E,0x48,0x98,0x98,0x24,0x24,0x74,0x02}, "AF-S Nikkor 400mm f/2.8D IF-ED II"},
{[8]uint8{0x6F,0x3C,0xA0,0xA0,0x30,0x30,0x75,0x02}, "AF-S Nikkor 500mm f/4D IF-ED II"},
{[8]uint8{0x70,0x3C,0xA6,0xA6,0x30,0x30,0x76,0x02}, "AF-S Nikkor 600mm f/4D IF-ED II"},
{[8]uint8{0x72,0x48,0x4C,0x4C,0x24,0x24,0x77,0x00}, "Nikkor 45mm f/2.8 P"},
{[8]uint8{0x74,0x40,0x37,0x62,0x2C,0x34,0x78,0x06}, "AF-S Zoom-Nikkor 24-85mm f/3.5-4.5G IF-ED"},
{[8]uint8{0x75,0x40,0x3C,0x68,0x2C,0x3C,0x79,0x06}, "AF Zoom-Nikkor 28-100mm f/3.5-5.6G"},
{[8]uint8{0x76,0x58,0x50,0x50,0x14,0x14,0x7A,0x02}, "AF Nikkor 50mm f/1.8D"},
{[8]uint8{0x77,0x48,0x5C,0x80,0x24,0x24,0x7B,0x0E}, "AF-S VR Zoom-Nikkor 70-200mm f/2.8G IF-ED"},
{[8]uint8{0x78,0x40,0x37,0x6E,0x2C,0x3C,0x7C,0x0E}, "AF-S VR Zoom-Nikkor 24-120mm f/3.5-5.6G IF-ED"},
{[8]uint8{0x79,0x40,0x3C,0x80,0x2C,0x3C,0x7F,0x06}, "AF Zoom-Nikkor 28-200mm f/3.5-5.6G IF-ED"},
{[8]uint8{0x7A,0x3C,0x1F,0x37,0x30,0x30,0x7E,0x06}, "AF-S DX Zoom-Nikkor 12-24mm f/4G IF-ED"},
{[8]uint8{0x7B,0x48,0x80,0x98,0x30,0x30,0x80,0x0E}, "AF-S VR Zoom-Nikkor 200-400mm f/4G IF-ED"},
{[8]uint8{0x7D,0x48,0x2B,0x53,0x24,0x24,0x82,0x06}, "AF-S DX Zoom-Nikkor 17-55mm f/2.8G IF-ED"},
{[8]uint8{0x7F,0x40,0x2D,0x5C,0x2C,0x34,0x84,0x06}, "AF-S DX Zoom-Nikkor 18-70mm f/3.5-4.5G IF-ED"},
{[8]uint8{0x80,0x48,0x1A,0x1A,0x24,0x24,0x85,0x06}, "AF DX Fisheye-Nikkor 10.5mm f/2.8G ED"},
{[8]uint8{0x81,0x54,0x80,0x80,0x18,0x18,0x86,0x0E}, "AF-S VR Nikkor 200mm f/2G IF-ED"},
{[8]uint8{0x82,0x48,0x8E,0x8E,0x24,0x24,0x87,0x0E}, "AF-S VR Nikkor 300mm f/2.8G IF-ED"},
{[8]uint8{0x83,0x00,0xB0,0xB0,0x5A,0x5A,0x88,0x04}, "FSA-L2, EDG 65, 800mm F13 G"},
{[8]uint8{0x89,0x3C,0x53,0x80,0x30,0x3C,0x8B,0x06}, "AF-S DX Zoom-Nikkor 55-200mm f/4-5.6G ED"},
{[8]uint8{0x8A,0x54,0x6A,0x6A,0x24,0x24,0x8C,0x0E}, "AF-S VR Micro-Nikkor 105mm f/2.8G IF-ED"},
{[8]uint8{0x8B,0x40,0x2D,0x80,0x2C,0x3C,0x8D,0x0E}, "AF-S DX VR Zoom-Nikkor 18-200mm f/3.5-5.6G IF-ED"},
{[8]uint8{0x8B,0x40,0x2D,0x80,0x2C,0x3C,0xFD,0x0E}, "AF-S DX VR Zoom-Nikkor 18-200mm f/3.5-5.6G IF-ED [II]"},
{[8]uint8{0x8C,0x40,0x2D,0x53,0x2C,0x3C,0x8E,0x06}, "AF-S DX Zoom-Nikkor 18-55mm f/3.5-5.6G ED"},
{[8]uint8{0x8D,0x44,0x5C,0x8E,0x34,0x3C,0x8F,0x0E}, "AF-S VR Zoom-Nikkor 70-300mm f/4.5-5.6G IF-ED"},
{[8]uint8{0x8F,0x40,0x2D,0x72,0x2C,0x3C,0x91,0x06}, "AF-S DX Zoom-Nikkor 18-135mm f/3.5-5.6G IF-ED"},
{[8]uint8{0x90,0x3B,0x53,0x80,0x30,0x3C,0x92,0x0E}, "AF-S DX VR Zoom-Nikkor 55-200mm f/4-5.6G IF-ED"},
{[8]uint8{0x92,0x48,0x24,0x37,0x24,0x24,0x94,0x06}, "AF-S Zoom-Nikkor 14-24mm f/2.8G ED"},
{[8]uint8{0x93,0x48,0x37,0x5C,0x24,0x24,0x95,0x06}, "AF-S Zoom-Nikkor 24-70mm f/2.8G ED"},
{[8]uint8{0x94,0x40,0x2D,0x53,0x2C,0x3C,0x96,0x06}, "AF-S DX Zoom-Nikkor 18-55mm f/3.5-5.6G ED II"},
{[8]uint8{0x95,0x4C,0x37,0x37,0x2C,0x2C,0x97,0x02}, "PC-E Nikkor 24mm f/3.5D ED"},
{[8]uint8{0x95,0x00,0x37,0x37,0x2C,0x2C,0x97,0x06}, "PC-E Nikkor 24mm f/3.5D ED"},
{[8]uint8{0x96,0x48,0x98,0x98,0x24,0x24,0x98,0x0E}, "AF-S VR Nikkor 400mm f/2.8G ED"},
{[8]uint8{0x97,0x3C,0xA0,0xA0,0x30,0x30,0x99,0x0E}, "AF-S VR Nikkor 500mm f/4G ED"},
{[8]uint8{0x98,0x3C,0xA6,0xA6,0x30,0x30,0x9A,0x0E}, "AF-S VR Nikkor 600mm f/4G ED"},
{[8]uint8{0x99,0x40,0x29,0x62,0x2C,0x3C,0x9B,0x0E}, "AF-S DX VR Zoom-Nikkor 16-85mm f/3.5-5.6G ED"},
{[8]uint8{0x9A,0x40,0x2D,0x53,0x2C,0x3C,0x9C,0x0E}, "AF-S DX VR Zoom-Nikkor 18-55mm f/3.5-5.6G"},
{[8]uint8{0x9B,0x54,0x4C,0x4C,0x24,0x24,0x9D,0x02}, "PC-E Micro Nikkor 45mm f/2.8D ED"},
{[8]uint8{0x9B,0x00,0x4C,0x4C,0x24,0x24,0x9D,0x06}, "PC-E Micro Nikkor 45mm f/2.8D ED"},
{[8]uint8{0x9C,0x54,0x56,0x56,0x24,0x24,0x9E,0x06}, "AF-S Micro Nikkor 60mm f/2.8G ED"},
{[8]uint8{0x9D,0x54,0x62,0x62,0x24,0x24,0x9F,0x02}, "PC-E Micro Nikkor 85mm f/2.8D"},
{[8]uint8{0x9D,0x00,0x62,0x62,0x24,0x24,0x9F,0x06}, "PC-E Micro Nikkor 85mm f/2.8D"},
{[8]uint8{0x9E,0x40,0x2D,0x6A,0x2C,0x3C,0xA0,0x0E}, "AF-S DX VR Zoom-Nikkor 18-105mm f/3.5-5.6G ED"},
{[8]uint8{0x9F,0x58,0x44,0x44,0x14,0x14,0xA1,0x06}, "AF-S DX Nikkor 35mm f/1.8G"},
{[8]uint8{0xA0,0x54,0x50,0x50,0x0C,0x0C,0xA2,0x06}, "AF-S Nikkor 50mm f/1.4G"},
{[8]uint8{0xA1,0x40,0x18,0x37,0x2C,0x34,0xA3,0x06}, "AF-S DX Nikkor 10-24mm f/3.5-4.5G ED"},
{[8]uint8{0xA1,0x40,0x2D,0x53,0x2C,0x3C,0xCB,0x86}, "AF-P DX Nikkor 18-55mm f/3.5-5.6G"},
{[8]uint8{0xA2,0x48,0x5C,0x80,0x24,0x24,0xA4,0x0E}, "AF-S Nikkor 70-200mm f/2.8G ED VR II"},
{[8]uint8{0xA3,0x3C,0x29,0x44,0x30,0x30,0xA5,0x0E}, "AF-S Nikkor 16-35mm f/4G ED VR"},
{[8]uint8{0xA4,0x54,0x37,0x37,0x0C,0x0C,0xA6,0x06}, "AF-S Nikkor 24mm f/1.4G ED"},
{[8]uint8{0xA5,0x40,0x3C,0x8E,0x2C,0x3C,0xA7,0x0E}, "AF-S Nikkor 28-300mm f/3.5-5.6G ED VR"},
{[8]uint8{0xA6,0x48,0x8E,0x8E,0x24,0x24,0xA8,0x0E}, "AF-S Nikkor 300mm f/2.8G IF-ED VR II"},
{[8]uint8{0xA7,0x4B,0x62,0x62,0x2C,0x2C,0xA9,0x0E}, "AF-S DX Micro Nikkor 85mm f/3.5G ED VR"},
{[8]uint8{0xA8,0x48,0x80,0x98,0x30,0x30,0xAA,0x0E}, "AF-S Zoom-Nikkor 200-400mm f/4G IF-ED VR II"},
{[8]uint8{0xA9,0x54,0x80,0x80,0x18,0x18,0xAB,0x0E}, "AF-S Nikkor 200mm f/2G ED VR II"},
{[8]uint8{0xAA,0x3C,0x37,0x6E,0x30,0x30,0xAC,0x0E}, "AF-S Nikkor 24-120mm f/4G ED VR"},
{[8]uint8{0xAC,0x38,0x53,0x8E,0x34,0x3C,0xAE,0x0E}, "AF-S DX Nikkor 55-300mm f/4.5-5.6G ED VR"},
{[8]uint8{0xAD,0x3C,0x2D,0x8E,0x2C,0x3C,0xAF,0x0E}, "AF-S DX Nikkor 18-300mm f/3.5-5.6G ED VR"},
{[8]uint8{0xAE,0x54,0x62,0x62,0x0C,0x0C,0xB0,0x06}, "AF-S Nikkor 85mm f/1.4G"},
{[8]uint8{0xAF,0x54,0x44,0x44,0x0C,0x0C,0xB1,0x06}, "AF-S Nikkor 35mm f/1.4G"},
{[8]uint8{0xB0,0x4C,0x50,0x50,0x14,0x14,0xB2,0x06}, "AF-S Nikkor 50mm f/1.8G"},
{[8]uint8{0xB1,0x48,0x48,0x48,0x24,0x24,0xB3,0x06}, "AF-S DX Micro Nikkor 40mm f/2.8G"},
{[8]uint8{0xB2,0x48,0x5C,0x80,0x30,0x30,0xB4,0x0E}, "AF-S Nikkor 70-200mm f/4G ED VR"},
{[8]uint8{0xB3,0x4C,0x62,0x62,0x14,0x14,0xB5,0x06}, "AF-S Nikkor 85mm f/1.8G"},
{[8]uint8{0xB4,0x40,0x37,0x62,0x2C,0x34,0xB6,0x0E}, "AF-S Zoom-Nikkor 24-85mm f/3.5-4.5G IF-ED VR"},
{[8]uint8{0xB5,0x4C,0x3C,0x3C,0x14,0x14,0xB7,0x06}, "AF-S Nikkor 28mm f/1.8G"},
{[8]uint8{0xB6,0x3C,0xB0,0xB0,0x3C,0x3C,0xB8,0x0E}, "AF-S VR Nikkor 800mm f/5.6E FL ED"},
{[8]uint8{0xB6,0x3C,0xB0,0xB0,0x3C,0x3C,0xB8,0x4E}, "AF-S VR Nikkor 800mm f/5.6E FL ED"},
{[8]uint8{0xB7,0x44,0x60,0x98,0x34,0x3C,0xB9,0x0E}, "AF-S Nikkor 80-400mm f/4.5-5.6G ED VR"},
{[8]uint8{0xB8,0x40,0x2D,0x44,0x2C,0x34,0xBA,0x06}, "AF-S Nikkor 18-35mm f/3.5-4.5G ED"},
{[8]uint8{0xA0,0x40,0x2D,0x74,0x2C,0x3C,0xBB,0x0E}, "AF-S DX Nikkor 18-140mm f/3.5-5.6G ED VR"},
{[8]uint8{0xA1,0x54,0x55,0x55,0x0C,0x0C,0xBC,0x06}, "AF-S Nikkor 58mm f/1.4G"},
{[8]uint8{0xA1,0x48,0x6E,0x8E,0x24,0x24,0xDB,0x4E}, "AF-S Nikkor 120-300mm f/2.8E FL ED SR VR"},
{[8]uint8{0xA2,0x40,0x2D,0x53,0x2C,0x3C,0xBD,0x0E}, "AF-S DX Nikkor 18-55mm f/3.5-5.6G VR II"},
{[8]uint8{0xA4,0x40,0x2D,0x8E,0x2C,0x40,0xBF,0x0E}, "AF-S DX Nikkor 18-300mm f/3.5-6.3G ED VR"},
{[8]uint8{0xA5,0x4C,0x44,0x44,0x14,0x14,0xC0,0x06}, "AF-S Nikkor 35mm f/1.8G ED"},
{[8]uint8{0xA6,0x48,0x98,0x98,0x24,0x24,0xC1,0x0E}, "AF-S Nikkor 400mm f/2.8E FL ED VR"},
{[8]uint8{0xA7,0x3C,0x53,0x80,0x30,0x3C,0xC2,0x0E}, "AF-S DX Nikkor 55-200mm f/4-5.6G ED VR II"},
{[8]uint8{0xA8,0x48,0x8E,0x8E,0x30,0x30,0xC3,0x4E}, "AF-S Nikkor 300mm f/4E PF ED VR"},
{[8]uint8{0xA8,0x48,0x8E,0x8E,0x30,0x30,0xC3,0x0E}, "AF-S Nikkor 300mm f/4E PF ED VR"},
{[8]uint8{0xA9,0x4C,0x31,0x31,0x14,0x14,0xC4,0x06}, "AF-S Nikkor 20mm f/1.8G ED"},
{[8]uint8{0xAA,0x48,0x37,0x5C,0x24,0x24,0xC5,0x4E}, "AF-S Nikkor 24-70mm f/2.8E ED VR"},
{[8]uint8{0xAA,0x48,0x37,0x5C,0x24,0x24,0xC5,0x0E}, "AF-S Nikkor 24-70mm f/2.8E ED VR"},
{[8]uint8{0xAB,0x3C,0xA0,0xA0,0x30,0x30,0xC6,0x4E}, "AF-S Nikkor 500mm f/4E FL ED VR"},
{[8]uint8{0xAC,0x3C,0xA6,0xA6,0x30,0x30,0xC7,0x4E}, "AF-S Nikkor 600mm f/4E FL ED VR"},
{[8]uint8{0xAD,0x48,0x28,0x60,0x24,0x30,0xC8,0x4E}, "AF-S DX Nikkor 16-80mm f/2.8-4E ED VR"},
{[8]uint8{0xAD,0x48,0x28,0x60,0x24,0x30,0xC8,0x0E}, "AF-S DX Nikkor 16-80mm f/2.8-4E ED VR"},
{[8]uint8{0xAE,0x3C,0x80,0xA0,0x3C,0x3C,0xC9,0x4E}, "AF-S Nikkor 200-500mm f/5.6E ED VR"},
{[8]uint8{0xAE,0x3C,0x80,0xA0,0x3C,0x3C,0xC9,0x0E}, "AF-S Nikkor 200-500mm f/5.6E ED VR"},
{[8]uint8{0xA0,0x40,0x2D,0x53,0x2C,0x3C,0xCA,0x8E}, "AF-P DX Nikkor 18-55mm f/3.5-5.6G"},
{[8]uint8{0xA0,0x40,0x2D,0x53,0x2C,0x3C,0xCA,0x0E}, "AF-P DX Nikkor 18-55mm f/3.5-5.6G VR"},
{[8]uint8{0xAF,0x4C,0x37,0x37,0x14,0x14,0xCC,0x06}, "AF-S Nikkor 24mm f/1.8G ED"},
{[8]uint8{0xA2,0x38,0x5C,0x8E,0x34,0x40,0xCD,0x86}, "AF-P DX Nikkor 70-300mm f/4.5-6.3G VR"},
{[8]uint8{0xA3,0x38,0x5C,0x8E,0x34,0x40,0xCE,0x8E}, "AF-P DX Nikkor 70-300mm f/4.5-6.3G ED VR"},
{[8]uint8{0xA3,0x38,0x5C,0x8E,0x34,0x40,0xCE,0x0E}, "AF-P DX Nikkor 70-300mm f/4.5-6.3G ED"},
{[8]uint8{0xA4,0x48,0x5C,0x80,0x24,0x24,0xCF,0x4E}, "AF-S Nikkor 70-200mm f/2.8E FL ED VR"},
{[8]uint8{0xA4,0x48,0x5C,0x80,0x24,0x24,0xCF,0x0E}, "AF-S Nikkor 70-200mm f/2.8E FL ED VR"},
{[8]uint8{0xA5,0x54,0x6A,0x6A,0x0C,0x0C,0xD0,0x46}, "AF-S Nikkor 105mm f/1.4E ED"},
{[8]uint8{0xA5,0x54,0x6A,0x6A,0x0C,0x0C,0xD0,0x06}, "AF-S Nikkor 105mm f/1.4E ED"},
{[8]uint8{0xA6,0x48,0x2F,0x2F,0x30,0x30,0xD1,0x46}, "PC Nikkor 19mm f/4E ED"},
{[8]uint8{0xA6,0x48,0x2F,0x2F,0x30,0x30,0xD1,0x06}, "PC Nikkor 19mm f/4E ED"},
{[8]uint8{0xA7,0x40,0x11,0x26,0x2C,0x34,0xD2,0x46}, "AF-S Fisheye Nikkor 8-15mm f/3.5-4.5E ED"},
{[8]uint8{0xA7,0x40,0x11,0x26,0x2C,0x34,0xD2,0x06}, "AF-S Fisheye Nikkor 8-15mm f/3.5-4.5E ED"},
{[8]uint8{0xA8,0x38,0x18,0x30,0x34,0x3C,0xD3,0x8E}, "AF-P DX Nikkor 10-20mm f/4.5-5.6G VR"},
{[8]uint8{0xA8,0x38,0x18,0x30,0x34,0x3C,0xD3,0x0E}, "AF-P DX Nikkor 10-20mm f/4.5-5.6G VR"},
{[8]uint8{0xA9,0x48,0x7C,0x98,0x30,0x30,0xD4,0x4E}, "AF-S Nikkor 180-400mm f/4E TC1.4 FL ED VR"},
{[8]uint8{0xA9,0x48,0x7C,0x98,0x30,0x30,0xD4,0x0E}, "AF-S Nikkor 180-400mm f/4E TC1.4 FL ED VR"},
{[8]uint8{0xAA,0x48,0x88,0xA4,0x3C,0x3C,0xD5,0x4E}, "AF-S Nikkor 180-400mm f/4E TC1.4 FL ED VR + 1.4x TC"},
{[8]uint8{0xAA,0x48,0x88,0xA4,0x3C,0x3C,0xD5,0x0E}, "AF-S Nikkor 180-400mm f/4E TC1.4 FL ED VR + 1.4x TC"},
{[8]uint8{0xAB,0x44,0x5C,0x8E,0x34,0x3C,0xD6,0xCE}, "AF-P Nikkor 70-300mm f/4.5-5.6E ED VR"},
{[8]uint8{0xAB,0x44,0x5C,0x8E,0x34,0x3C,0xD6,0x0E}, "AF-P Nikkor 70-300mm f/4.5-5.6E ED VR"},
{[8]uint8{0xAB,0x44,0x5C,0x8E,0x34,0x3C,0xD6,0x4E}, "AF-P Nikkor 70-300mm f/4.5-5.6E ED VR"},
{[8]uint8{0xAC,0x54,0x3C,0x3C,0x0C,0x0C,0xD7,0x46}, "AF-S Nikkor 28mm f/1.4E ED"},
{[8]uint8{0xAC,0x54,0x3C,0x3C,0x0C,0x0C,0xD7,0x06}, "AF-S Nikkor 28mm f/1.4E ED"},
{[8]uint8{0xAD,0x3C,0xA0,0xA0,0x3C,0x3C,0xD8,0x0E}, "AF-S Nikkor 500mm f/5.6E PF ED VR"},
{[8]uint8{0xAD,0x3C,0xA0,0xA0,0x3C,0x3C,0xD8,0x4E}, "AF-S Nikkor 500mm f/5.6E PF ED VR"} }
