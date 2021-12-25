package exif

import (
    "fmt"
//    "bytes"
//    "strings"
    "encoding/binary"
    "io"
)

// Serialize Desc

func (d *Desc)serialize( w io.Writer ) (written int, err error) {

    if written, err = w.Write( []byte( "Exif\x00\x00" ) ); err != nil {
        return
    }

    var es string                // TIFF header starts here
    if d.endian == binary.BigEndian { es = "MM" } else { es = "II" }
    _, err = w.Write( []byte( es ) )
    if err != nil {
        return
    }
    written += 2

    err = binary.Write( w, d.endian, uint16(0x002a) )
    if err != nil {
        return
    }
    written += 2

    err = binary.Write( w, d.endian, uint32(0x00000008) )
    if err != nil {
        return
    }
    written += 4
//fmt.Printf( "exif/tiff header uses %d bytes\n", written )
    var ns uint32
    ns, err = d.root.serializeEntries( w, _headerSize )
    if err != nil {
        return
    }
//fmt.Printf( "ifd0 entries use %d bytes\n", ns )
//fmt.Printf( "after serializeEntries ifd0 dOffset=%d\n", d.root.dOffset )
    written += int(ns)
    ns, err = d.root.serializeDataArea( w, _headerSize )
    if err != nil {
        return
    }
//fmt.Printf( "ifd0 data area uses %d bytes\n", ns )
//fmt.Printf( "after serializeDataArea ifd0 dOffset=%d\n", d.root.dOffset )
    written += int(ns)
//fmt.Printf("After ifd0, written=%d\n", written )
    if d.root.next != nil {    // store thumbnail IFD
        offset := d.root.dOffset
        ns, err = d.root.next.serializeEntries( w, offset )
        if err != nil {
            return
        }
        written += int(ns)
        ns, err = d.root.next.serializeDataArea( w, offset )
        if err != nil {
            return
        }
        written += int(ns)
        tOffset, _ := d.global["thumbOffset"].(uint32)
        tLen, _ := d.global["thumbLen"].(uint32)
        var nw int      // store embedded thumnail data
        nw, err = w.Write( d.data[tOffset:tOffset+tLen] )
        if err == nil {
            written += nw
        }
    }
    return
}

func (ifd *ifdd)serializeEntries( w io.Writer, offset uint32 ) (uint32, error) {

    // calculate where data area starts
    ifd.dOffset = offset +                      // from parent ifd (if any)
                  _ShortSize +                  // number of IFD entries
                  (uint32(len(ifd.values)) * _IfdEntrySize) +
                  _LongSize                     // next IFD offset

    endian := ifd.desc.endian
    written := uint32(0)

    // write number of entries first as an _UnsignedShort
    fmt.Printf( "ifd %d serialize: n entries %d dOffset %#08x\n",
                ifd.id, len(ifd.values), ifd.dOffset )
    err := binary.Write( w, endian, uint16(len(ifd.values)) )
    if err != nil {
        return written, err
    }
    written += _ShortSize

    // Write fixed size entries, including in-place values
    for i := 0; i < len(ifd.values); i++ {
        err = ifd.values[i].serializeEntry( w )
        if err != nil {
            fmt.Printf("ifd %d serialize entry %d returned error %v\n",
                        ifd.id, i, err )
            return written, err
        }
//        fmt.Printf( "ifd %d serialized entry %d dOffset %#08x\n", ifd.id, i, ifd.dOffset )
        written += _IfdEntrySize
    }

    var nOffset uint32
    if ifd.next != nil {      // next IFD follows immediately the current one
        nOffset = ifd.dOffset
    }
    fmt.Printf( "ifd %d serialize: next ifd at offset %#08x\n", ifd.id, nOffset )
    err = binary.Write( w, endian, nOffset )
    if err != nil {
        return written, err
    }
    return written + _LongSize, nil
}

