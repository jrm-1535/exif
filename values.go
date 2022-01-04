package exif

import (
    "fmt"
    "encoding/binary"
    "io"
    "os"
    "reflect"
)

// common IFD entry structure (offset/value are specific to each value)
type tEntry struct {
    vTag    tTag
    vType   tType
    vCount  uint32
}

/*
    In order to allow modifying/removing TIFF metadata, each IFD entry is
    stored individually as a tiffvalue.

    In input TIFF data types are converted in go types:
    _UnsignedByte       => []uint8
    _ASCIIString        => []uint8
    _UnsignedShort      => []uint16
    _UnsignedLong       => []uint32
    _UnsignedRational   => []unsignedRational struct
    _SignedByte         => []int8
    _Undefined          => transformed in actual type, from the tag value
    _SignedShort        => []int16
    _SignedLong         => []int32
    _SignedRational     => []signedRational struct
    _Float              => []float32
    _Double             => []float64

    Then those go types are saved as tiffValue:
    []uint8         -> unsignedByteValue for _UnsignedByte(s) and _ASCIIString
                    -> descValue for same Maker notes requiring their own reference
                    -> ifdValue for embedded ifd or some maker notes
    []uint16        -> unsignedShortValue for _UnsignedShort(s)
    []uint32        -> unsignedLongValue for _UnsignedLong(s)
    []unsignedRational -> unsignedRationalValue for _UnsignedRational(s)
    []int8          -> signedByteValue for _SignedByte(s)
    []int16         -> signedShortValue for _SignedShort(s)
    []int32         -> signedLongValue for _SignedLong(s)
    []signedRational -> signedRationalValue for _SignedRational(s)
*/

type unsignedRational struct {
    Numerator, Denominator  uint32  // unexported type, but exported fields ;-)
}

type signedRational struct {
    Numerator, Denominator  int32
}

// A tiffValue is defined as its entry definition followed by one of the
// above types and implementing the following interface:

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

// implement tag specific formated print of the value
    format( w io.Writer )
}

// All ifd.get<type> functions ignore the actual entry type and read <count> 
// value of their type.

func (ifd *ifdd) getUnsignedBytes( ) []uint8 {
    if ifd.fCount <= 4 {
        return ifd.desc.getUnsignedBytes( ifd.sOffset, ifd.fCount )
    }
    rOffset := ifd.desc.getUnsignedLong( ifd.sOffset )
    return ifd.desc.getUnsignedBytes( rOffset, ifd.fCount )
}

func (ifd *ifdd) getSignedBytes( ) []int8 {
    if ifd.fCount <= 4 {
        return ifd.desc.getSignedBytes( ifd.sOffset, ifd.fCount )
    }
    rOffset := ifd.desc.getUnsignedLong( ifd.sOffset )
    return ifd.desc.getSignedBytes( rOffset, ifd.fCount )
}

func (ifd *ifdd) getUnsignedShorts( ) []uint16 {
    if ifd.fCount * _ShortSize <= 4 {
        return ifd.desc.getUnsignedShorts( ifd.sOffset, ifd.fCount )
    } else {
        rOffset := ifd.desc.getUnsignedLong( ifd.sOffset )
        return ifd.desc.getUnsignedShorts( rOffset, ifd.fCount )
    }
}

func (ifd *ifdd) getSignedShorts( ) []int16 {
    if ifd.fCount * _ShortSize <= 4 {
        return ifd.desc.getSignedShorts( ifd.sOffset, ifd.fCount )
    } else {
        rOffset := ifd.desc.getUnsignedLong( ifd.sOffset )
        return ifd.desc.getSignedShorts( rOffset, ifd.fCount )
    }
}

func (ifd *ifdd) getUnsignedLongs( ) []uint32 {
    if ifd.fCount * _LongSize <= 4 {
        return ifd.desc.getUnsignedLongs( ifd.sOffset, ifd.fCount )
    } else {
        rOffset := ifd.desc.getUnsignedLong( ifd.sOffset )
        return ifd.desc.getUnsignedLongs( rOffset, ifd.fCount )
    }
}

// All ifd.check<type> functions check the entry type (and sometimes count)
// and return an error if it does not match expectations, otherwise return
// the corresponding value

