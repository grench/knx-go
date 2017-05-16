package proto

import (
	"bytes"
	"errors"
	"io"

	"github.com/vapourismo/knx-go/knx/binary"
)

// A TPCI is the transport-layer protocol control information (TPCI).
type TPCI uint8

//
const (
	UnnumberedDataPacket    TPCI = 0
	NumberedDataPacket      TPCI = 1
	UnnumberedControlPacket TPCI = 2
	NumberedControlPacket   TPCI = 3
)

// An APCI is the application-layer protocol control information (APCI).
type APCI uint8

//
const (
	GroupValueRead         APCI = 0
	GroupValueResponse     APCI = 1
	GroupValueWrite        APCI = 2
	IndividualAddrWrite    APCI = 3
	IndividualAddrRequest  APCI = 4
	IndividualAddrResponse APCI = 5
	AdcRead                APCI = 6
	AdcResponse            APCI = 7
	MemoryRead             APCI = 8
	MemoryResponse         APCI = 9
	MemoryWrite            APCI = 10
	UserMessage            APCI = 11
	MaskVersionRead        APCI = 12
	MaskVersionResponse    APCI = 13
	Restart                APCI = 14
	Escape                 APCI = 15
)

// A TPDU is the transport-layer protocol data unit within a L_Data frame.
type TPDU struct {
	PacketType TPCI
	SeqNumber  uint8
	Control    uint8
	Info       APCI
	Data       []byte
}

// Errors returned from ReadTPDU
var (
	ErrDataUnitTooShort = errors.New("Data segment of the TPDU is too short")
)

// ReadFrom parses the given data in order to fill the TPDU struct.
func (tpdu *TPDU) ReadFrom(r io.Reader) error {
	var head uint8
	err := binary.ReadSequence(r, &head)
	if err != nil {
		return err
	}

	packetType := TPCI((head >> 6) & 3)
	seqNumber := (head >> 2) & 15

	switch packetType {
	case UnnumberedControlPacket, NumberedControlPacket:
		tpdu.PacketType = packetType
		tpdu.SeqNumber = seqNumber
		tpdu.Control = head & 3
		tpdu.Info = 0
		tpdu.Data = nil

		return nil

	case UnnumberedDataPacket, NumberedDataPacket:
		buffer := &bytes.Buffer{}
		len, err := buffer.ReadFrom(r)
		if err != nil {
			return err
		} else if len < 1 {
			return ErrDataUnitTooShort
		}

		data := buffer.Bytes()
		info := APCI((head & 3) << 2 | (data[0] >> 6) & 3)

		var appData []byte
		if len > 1 {
			appData = data[1:]
		} else {
			appData = []byte{data[0] & 63}
		}

		tpdu.PacketType = packetType
		tpdu.SeqNumber = seqNumber
		tpdu.Control = 0
		tpdu.Info = info
		tpdu.Data = appData

		return nil
	}

	return errors.New("Unreachable")
}

// WriteTo writes the TPDU structure to the given Writer.
func (tpdu *TPDU) WriteTo(w io.Writer) error {
	buffer := []byte{
		byte(tpdu.PacketType & 3) << 6 | byte(tpdu.SeqNumber & 15) << 2,
	}

	switch tpdu.PacketType {
	case UnnumberedControlPacket, NumberedControlPacket:
		buffer[0] |= byte(tpdu.Control & 3)

	case UnnumberedDataPacket, NumberedDataPacket:
		buffer[0] |= byte(tpdu.Info >> 2) & 3

		if len(tpdu.Data) > 0 {
			buffer = append(buffer, tpdu.Data...)
			buffer[1] &= 63
			buffer[1] |= byte(tpdu.Info & 3) << 6
		} else {
			buffer = []byte{buffer[0], byte(tpdu.Info & 3) << 6}
		}
	}

	_, err := w.Write(buffer)
	return err
}
