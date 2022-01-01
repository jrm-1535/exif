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
    var ns uint32
    ns, err = d.root.serializeEntries( w, _headerSize )
    if err != nil {
        return
    }
//fmt.Printf( "ifd0 entries use %d bytes data area @offset=%d\n", ns, d.root.dOffset )
    written += int(ns)
    ns, err = d.root.serializeDataArea( w, _headerSize )
    if err != nil {
        return
    }
//fmt.Printf( "ifd0 data area uses %d bytes next offset=%d\n", ns, d.root.dOffset )
    written += int(ns)
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
    }
    return
}

func (ifd *ifdd)setDataAreaStart( offset uint32 ) {
// TODO: ignore offset as far as alignment is concerned?
    // calculate where data area starts

    ifd.dOffset = offset + ifd.getAlignedDataSize(
                            _ShortSize +    // number of IFD entries
                            (uint32(len(ifd.values)) * _IfdEntrySize) +
                            _LongSize )     // next IFD offset
}

func (ifd *ifdd)alignDataArea( w io.Writer, end uint32 ) uint32 {
    nOffset := ifd.getAlignedDataSize( end )
    if nOffset != end {
        pad := make( []byte, nOffset - end )
        w.Write( pad )
        return nOffset - end
    }
    return 0
}

func (ifd *ifdd)serializeEntries( w io.Writer, offset uint32 ) (uint32, error) {
    ifd.setDataAreaStart( offset )
    // calculate where data area starts
//    ifd.dOffset = offset +                      // from parent ifd (if any)
//                  _ShortSize +                  // number of IFD entries
//                  (uint32(len(ifd.values)) * _IfdEntrySize) +
//                  _LongSize                     // next IFD offset

    endian := ifd.desc.endian
    written := uint32(0)

    // write number of entries first as an _UnsignedShort
    fmt.Printf( "%s ifd serialize: %d entries starting @%#08x data Offset %#08x\n",
                ifd.getIfdName(), len(ifd.values), offset, ifd.dOffset )
    err := binary.Write( w, endian, uint16(len(ifd.values)) )
    if err != nil {
        return written, err
    }
    written += _ShortSize

    // Write fixed size entries, including in-place values
    for i := 0; i < len(ifd.values); i++ {
        err = ifd.values[i].serializeEntry( w )
        if err != nil {
            fmt.Printf("%s ifd serializeEntry %d returned error %v\n",
                        ifd.getIfdName(), i, err )
            return written, err
        }
        fmt.Printf( "%s ifd serialized entry %d dOffset %#08x\n",
                    ifd.getIfdName(), i, ifd.dOffset )
        written += _IfdEntrySize
    }

    nOffset := ifd.getAlignedDataSize( ifd.dOffset )
    if nOffset != ifd.dOffset {
        fmt.Printf( "####### %s ifd aligned nOffset %#08x actual end of data area %#08x\n",
                    ifd.getIfdName(), nOffset, ifd.dOffset )
        ifd.dOffset = nOffset
    }

    if ifd.next == nil {      // next IFD follows immediately the current one
        nOffset = 0
    }
    fmt.Printf( "%s ifd serialize: next ifd at offset %#08x\n", ifd.getIfdName(), nOffset )
    err = binary.Write( w, endian, nOffset )
    if err != nil {
        return written, err
    }
    written += _LongSize
    written += ifd.alignDataArea( w, written ) // keep data area correctly aligned
    return written, nil
}

func (ifd *ifdd)serializeDataArea( w io.Writer, offset uint32 ) (uint32, error) {
    ifd.setDataAreaStart( offset )
    offset = ifd.dOffset            // keep start of data area for later use

    // calculate where data area starts
//    offset += _ShortSize + (uint32(len(ifd.values)) * _IfdEntrySize) + _LongSize
//    ifd.dOffset = offset
    var err error

    // Write variable size values, excluding in-place values
    for i := 0; i < len(ifd.values); i++ {
        err = ifd.values[i].serializeData( w )
        if err != nil {
            fmt.Printf("%s ifd serialize data for entry %d returned error %v\n",
                        ifd.getIfdName(), i, err )
            break
        }
//        fmt.Printf( "ifd %d serialized data for entry %d dOffset %#08x\n", ifd.id, i, ifd.dOffset )
    }

    written := ifd.dOffset - offset
    fmt.Printf( "%s ifd serialize data: returning with size %d\n", ifd.getIfdName(), written )
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

func (ifd *ifdd)getAlignedDataSize( sz uint32 ) uint32 {
    if ifd.desc.Align4 {    // rount up to 4-byte boundary
        return ((sz + 3)/4) * 4
    }
    if sz & 1 == 1 {        // round up to 2-byte boundary
        sz ++
    }
    return sz
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
        if err = binary.Write( w, endian, ifd.dOffset ); err == nil {
            size = ifd.getAlignedDataSize( size )
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
            aSize := ifd.getAlignedDataSize( size )
            if aSize != size {
                pad := make( []byte, aSize - size )
                 _, err = w.Write( pad )
                size = aSize
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