func (ifd *ifdd) checkUnsignedBytes( count uint32 ) ([]uint8, error) {
    if ifd.fType != _UnsignedByte {
        return nil, fmt.Errorf( "checkUnsignedBytes: incorrect type (%s)\n",
                                getTiffTString( ifd.fType ) )
    }
    if count != 0 && count != ifd.fCount {
        return nil, fmt.Errorf( "checkUnsignedBytes: incorrect count (%d)\n",
                                ifd.fCount )
    }
    return ifd.getUnsignedBytes( ), nil
}

func (ifd *ifdd) checkSignedBytes( count uint32 ) ([]int8, error) {
    if ifd.fType != _SignedByte {
        return nil, fmt.Errorf( "checkSignedBytes: incorrect type (%s)\n",
                                getTiffTString( ifd.fType ) )
    }
    if count != 0 && count != ifd.fCount {
        return nil, fmt.Errorf( "checkSignedBytes: incorrect count (%d)\n",
                                ifd.fCount )
    }
    return ifd.getSignedBytes( ), nil
}

func (ifd *ifdd) checkTiffAsciiString( ) ([]byte, error) {
    if ifd.fType != _ASCIIString {
        return nil, fmt.Errorf( "checkTiffAsciiString: incorrect type (%s)\n",
                                getTiffTString( ifd.fType ) )
    }
    return ifd.getUnsignedBytes( ), nil
}

func (ifd *ifdd) checkUnsignedShorts( count uint32 ) ([]uint16, error) {
    if ifd.fType != _UnsignedShort {
        return nil, fmt.Errorf( "checkUnsignedShorts: incorrect type (%s)\n",
                                getTiffTString( ifd.fType ) )
    }
    if count != 0 && count != ifd.fCount {
        return nil, fmt.Errorf( "checkUnsignedShorts: incorrect count (%d)\n",
                                ifd.fCount )
    }
    return ifd.getUnsignedShorts( ), nil
}

func (ifd *ifdd) checkSignedShorts( count uint32 ) ([]int16, error) {
    if ifd.fType != _SignedShort {
        return nil, fmt.Errorf( "checkSignedShorts: incorrect type (%s)\n",
                                getTiffTString( ifd.fType ) )
    }
    if count != 0 && count != ifd.fCount {
        return nil, fmt.Errorf( "checkSignedShorts: incorrect count (%d)\n",
                                ifd.fCount )
    }
    return ifd.getSignedShorts( ), nil
}

func (ifd *ifdd) checkUnsignedLongs( count uint32 ) ([]uint32, error) {
    if ifd.fType != _UnsignedLong {
        return nil, fmt.Errorf( "checkUnsignedLongs: incorrect type (%s)\n",
                            getTiffTString( ifd.fType ) )
    }
    if count != 0 && count != ifd.fCount {
        return nil, fmt.Errorf( "checkUnsignedLongs: incorrect count (%d)\n",
                                ifd.fCount )
    }
    return ifd.desc.getUnsignedLongs( ifd.sOffset, ifd.fCount ), nil
}

func (ifd *ifdd) checkSignedLongs( count uint32 ) ([]int32, error) {
    if ifd.fType != _SignedLong {
        return nil, fmt.Errorf( "checkSignedLongs: incorrect type (%s)\n",
                            getTiffTString( ifd.fType ) )
    }
    if count != 0 && count != ifd.fCount {
        return nil, fmt.Errorf( "checkSignedLongs: incorrect count (%d)\n",
                                ifd.fCount )
    }
    return ifd.desc.getSignedLongs( ifd.sOffset, ifd.fCount ), nil
}

func (ifd *ifdd) checkUnsignedRationals( 
                                count uint32 ) ([]unsignedRational, error) {
    if ifd.fType != _UnsignedRational {
        return nil, fmt.Errorf( "checkUnsignedRational: incorrect type (%s)\n",
                                getTiffTString( ifd.fType ) )
    }
    if count != 0 && count != ifd.fCount {
        return nil, fmt.Errorf( "checkUnsignedRational: incorrect count (%d)\n",
                                ifd.fCount )
    }
    // a rational never fits directly in valOffset (requires more than 4 bytes)
    offset := ifd.desc.getUnsignedLong( ifd.sOffset )
    return ifd.desc.getUnsignedRationals( offset, ifd.fCount ), nil
}

