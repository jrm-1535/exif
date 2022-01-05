package exif

// support for Apple Maker notes

import (
    "fmt"
    "bytes"
    "math"
    "encoding/binary"
    "io"
)

const (             // Apple Maker note tags
    _Apple001                   = 0x0001  // should be _SignedLong
    _Apple002                   = 0x0002  // should be _Undefined, actually _UnsignedLong offset to a pList
    _AppleRunTime               = 0x0003  // should be _Undefined, actually _UnsignedLong offset to runtime pList
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
    _AppleOrientation           = 0x000e  // 1 _SignedLong Orientation? 0=landscape? 4=portrait?
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
)

/* pList types are mapped to go types as follow:
    - null is nil
    - bool is bool
    - fill is 
    - int is uint64
    - real is ???
    - date is byte array[8]
    - data is byte array[n] (raw data)
    - string is string  (See if Unicode needs extra processing)
    - uid is byte array[n]
    - array is *pNode array
    - set is *pNode array
    - dict is a map[string]*pNode (assuming keys are always strings)
*/

type pNode struct {
    marker  byte        // the object code & length (for serialization)
    value   interface{}
}
/* was used intially for debugging
func (pn *pNode)print( indent string ) {
    switch o := pn.value.(type) {
    case nil:
        fmt.Printf( "%snull\n")
    case bool:
        fmt.Printf( "%s%v\n", indent, o )
    case byte, uint64:
        fmt.Printf( "%s%d\n", indent, o )
    case string:
        fmt.Printf( "%s%s\n", indent, o )
    case []byte:
        dumpData( os.Stdout, indent+"data", indent+"  ", o )
    case map[string]*pNode:
        for k, v := range o {
            fmt.Printf( "%s%s", indent, k )
            v.print( " : " )
        }
    default:
        panic("Not supported (yet)\n")
    }
}
*/
func getPlist( pList []byte ) ( *pNode, error ) {
    if ! bytes.Equal( pList[:8], []byte("bplist00") ) {
        return nil, fmt.Errorf( "getPList: not an Apple plist (%s)\n", string(pList[:8]) )
    }

    // get trailer info first
    trailer := len( pList ) - 32
    if trailer < 8 {
        return nil, fmt.Errorf( "getPList: wrong size for an Apple plist (%d)\n", len( pList) )
    }

    getbeOffset := func( o, s uint64 ) uint64 {
    //    fmt.Printf( "getbeOffset: offset %d, size %d\n", o, s )
        switch( s ) {
        case 1: return uint64(pList[o])
        case 2: return uint64(binary.BigEndian.Uint16(pList[o:]))
        case 4: return uint64(binary.BigEndian.Uint32(pList[o:]))
        case 8: return uint64(binary.BigEndian.Uint64(pList[o:]))
        default:
            panic(fmt.Sprintf("invalid offsetSize %d\n", s))
        }
    }

    // Skip 5 unused bytes + 1-byte _sortVersion
    offsetEntrySize := uint64(pList[trailer+6]) // 1-byte _offsetIntSize
//    TODO: used in arrays, sets and dictionaries only (TBI)
    objectRefSize := uint64(pList[trailer+7])   // 1-byte _objectRefSize
    // 8-byte _numObjects
    //numObjects := getbeOffset( uint64(trailer+8), 8 )
    // 8-byte _topObject
    topObjectOffset := getbeOffset( uint64(trailer+16), 8 )
    // 8-byte _offsetTableOffset
    offsetTableOffset := getbeOffset( uint64(trailer+24), 8 )
/*
    fmt.Printf( "offsetEntrySize: %d bytes\n", offsetEntrySize )
    fmt.Printf( "objectRefSize: %d bytes\n", objectRefSize )
    fmt.Printf( "numObjects: %d\n", numObjects )
    fmt.Printf( "topObjectOffset: %d\n", topObjectOffset )
    fmt.Printf( "offsetTableOffset: %d\n", offsetTableOffset )
*/
    checkSize := func( s uint64 ) error {
        switch s {
        case 1, 2, 4, 8:
            return nil
        default:
            return fmt.Errorf( "getPlist: invalid offsetSize %d\n", offsetEntrySize )
        }
    }

    if err := checkSize( offsetEntrySize ); err != nil {
        return nil, err
    }
    if err := checkSize( objectRefSize ); err != nil {
        return nil, err
    }

    getOffsetTableEntry := func( o uint64 ) (uint64, error) {
        o += offsetTableOffset
        if o > uint64(trailer) {
            return 0, fmt.Errorf("getPlist: Invalid offsetTable Entry @%#04x\n", o)
        }
        return getbeOffset( o, offsetEntrySize ), nil
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
        return offset + 1, size                 // move to the fist data byte
    }

    var getObject func ( object uint64 ) (*pNode, error)
    getObject = func ( object uint64 ) (*pNode, error) {

        pn := new(pNode)
        pn.marker = pList[object]          // set the marker code

        switch pList[object] & 0xf0 {
        case 0x00:          // special
            switch pList[object] & 0x0f {
            case 0x00:                      // null => nil
            case 0x01:  pn.value = true     // boolean true
            case 0x08:  pn.value = false    // boolean false
            case 0x0f:                      // treat fill as null?
            default:
                return nil, fmt.Errorf("getPlist: invalid marker byte %#02x\n", pList[object])
            }

        case 0x10:          // int, less significant 4 bits are exponent of following size
            size := 1 << (pList[object] & 0x0f)
            object ++
            switch size {
            case 1: pn.value = uint64(pList[object])
            case 2: pn.value = uint64(binary.BigEndian.Uint16(pList[object:]))
            case 4: pn.value = uint64(binary.BigEndian.Uint32(pList[object:]))
            case 8: pn.value = binary.BigEndian.Uint64(pList[object:])
            default:
                return nil, fmt.Errorf("getPlist: invalid marker size %#02x\n", size)
            }

        case 0x20:          // real, less significant 4 bits are exponent of following size
            size := 1 << (pList[object] & 0x0f)
            object ++
            switch( size ) {
            case 4:
                pn.value = float64(math.Float32frombits(binary.BigEndian.Uint32(pList[object:])))
            case 8:         // IEEE 754 Double precision 64-bit format
                pn.value = math.Float64frombits(binary.BigEndian.Uint64(pList[object:]))
            default:
                return nil, fmt.Errorf("getPlist: unsupported real size %#02x\n", size)
            }

        case 0x30:          // should be date as 8-byte float (EEE 754 Double precision)
            if pList[object] != 0x33 {
                return nil, fmt.Errorf( "getPlist: invalid marker byte %#02x\n", pList[object] )
            }
            object ++
    		pn.value = math.Float64frombits(binary.BigEndian.Uint64(pList[object:]))

        case 0x40:          // raw data byte array
            start, size := getOSize( pList, object ); if size == 0 {
                return nil, fmt.Errorf( "getPlist: invalid data size encoding\n" )
            }
            pn.value = pList[start:start+size]

        case 0x50, 0x60:    // ASCII string or Unicode string
            start, count := getOSize( pList, object ); if count == 0 {
                return nil, fmt.Errorf( "getPlist: invalid data size encoding\n" )
            }
            pn.value = string(pList[start:start+count])

        case 0x80:          // uid
            size := 1 + (pList[object] & 0x0f)
            pn.value = pList[object+1:object+1+uint64(size)]

        case 0xa0, 0xc0:    // Array & set
            start, count := getOSize( pList, object ); if count == 0 {
                return nil, fmt.Errorf( "getPList: invalid array size encoding\n" )
            }
            pn.value = make( []*pNode, count )
            for j := uint64(0); j < count; j ++ {
                var err error
                var off uint64
                off, err = getOffsetTableEntry(
                               getbeOffset(start+(j*objectRefSize), objectRefSize))
                var v *pNode
                v, err = getObject( off ); if err != nil {
                    return nil, err
                }
                pn.value.([]*pNode)[j] = v
            }

        case 0xd0:          // dict
            start, count := getOSize( pList, object )
            dist := count * objectRefSize
            pn.value = make(map[string]*pNode)
//            fmt.Printf( "%sDict (%d entries) @offset%d\n", indent, count, start )
            for j := uint64(0); j < dist; j += uint64(objectRefSize) {
                var err error
                var off uint64
                off, err = getOffsetTableEntry(getbeOffset(start+j, objectRefSize))
                var k, v *pNode
                k, err = getObject( off ); if err != nil {
                    return nil, err
                }
                key := k.value.(string)
                off, err = getOffsetTableEntry(getbeOffset(start+j+dist, objectRefSize))
                v, err = getObject( off ); if err != nil {
                    return nil, err
                }
                pn.value.(map[string]*pNode)[key] = v
            }

        default:
            return nil, fmt.Errorf( "getPList: invalid object marker (%#02x)\n", pList[object] )
        }
        return pn, nil
    }

    topObjectStart, err := getOffsetTableEntry( topObjectOffset )
    if err != nil {
        return nil, err
    }
//    fmt.Printf( "pList: %d object(s) - top level object starts at offset %#04x in plist\n",
//                numObjects, topObjectStart )

    return getObject( topObjectStart )
}

