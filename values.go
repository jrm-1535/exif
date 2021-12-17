package exif

import (
    "fmt"
    "encoding/binary"
    "io"
    "os"
)

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

func (ifd *ifdd) getUnsignedShorts( ) []uint16 {
    if ifd.fCount * _ShortSize <= 4 {
        return ifd.desc.getUnsignedShorts( ifd.sOffset, ifd.fCount )
    } else {
        rOffset := ifd.desc.getUnsignedLong( ifd.sOffset )
        return ifd.desc.getUnsignedShorts( rOffset, ifd.fCount )
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
func (iv *ifdValue)format( w io.Writer ) {
    return  // Do nothing. The IFD will be separately formatted.
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
    as.vCount = uint32(len(asVal))
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
    if i >= cap(ifd.values) {
        panic( "storeValue called with no more current IFD entries\n" )
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

func (ifd *ifdd) storeUndefinedAsBytes( name string, count uint32,
                                       print func(v interface{}) ) error {
    if ifd.fType != _Undefined {
        return fmt.Errorf( "%s: incorrect type (%s)\n",
                           name, getTiffTString( ifd.fType ) )
    }
    if count != 0 && count != ifd.fCount {
        return fmt.Errorf( "%s: incorrect count (%d)\n", name, ifd.fCount )
    }
    ifd.storeValue( ifd.newUnsignedByteValue( name, print,
                                ifd.getUnsignedBytes( ) ) )
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