func (ifd *ifdd) checkSignedRationals(
                                 count uint32 ) ([]signedRational, error) {
    if ifd.fType != _SignedRational {
        return nil, fmt.Errorf( "checkUnsignedRational: incorrect type (%s)\n",
                                getTiffTString( ifd.fType ) )
    }
    if count != 0 && count != ifd.fCount {
        return nil, fmt.Errorf( "checkUnsignedRational: incorrect count (%d)\n",
                                ifd.fCount )
    }
    // a rational never fits directly in valOffset (requires more than 4 bytes)
    offset := ifd.desc.getUnsignedLong( ifd.sOffset )
    return ifd.desc.getSignedRationals( offset, ifd.fCount ), nil
}

// Common value structure to embed in specific value definition
type tVal struct {
    ifd     *ifdd               // parent IFD
    fpr     func( interface{} ) // value specific print func
    name    string              // value name
            tEntry              // common entry structure
}

// TIFF Value definitions

type descValue struct {     // used for some maker notes
            tVal
    header  string
    origin  uint32
    v      *Desc
}
func (ifd *ifdd) newDescValue( dVal *Desc, header string,
                               origin uint32 ) (dv *descValue) {
    dv = new( descValue )
    dv.ifd = ifd
    dv.vTag = ifd.fTag
    dv.vType = ifd.fType
    dv.origin = origin
//  dv.vCount will be calculated when serializeEntry is called
    dv.header = header
    dv.v = dVal
    return
}

func (dv *descValue) serializeEntry( w io.Writer ) (err error) {
    sz := dv.v.root.dSize
    if sz == 0 {
        if dv.ifd.desc.srlzDbg {
            fmt.Printf( "%s ifd serializeEntry: Get %s ifd size @offset %#08x\n",
                        dv.ifd.getIfdName(), dv.v.root.getIfdName(), dv.ifd.dOffset )
        }
        _, err = dv.v.root.serializeEntries( io.Discard, 0 )
        if err != nil {
            err = fmt.Errorf( "%s ifd serializeEntry: Get %s ifd size: %v",
                              dv.ifd.getIfdName(), dv.v.root.getIfdName(), err )
            return
        }
        sz = dv.v.root.dOffset + uint32(len(dv.header))
        dv.v.root.dSize = sz
    }

    dv.vCount = sz
    if dv.ifd.desc.srlzDbg {
        fmt.Printf( "%s ifd got embedded %s ifd size=%d\n",
                    dv.ifd.getIfdName(), dv.v.root.getIfdName(), sz )
    }

    if err = binary.Write( w, dv.ifd.desc.endian, dv.tVal.tEntry ); err == nil {
        err = binary.Write( w, dv.ifd.desc.endian, dv.ifd.dOffset )
        dv.ifd.dOffset += sz
    }
    return
}

func (dv *descValue)serializeData( w io.Writer ) (err error) {
    if dv.ifd.desc.srlzDbg {
        fmt.Printf( "%s ifd Serialize in data whole %s ifd @offset %#08x\n",
                    dv.ifd.getIfdName(), dv.v.root.getIfdName(), dv.ifd.dOffset )
    }

    _, err = w.Write( []byte( dv.header ) ) // including endian+0x002a+0x00000008
    if err != nil {
        return
    }
    _, err = dv.v.root.serializeEntries( w, dv.origin )
    if err != nil {
        return
    }
    _, err = dv.v.root.serializeDataArea( w, dv.origin )
    if err != nil {
        return
    }
    dv.ifd.dOffset += dv.v.root.dSize
    return
}