func (ifd *ifdd) storeApplePLIST( name string, p func( io.Writer, interface{} ) ) error {

    if ifd.fType != _Undefined {
        return fmt.Errorf( "%s (PList): invalid type (%s)\n", name, getTiffTString( ifd.fType ) )
    }
    pList := ifd.getUnsignedBytes( )
    ifd.storeValue( ifd.newUnsignedByteValue( name, p, pList ) )
    return nil
}

func dumpPlist( w io.Writer, v interface{} ) {
    data := v.([]byte)
    dumpData( w, "plist", "     ", data )
}

func printRuntime( w io.Writer, v interface{} ) {
    pList := v.([]byte)
    root, err := getPlist( pList ); if err != nil {
        fmt.Fprintf( w, "Invalid runtime (not a plist)\n" )
        return
    }
    o, ok := root.value.(map[string]*pNode); if !ok {
        fmt.Fprintf( w, "Invalid runtime (not a dictionary)\n" )
        return
    }
/*
    This represents a CMTime structure giving the amount of time the phone has
    been running since the last boot, not including standby time.

    value     runtime in ns to divide by timescale to get seconds
    timescale in ns
    epoch     0 ?
    flags:    0 Valid
              1 Has been rounded
              2 Positive infinity
              3 Negative infinity
              4 Indefinite
    See
https://developer.apple.com/library/ios/documentation/CoreMedia/Reference/CMTime/Reference/reference.html)
*/
    var fs string
    var value float64
    var scale float64
    var epoch uint64

    for k, v := range o {
        switch k {
        case "flags":
            flag, ok := v.value.(uint64); if ! ok {
                fmt.Fprintf( w, "        Invalid flag (not an int)\n" )
            }
            switch flag {
            case 0: fs = "is Valid"
            case 1: fs = "has been rounded"
            case 2: fs = "is positive infinity"
            case 3: fs = "is negative infinity"
            case 4: fs = "is indefinite"
            }

        case "value":
            val, ok := v.value.(uint64); if ! ok {
                fmt.Fprintf( w, "        Invalid flag (not an int)\n" )
            }
            value = float64(val)

        case "epoch":
            e, ok := v.value.(uint64); if ! ok {
                fmt.Fprintf( w, "        Invalid epoch (not an int)\n" )
            }
            epoch = e

        case "timescale":
            ts, ok := v.value.(uint64); if ! ok {
                fmt.Fprintf( w, "      Invalid timescale (not an int)\n" )
            }
            scale = float64(ts)
        }
    }

    if fs != "" && value != 0.0 && scale != 0.0 {
        fmt.Fprintf( w, "\n     Value : %f seconds (epoch %d) - %s\n", value/scale, epoch, fs )
    }
}

