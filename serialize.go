package exif

import (
    "fmt"
    "encoding/binary"
    "io"
)

// Serialize the parsed EXIF metadata, including all current IFDs.
// The argument w is the io.Writer to use.
//
// It returns the number of bytes written in case of success or a non-nil error
// in case of failure.
func (d *Desc)Serialize( w io.Writer ) (written int, err error) {

    if d.root == nil {
        return 0, fmt.Errorf( "Serialize: empty descriptor\n" )
    }

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
    written += int(ns)
    ns, err = d.root.serializeDataArea( w, _headerSize )
    if err != nil {
        return
    }
    written += int(ns)
    if d.root.next != nil {    // thumbnail IFD
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

func (ifd *ifdd)setDataAreaStart( origin uint32 ) (nEntries uint32 ){
    if origin & 1 == 1 {
        panic( fmt.Sprintf(
                "setDataAreaStart: origin is not aligned on 2-byte boundaries: %#08x\n",
                origin ) )
    }

    // since some entries may have been removed after parsing, calculate and
    // return the actual number of entries that will be written.
    for _, val := range ifd.values {
        if val != nil { nEntries ++ }
    }

    // calculate where data area starts
    ifd.dOffset = origin + _ShortSize +         // number of IFD entries
                  (nEntries * _IfdEntrySize) +  // actual entries
                  _LongSize                     // next IFD offset
    return
}

func (ifd *ifdd)serializeEntries( w io.Writer, offset uint32 ) (uint32, error) {
    nEntries := ifd.setDataAreaStart( offset )
    endian := ifd.desc.endian
    written := uint32(0)

    if ifd.desc.SrlzDbg {
        fmt.Printf( "%s ifd serialize: %d entries starting @%#08x data Offset %#08x\n",
                    GetIfdName(ifd.id), len(ifd.values), offset, ifd.dOffset )
    }
    // write number of entries first as an _UnsignedShort
    err := binary.Write( w, endian, uint16(nEntries) )
    if err != nil {
        return written, err
    }
    written += _ShortSize

    // Write fixed size entries, including in-place values
    for i := 0; i < len(ifd.values); i++ {
        if ifd.values[i] == nil {   // removed entries must be ignored
            if ifd.desc.SrlzDbg {
                fmt.Printf( "%s ifd serializeEntry %d skipping empty entry\n",
                            GetIfdName(ifd.id), i )
            }
            continue
        }
        err = ifd.values[i].serializeEntry( w )
        if err != nil {
            err = fmt.Errorf( "%s ifd serializeEntry %d: %v\n",
                              GetIfdName(ifd.id), i, err )
            return written, err
        }
        if ifd.desc.SrlzDbg {
            fmt.Printf( "%s ifd serialized entry %d dOffset %#08x\n",
                        GetIfdName(ifd.id), i, ifd.dOffset )
        }
        written += _IfdEntrySize
    }

    nIfdOffset := ifd.dOffset
    if ifd.next == nil {
        nIfdOffset = 0
    }
    if ifd.desc.SrlzDbg {
        fmt.Printf( "%s ifd serialize: next ifd at offset %#08x\n",
                    GetIfdName(ifd.id), nIfdOffset )
    }
    err = binary.Write( w, endian, nIfdOffset )
    if err != nil {
        return written, err
    }
    written += _LongSize
    return written, nil
}

func (ifd *ifdd)serializeDataArea( w io.Writer, origin uint32 ) (uint32, error) {
    ifd.setDataAreaStart( origin )
    origin = ifd.dOffset            // keep start of data area for later use
    var err error

    // Write variable size values, excluding in-place values
    for i := 0; i < len(ifd.values); i++ {
        if ifd.values[i] == nil {   // removed entries must be ignored
            if ifd.desc.SrlzDbg {
                fmt.Printf( "%s ifd serializeDataArea %d skipping empty entry\n",
                            GetIfdName(ifd.id), i )
            }
            continue
        }
        err = ifd.values[i].serializeData( w )
        if err != nil {
            err = fmt.Errorf( "%s ifd serializeDataArea for entry %d: %v\n",
                              GetIfdName(ifd.id), i, err )
            return 0, err
        }
        if ifd.desc.SrlzDbg {
            fmt.Printf( "%s ifd serialized data for entry %d dOffset %#08x\n",
                        GetIfdName(ifd.id), i, ifd.dOffset )
        }
    }

    written := ifd.dOffset - origin
    if ifd.desc.SrlzDbg {
        fmt.Printf( "%s ifd serialize data: returning with size %d\n",
                    GetIfdName(ifd.id), written )
    }
    return written, err
}

func getSliceSize( sl interface{} ) uint32 {
    size := binary.Size( sl )
    if size == -1 {         // binary does not like strings or some other types
        if v, ok := sl.(string); ok {
            return uint32(len(v))
        }
        panic(fmt.Sprintf( "getSliceType: data type (%T) is not suitable\n", sl ))
    }
    return uint32(size)
}

func (ifd *ifdd)getAlignedDataSize( sz uint32 ) uint32 {
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
    if ifd.desc.SrlzDbg {
        fmt.Printf( "serializeSliceEntry: tag %#04x type %s count %d size %d\n",
                    eTT.vTag, getTiffTString(eTT.vType), eTT.vCount, size )
    }
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