func (ifd *ifdd)serializeDataArea( w io.Writer, offset uint32 ) (uint32, error) {

    // calculate where data area starts
    offset += _ShortSize + (uint32(len(ifd.values)) * _IfdEntrySize) + _LongSize
    ifd.dOffset = offset

    var err error
    written := uint32(0)

    // Write variable size values, excluding in-place values
    for i := 0; i < len(ifd.values); i++ {
        err = ifd.values[i].serializeData( w )
        if err != nil {
            fmt.Printf("ifd %d serialize data for entry %d returned error %v\n",
                        ifd.id, i, err )
            break
        }
//        fmt.Printf( "ifd %d serialized data for entry %d dOffset %#08x\n", ifd.id, i, ifd.dOffset )
    }
    written += ifd.dOffset - offset
    fmt.Printf( "ifd serialize: returning with size %d\n", written )
    return written, err
}

func getSliceSize( sl interface{} ) uint32 {
    size := binary.Size( sl )
//    fmt.Printf("getSliceSize: size = %d\n", size )
    if size == -1 {         // binary does not like strings or some other types
        if v, ok := sl.(string); ok {
            return uint32(len(v))
        }
        panic(fmt.Sprintf( "getSliceType: data type (%T) is not suitable\n", sl ))
    }
    return uint32(size)
}

// (ifd *ifdd)serializeSliceEntry()
// It serializes an IFD entry containing a slice of values (most common case)
// The arguments are an io.Writer, the entry tag+type+count, the slice values
// and an offset into the IFD data area where the value will be written in a
// later phase if the value does not fit in _valOffSize (4 bytes).
// The return value is an error indicating a failure.
// By side effect the ifd dOffset is updated for next calls with the size to
// be written later in the IFD data area or 0 if it fits in _valOffSize
func (ifd *ifdd)serializeSliceEntry( w io.Writer, eTT tEntry,
                                     sl interface{} ) (err error) {

    endian := ifd.desc.endian
    err = binary.Write( w, endian, eTT )
    if err != nil {
        return
    }
    size := getSliceSize( sl )
//    fmt.Printf( "serializeSliceEntry: tag %#04x type %s count %d size %d\n",
//                eTT.vTag, getTiffTString(eTT.vType), eTT.vCount, size )
    if size <= _valOffSize {
        if err = binary.Write( w, endian, sl ); err == nil {
            if size != _valOffSize {
                _, err = w.Write( make( []byte, _valOffSize-size ) )
            }
        }
    } else {
        if size & 1 == 1 {  // round up next data offset to 2-byte boundary
            size += 1
        }
        if err = binary.Write( w, endian, ifd.dOffset ); err == nil {
            ifd.dOffset += size
        }
    }
    return
}

func getSliceDataSize( sl interface{} ) uint32 {
    size := getSliceSize( sl )
    if size <= _valOffSize {
        return 0
    }
    return size
}

// (ifd *ifdd)serializeSliceData(
// This is the second phase of writing an IFD entry. It serializes an IFD entry
// containing a slice of values (most common case) in the IFD data area.
// The arguments are an io.Writer, and the slice values. 
// If the value does fit in _valOffSize (4 bytes), the function does nothing.
// The return value is an error indicating a failure.
// By side effect the ifd dOffset is updated for next calls with the size to
// be written later in the IFD data area or 0 if it fits in _valOffSize
func (ifd *ifdd)serializeSliceData( w io.Writer,
                                  sl interface{} ) (err error) {
    size := getSliceDataSize( sl )
    if size > 0 {
        if err = binary.Write( w, ifd.desc.endian, sl ); err == nil {
            if size & 1 == 1 {
                _, err = w.Write( []byte{0} )
                if err == nil {
                    size += 1
                }
            }
        }
    }
    ifd.dOffset += size
    return
}

func (ifd *ifdd)format( w io.Writer ) error {

    for i := 0; i < len(ifd.values); i++ {
        ifd.values[i].format( w )
    }
    return nil
}