func (ifd *ifdd) storeAppleAccelerationVector( ) error {
    v, err := ifd.checkSignedRationals( 3 )
    if err == nil {
        p := func( w io.Writer,v interface{} ) {
            av := v.([]signedRational)
    /*
        AccelerationVector
        XYZ coordinates of the acceleration vector in units of g.  As viewed from
        the front of the phone, positive X is toward the left side, positive Y is
        toward the bottom, and positive Z points into the face of the phone

        See
    http://nscookbook.com/2013/03/ios-programming-recipe-19-using-core-motion-to-access-gyro-and-accelerometer/
    */
            fmt.Fprintf( w, "\n     Vector X: %f (%d/%d)\n",
                        float32(av[0].Numerator)/float32(av[0].Denominator),
                        av[0].Numerator, av[0].Denominator )
            fmt.Fprintf( w, "     Vector Y: %f (%d/%d)\n",
                        float32(av[1].Numerator)/float32(av[1].Denominator),
                        av[1].Numerator, av[1].Denominator )
            fmt.Fprintf( w, "     Vector Z: %f (%d/%d)\n",
                        float32(av[2].Numerator)/float32(av[2].Denominator),
                        av[2].Numerator, av[2].Denominator )
        }
        ifd.storeValue( ifd.newSignedRationalValue( "Acceleration Vectors", p, v ) )
    }
    return err
}