func (dv *descValue)format( w io.Writer ) {
    return  // Do nothing. Maker note will be separately formatted.
}

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
        if iv.ifd.desc.srlzDbg {
            fmt.Printf( "%s ifd serializeEntry: Get %s ifd size @offset %#08x\n",
                        iv.ifd.getIfdName(), iv.v.getIfdName(), iv.ifd.dOffset )
        }
        _, err = iv.v.serializeEntries( io.Discard, 0 )
        if err != nil {
            err = fmt.Errorf( "%s ifd serializeEntry: Get %s ifd size: %v\n",
                        iv.ifd.getIfdName(), iv.v.getIfdName(), err )
            return
        }
        sz = iv.v.dOffset   // since we serialized from offset 0
        iv.v.dSize = sz     // save in case serializeEntry is called again
    }
    if iv.ifd.desc.srlzDbg {
        fmt.Printf( "%s ifd got embedded %s ifd size=%d\n",
                    iv.ifd.getIfdName(), iv.v.getIfdName(), sz )
    }
    err = binary.Write( w, iv.ifd.desc.endian, iv.ifd.dOffset )
    iv.ifd.dOffset += sz
    return
}
func (iv *ifdValue)serializeData( w io.Writer ) (err error) {
    if iv.ifd.desc.srlzDbg {
        fmt.Printf( "%s ifd Serialize in data whole %s ifd @offset %#08x\n",
                    iv.ifd.getIfdName(), iv.v.getIfdName(), iv.ifd.dOffset )
    }
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
func (iv *ifdValue)format( w io.Writer ) {
    return  // Do nothing. The IFD will be separately formatted.
}

type thumbnailValue struct {
        tVal
    v   []uint8
}
func (ifd *ifdd) newThumbnailValue( tag tTag, tbnVal []uint8 ) (tbn *thumbnailValue) {
    tbn = new( thumbnailValue )
    tbn.ifd = ifd
    tbn.vTag = tag
    tbn.vType = ifd.fType
    tbn.vCount = ifd.fCount
    tbn.v = tbnVal
    return
}

func (tbn *thumbnailValue)serializeEntry( w io.Writer ) (err error) {
    endian := tbn.ifd.desc.endian
    err = binary.Write( w, endian, tbn.tEntry )
    if err != nil {
        return
    }
    if err = binary.Write( w, endian, tbn.ifd.dOffset ); err == nil {
        size := tbn.ifd.getAlignedDataSize( uint32(len(tbn.v)) )
        tbn.ifd.dOffset += size
    }
    return
}
func (tbn *thumbnailValue)serializeData( w io.Writer ) error {
    return tbn.ifd.serializeSliceData( w, tbn.v )
}
func (ub *thumbnailValue)format( w io.Writer ) {
}

type unsignedByteValue struct {
        tVal
    v   []uint8
    s   bool        // true if AsciiString (seen as unsigned byte slice)
}
func (ifd *ifdd) newUnsignedByteValue( name string, f func( interface{} ),
                                       ubVal []uint8 ) (ub *unsignedByteValue) {
    ub = new( unsignedByteValue )
    ub.ifd = ifd
    ub.fpr = f
    ub.name = name
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
func (ub *unsignedByteValue)format( w io.Writer ) {
    if ub.name != "" {
        fmt.Printf( "  %s:", ub.name )
        if ub.fpr == nil {
            if ub.s {
                fmt.Printf( " %s\n", string(ub.v) )
            } else {
                for i := 0; i < len(ub.v); i++ {
                    if i > 0 { io.WriteString( os.Stdout, "," ) }
                    fmt.Printf( " %d", ub.v[i] )
                }
                io.WriteString( os.Stdout, "\n" )
            }
        } else {
            io.WriteString( os.Stdout, " " )
            ub.fpr( ub.v )
        }
    }
}

// treat asciiStringgValue as unsignedByteValue 
func (ifd *ifdd) newAsciiStringValue( name string, asVal []byte ) (as *unsignedByteValue) {
    as = new( unsignedByteValue )
    as.ifd = ifd
    as.name = name
    as.vTag = ifd.fTag
    as.vType = ifd.fType
    as.vCount = uint32(len(asVal))  // assuming terminating 0 was included
    as.v = asVal
    as.s = true
    return
}

type signedByteValue  struct {
        tVal
    v   []int8
}
func (ifd *ifdd) newSignedByteValue( name string, f func( interface{} ),
                                     sbVal []int8 ) (sb *signedByteValue) {
    sb = new( signedByteValue )
    sb.ifd = ifd
    sb.fpr = f
    sb.name = name
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
func (sb *signedByteValue)format( w io.Writer ) {
    if sb.name != "" {
        fmt.Printf( "  %s:", sb.name )
        if sb.fpr == nil {
            for i:= 0; i < len(sb.v); i++ {
                if i > 0 { io.WriteString( os.Stdout, "," ) }
                fmt.Printf( " %d", sb.v[i] )
            }
            io.WriteString( os.Stdout, "\n" )
        } else {
            io.WriteString( os.Stdout, " " )
            sb.fpr( sb.v )
        }
    }
}

type unsignedShortValue struct {
        tVal
    v   []uint16
}
func (ifd *ifdd) newUnsignedShortValue( name string, f func( interface{} ),
                                    usVal []uint16 ) (us *unsignedShortValue) {
    us = new( unsignedShortValue )
    us.ifd = ifd
    us.fpr = f
    us.name = name
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
func (us *unsignedShortValue)format( w io.Writer ) {
    if us.name != "" {
        fmt.Printf( "  %s:", us.name )
        if us.fpr == nil {
            for i:= 0; i < len(us.v); i++ {
                if i > 0 { io.WriteString( os.Stdout, "," ) }
                fmt.Printf( " %d", us.v[i] )
            }
            io.WriteString( os.Stdout, "\n" )
        } else {
            io.WriteString( os.Stdout, " " )
            us.fpr( us.v )
        }
    }
}

type signedShortValue struct {
        tVal
    v   []int16
}
func (ifd *ifdd) newSignedShortValue( name string, f func( interface{} ),
                                      ssVal []int16 ) (ss *signedShortValue) {
    ss = new( signedShortValue )
    ss.ifd = ifd
    ss.fpr = f
    ss.name = name
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
func (ss *signedShortValue)format( w io.Writer ) {
    if ss.name != "" {
        fmt.Printf( "  %s:", ss.name )
        if ss.fpr == nil {
            i := 0
            for ; i < len(ss.v); i++ {
                if i > 0 { io.WriteString( os.Stdout, "," ) }
                fmt.Printf( " %d", ss.v[i] )
            }
            io.WriteString( os.Stdout, "\n" )
        } else {
            io.WriteString( os.Stdout, " " )
            ss.fpr( ss.v )
        }
    }
}

type unsignedLongValue struct {
        tVal
    v   []uint32
}
func (ifd *ifdd) newUnsignedLongValue( name string, f func( interface{} ),
                                     ulVal []uint32 ) (ul *unsignedLongValue) {
    ul = new( unsignedLongValue )
    ul.ifd = ifd
    ul.fpr = f
    ul.name = name
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
func (ul *unsignedLongValue)format( w io.Writer ) {
    if ul.name != "" {
        fmt.Printf( "  %s:", ul.name )
        if ul.fpr == nil {
            for i := 0; i < len(ul.v); i++ {
                if i > 0 { io.WriteString( os.Stdout, "," ) }
                fmt.Printf( " %d", ul.v[i] )
            }
            io.WriteString( os.Stdout, "\n" )
        } else {
            io.WriteString( os.Stdout, " " )
            ul.fpr( ul.v )
        }
    }
}

type signedLongValue struct {
        tVal
    v   []int32
}
func (ifd *ifdd) newSignedLongValue( name string, f func( interface{} ),
                                     slVal []int32 ) (sl *signedLongValue) {
    sl = new( signedLongValue )
    sl.ifd = ifd
    sl.fpr = f
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
func (sl *signedLongValue)format( w io.Writer ) {
    if sl.name != "" {
        fmt.Printf( "  %s:", sl.name )
        if sl.fpr == nil {
            for i:= 0; i < len(sl.v); i++ {
                if i > 0 { io.WriteString( os.Stdout, "," ) }
                fmt.Printf( " %d", sl.v[i] )
            }
            io.WriteString( os.Stdout, "\n" )
        } else {
            io.WriteString( os.Stdout, " " )
            sl.fpr( sl.v )
        }
    }
}

type unsignedRationalValue struct {
        tVal
    v  []unsignedRational
}
func (ifd *ifdd) newUnsignedRationalValue(
                    name string, f func( interface{} ),
                    urVal []unsignedRational ) (ur *unsignedRationalValue) {
    ur = new( unsignedRationalValue )
    ur.ifd = ifd
    ur.fpr = f
    ur.name = name
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
func (ur *unsignedRationalValue)format( w io.Writer ) {
    if ur.name != "" {
        fmt.Printf( "  %s:", ur.name )
        if ur.fpr == nil {
            for i := 0; i < len(ur.v); i++ {
                if i > 0 { io.WriteString( os.Stdout, "," ) }
                fmt.Printf( " %f (%d/%d)",
                        float32(ur.v[i].Numerator)/float32(ur.v[i].Denominator),
                        ur.v[i].Numerator, ur.v[i].Denominator )
            }
            io.WriteString( os.Stdout, "\n" )
        } else {
            io.WriteString( os.Stdout, " " )
            ur.fpr( ur.v )
        }
    }
}

type signedRationalValue struct {
        tVal
    v   []signedRational
}
func (ifd *ifdd) newSignedRationalValue(
                        name string, f func( interface{} ),
                        srVal []signedRational ) (sr *signedRationalValue) {
    sr = new( signedRationalValue )
    sr.ifd = ifd
    sr.fpr = f
    sr.name = name
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
func (sr *signedRationalValue)format( w io.Writer ) {
    if sr.name != "" {
        fmt.Printf( "  %s:", sr.name )
        if sr.fpr == nil {
            for i := 0; i < len(sr.v); i++ {
                if i > 0 { io.WriteString( os.Stdout, "," ) }
                fmt.Printf( " %f (%d/%d)",
                        float32(sr.v[i].Numerator)/float32(sr.v[i].Denominator),
                        sr.v[i].Numerator, sr.v[i].Denominator )
            }
            io.WriteString( os.Stdout, "\n" )
        } else {
            io.WriteString( os.Stdout, " " )
            sr.fpr( sr.v )
        }
    }
}

// storage does not presume any ifd data layout. This is done only at serializing
func (ifd *ifdd) storeValue( value serializer ) {
    i := len(ifd.values)
    if i == cap(ifd.values) {
        panic( "storeValue called with no more current IFD entries\n" )
    }
    if value == nil {
        panic("storeValue called with nil value\n")
    }
    if reflect.ValueOf(value).IsNil() {
        fmt.Printf( "storeValue called with a nil dynamic value (dynamic type %s)\n",
                    reflect.TypeOf(value).String() )
        panic("Stop")
    }
    ifd.values = ifd.values[:i+1]         // extend slice within capacity
//    fmt.Printf("storeValue: cap=%d len=%d i=%d\n", cap(ifd.values), len(ifd.values), i )
    ifd.values[i] = value
}

// All ifd.store<type> functions are always called with a valid ifd entry
// (fTag, fType, fCount and sOffset pointing at the value|offset). They
// check for a valid type and count (if appropriate), and if no error was
// found store the corresponding value in the current ifd. The argument
// name is the entry name that is displayed with the value. The argument
// print is the function that formats the value for display.

func (ifd *ifdd) storeEmbeddedIfd( name string, id IfdId,
                                   storeTags func( ifd *ifdd) error ) error {
    offset, err := ifd.checkUnsignedLongs( 1 )
    if err == nil {
        // recusively process the embedded IFD here
        var eIfd *ifdd
        _, eIfd, err = ifd.desc.storeIFD( id, offset[0], storeTags )
        if err == nil {
            ifd.storeValue( ifd.newIfdValue( eIfd ) )
        }
    }
    return err
}

func (ifd *ifdd) storeUnsignedBytes( name string, count uint32,
                                     p func(v interface{}) ) error {
    values, err := ifd.checkUnsignedBytes( count )
    if err == nil {
        ifd.storeValue( ifd.newUnsignedByteValue( name, p, values ) )
    }
    return err
}

func (ifd *ifdd) storeSignedBytes( name string, count uint32,
                                     p func(v interface{}) ) error {
    values, err := ifd.checkSignedBytes( count )
    if err == nil {
        ifd.storeValue( ifd.newSignedByteValue( name, p, values ) )
    }
    return err
}

func (ifd *ifdd) storeUndefinedAsUnsignedBytes( name string, count uint32,
                                                p func(v interface{}) ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "%s: incorrect type (%s)\n",
                           name, getTiffTString( ifd.fType ) )
    }
    if count != 0 && count != ifd.fCount {
        return fmt.Errorf( "%s: incorrect count (%d)\n", name, ifd.fCount )
    }
    ifd.storeValue( ifd.newUnsignedByteValue( name, p, ifd.getUnsignedBytes( ) ) )
    return nil
}

func (ifd *ifdd) storeUndefinedAsSignedBytes( name string, count uint32,
                                              p func(v interface{}) ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "%s: incorrect type (%s)\n",
                           name, getTiffTString( ifd.fType ) )
    }
    if count != 0 && count != ifd.fCount {
        return fmt.Errorf( "%s: incorrect count (%d)\n", name, ifd.fCount )
    }
    ifd.storeValue( ifd.newSignedByteValue( name, p, ifd.getSignedBytes( ) ) )
    return nil
}

// no pretty print should be necessary for strings.
func (ifd *ifdd) storeAsciiString( name string ) error {
    text, err := ifd.checkTiffAsciiString( )
    if err == nil {
        ifd.storeValue( ifd.newAsciiStringValue( name, text ) )
    }
    return err
}

func (ifd *ifdd) storeUnsignedShorts( name string, count uint32,
                                      p func(v interface{}) ) error {
    values, err := ifd.checkUnsignedShorts( count )
    if err == nil {
        ifd.storeValue( ifd.newUnsignedShortValue( name, p, values ) )
    }
    return err
}

func (ifd *ifdd) storeSignedShorts( name string, count uint32,
                                    p func(v interface{}) ) error {
    values, err := ifd.checkSignedShorts( count )
    if err == nil {
        ifd.storeValue( ifd.newSignedShortValue( name, p, values ) )
    }
    return err
}

func (ifd *ifdd) storeUndefinedAsSignedShorts( name string, count uint32,
                                          print func(v interface{}) ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "%s: incorrect type (%s)\n",
                           name, getTiffTString( ifd.fType ) )
    }
    if count != 0 && count != ifd.fCount {
        return fmt.Errorf( "%s: incorrect count (%d)\n", name, ifd.fCount )
    }
    ifd.storeValue( ifd.newSignedShortValue( name, print,
                                ifd.getSignedShorts( ) ) )
    return nil
}

func (ifd *ifdd) storeUnsignedLongs( name string, count uint32,
                                     p func(v interface{}) ) error {
    values, err := ifd.checkUnsignedLongs( count )
    if err == nil {
        ifd.storeValue( ifd.newUnsignedLongValue( name, p, values ) )
    }
    return err
}

func (ifd *ifdd) storeSignedLongs( name string, count uint32,
                                     p func(v interface{}) ) error {
    values, err := ifd.checkSignedLongs( count )
    if err == nil {
        ifd.storeValue( ifd.newSignedLongValue( name, p, values ) )
    }
    return err
}

func (ifd *ifdd) storeUnsignedRationals( name string, count uint32,
                                        p func(v interface{}) ) error {
    values, err := ifd.checkUnsignedRationals( count )
    if err == nil {
        ifd.storeValue( ifd.newUnsignedRationalValue( name, p, values ) )
    }
    return err
}

func (ifd *ifdd) storeSignedRationals( name string, count uint32,
                                       p func(v interface{}) ) error {
    values, err := ifd.checkSignedRationals( count )
    if err == nil {
        ifd.storeValue( ifd.newSignedRationalValue( name, p, values ) )
    }
    return err
}

// Store as read from the ifd entry fType and fCount, as long as fType is not
// _Undefined, which would not allow to know the data size.
func (ifd *ifdd) storeAnyNonUndefined( name string,
                                       p func(v interface{}) ) error {
    switch ifd.fType {
    case _UnsignedByte:     return ifd.storeUnsignedBytes( name, ifd.fCount, p )
    case _ASCIIString:      return ifd.storeAsciiString( name )
    case _UnsignedShort:    return ifd.storeUnsignedShorts( name, ifd.fCount, p )
    case _UnsignedLong:     return ifd.storeUnsignedLongs( name, ifd.fCount, p )
    case _UnsignedRational: return ifd.storeUnsignedRationals( name, ifd.fCount, p )
    case _SignedByte:       return ifd.storeSignedBytes( name, ifd.fCount, p )
    case _Undefined:        return fmt.Errorf( "storeAnyNonUndefined: undefined type\n")
    case _SignedShort:      return ifd.storeSignedShorts( name, ifd.fCount, p )
    case _SignedLong:       return ifd.storeSignedLongs( name, ifd.fCount, p )
    case _SignedRational:   return ifd.storeSignedRationals( name, ifd.fCount, p )
    }
    return fmt.Errorf( "storeAnyNonUndefined: unsupported type %s\n",
                       getTiffTString( ifd.fType ) )
}

func (ifd *ifdd) storeAnyUnknownSilently( ) error {
    if ifd.fType == _Undefined {
        return ifd.storeUndefinedAsUnsignedBytes( "", 0, nil )
    } else {
        return ifd.storeAnyNonUndefined( "", nil )
    }
}