func (ifd *ifdd) storeAppleImageType( ) error {
//          = 0x000a  // 1 _SignedLong: 2=iPad mini 2, 3=HDR Image, 4=Original Image
    var fait = func ( w io.Writer, v interface{} ) {
        it := v.(int32)
        var s string
        switch it {
        case 2: s = "iPad mini 2"
        case 3: s = "HDR Image"
        case 4: s = "Original Image"
        default: s = "Unknown Image Type"
        }
        fmt.Fprintf( w, "%s\n", s )
    }
    return ifd.storeSignedLongs( "  Apple Image Type", 1, fait )
}

func (ifd *ifdd) storeAppleOrientation( ) error {
// 1 _SignedLong Orientation? 0=landscape? 4=portrait?
    var fao = func( w io.Writer, v interface{} ) {
        o := v.(int32)
        var s string
        switch v {
        case 0: s = "Landscape"
        case 4: s = "portrait"
        default: s = "?"
        }
        fmt.Fprintf( w, "%s (%d)\n", s, o )
    }
    return ifd.storeSignedLongs( "  Apple Image orientation", 1, fao )
}

func storeAppleTags( ifd *ifdd ) error {
//    fmt.Printf( "storeAppleTags: tag (%#04x) @offset %#04x type %s count %d\n",
//                 ifd.fTag, ifd.sOffset-8, getTiffTString( ifd.fType ), ifd.fCount )

    switch ifd.fTag {
    case _Apple001:
        return ifd.storeSignedLongs( "Apple #0001", 1, nil )
    case _Apple002:
        return ifd.storeApplePLIST( "Apple #0002", dumpPlist )
    case _AppleRunTime:
        return ifd.storeApplePLIST( "Apple RunTime", printRuntime )
    case _Apple004:
        return ifd.storeSignedLongs( "Apple #0004", 1, nil )
    case _Apple005:
        return ifd.storeSignedLongs( "Apple #0005", 1, nil )
    case _Apple006:
        return ifd.storeSignedLongs( "Apple #0006", 1, nil )
    case _Apple007:
        return ifd.storeSignedLongs( "Apple #0007", 1, nil )
    case _AppleAccelerationVector:
        return ifd.storeAppleAccelerationVector( )
    case _Apple009:
        return ifd.storeSignedLongs( "Apple #0009", 1, nil )
    case _AppleHDRImageType:
        return ifd.storeAppleImageType( )
    case _BurstUUID:
        return ifd.storeAsciiString( "Apple burst UUID" )
    case _Apple00c:
        return ifd.storeUnsignedRationals( "Apple #000c", 2, nil )
    case _Apple00d:
        return ifd.storeSignedLongs( "Apple #000d", 1, nil )
    case _AppleOrientation:
        return ifd.storeAppleOrientation( )
    case _Apple00f:
        return ifd.storeSignedLongs( "Apple #000f", 1, nil )
    case _Apple010:
        return ifd.storeSignedLongs( "Apple #0010", 1, nil )
    case _AppleMediaGroupUUID:
        return ifd.storeAsciiString( "Apple Media Group UUID" )
    case _Apple0014:
//           = 0x0014  // 1 _SignedLong 1, 2, 3, 4, 5 (iPhone 6s, iOS 6.1)
        return ifd.storeSignedLongs( "Apple Device Type", 1, nil )
    case _AppleImageUniqueID:
        return ifd.storeAsciiString( "Apple Image UUID" )
    case _Apple0016:
//            = 0x0016  // 1 _ASCIIString [29]"AXZ6pMTOh2L+acSh4Kg630XCScoO\0"
        return ifd.storeAsciiString( "Apple #0016" )
    case _Apple0017:
        return ifd.storeSignedLongs( "Apple #0017", 1, nil )
    case _Apple0019:
        return ifd.storeSignedLongs( "Apple #0019", 1, nil )
    case _Apple001a:
//             = 0x001a  // 1 _ASCIIString [6]"q825s\0"
        return ifd.storeAsciiString( "  Apple #001a" )
    case _Apple001f:
        return ifd.storeSignedLongs( "  Apple #001f", 1, nil )
    default:
        return ifd.processUnknownTag( )
    }
    return nil
}

const (
    _APPLE_MAKER_SIGNATURE = "Apple iOS\x00"
    _APPLE_MAKER_SIGNATURE_SIZE = 10
    _APPLE_MAKER_ENDIAN_OFFSET = 12
    _APPLE_MAKER_IFD_OFFSET = 14
)

func (ifd *ifdd) processAppleMakerNote( offset uint32 ) error {

    // Apple maker notes look like an IFD but within its own endianess and
    // own reference (not the same origin as the TIFF descriptor). Its starts
    // with 10-byte identifier: "Apple iOS\x00", plus 2-byte version \x0001
    // in big endian and a 2-byte endian idendifier: "MM" for big endian,
    // before mapping to a regular IFD structure: 2-byte number of entries
    // in the IFD followed by the regular IFD entries and IFD data
    endian, err := getEndianess(
                ifd.desc.data[offset + _APPLE_MAKER_ENDIAN_OFFSET:
                              offset+ifd.fCount - _APPLE_MAKER_ENDIAN_OFFSET] )
    if err != nil {
        return err
    }
    mknd := new(Desc)
    mknd.Control = ifd.desc.Control     // propagate original control
    mknd.data = ifd.desc.data[offset:offset+ifd.fCount]
    mknd.endian = endian

//    fmt.Printf( "Apple maker notes: origin %#04x start %#04x, end %#04x, endian %v\n",
//                offset, 14, offset + ifd.fCount, endian )
//    dumpData( os.Stdout, "    MakerNote", "      ", ifd.desc.data[offset:offset+ifd.fCount] )
    var apple *ifdd
    _, apple, err = mknd.storeIFD( MAKER, _APPLE_MAKER_IFD_OFFSET, storeAppleTags )
    if err != nil {
        return err
    }
//    fmt.Printf( "End Apple maker notes @offset %#08x - expected offset %#08x\n",
//                ifd.dOffset + apple.dOffset, ifd.dOffset + ifd.fCount )

    mknd.root = apple
    ifd.storeValue( ifd.newDescValue( mknd, "Apple iOS\x00\x00\x01MM",
                                      _APPLE_MAKER_IFD_OFFSET ) )
    ifd.desc.ifds[MAKER] = apple
    return err
}

func tryAppleMakerNote( ifd *ifdd, offset uint32 ) ( func( uint32 ) error ) {

    if bytes.Equal( ifd.desc.data[offset:offset+_APPLE_MAKER_SIGNATURE_SIZE],
                    []byte( _APPLE_MAKER_SIGNATURE ) ) {
//        fmt.Printf("    MakerNote: Apple iOS\n" )
        return ifd.processAppleMakerNote
    }
    return nil
}

