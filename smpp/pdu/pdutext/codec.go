// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdutext

import (
	"unicode"
)

// DataCoding to define text codecs.
type DataCoding uint8

// Supported text codecs.
const (
	DefaultType DataCoding = 0x00 // SMSC Default Alphabet
	//	IA5Type       DataCoding = 0x01 // IA5 (CCITT T.50)/ASCII (ANSI X3.4)
	//	BinaryType    DataCoding = 0x02 // Octet unspecified (8-bit binary)
	Latin1Type  DataCoding = 0x03 // Latin 1 (ISO-8859-1)
	Binary2Type DataCoding = 0x04 // Octet unspecified (8-bit binary)
	//	JISType       DataCoding = 0x05 // JIS (X 0208-1990)
	ISO88595Type DataCoding = 0x06 // Cyrillic (ISO-8859-5)
	//	ISO88598Type  DataCoding = 0x07 // Latin/Hebrew (ISO-8859-8)
	UCS2Type DataCoding = 0x08 // UCS2 (ISO/IEC-10646)
	//	PictogramType DataCoding = 0x09 // Pictogram Encoding
	//	ISO2022JPType DataCoding = 0x0A // ISO-2022-JP (Music Codes)
	//	EXTJISType    DataCoding = 0x0D // Extended Kanji JIS (X 0212-1990)
	//	KSC5601Type   DataCoding = 0x0E // KS C 5601
)

// Codec defines a text codec.
type Codec interface {
	// Type returns the value for the data_coding PDU.
	Type() DataCoding

	// Encode text.
	Encode() []byte

	// Decode text.
	Decode() []byte

	Value() []byte
}

func (c DataCoding) Validate(input []byte) bool {
	if c == UCS2Type {
		return true
	}
	for _, r := range input {
		if !unicode.Is(alphabetMap[c], rune(r)) {
			return false
		}
	}
	return true
}

const (
	GSM_SINGLE_MAX_LEN  = 160
	UCS2_SINGLE_MAX_LEN = 70
)

func (c DataCoding) Encode(input []byte) (Codec, bool) {
	if c == DefaultType {
		code := GSM7(input)
		if len(input) > GSM_SINGLE_MAX_LEN {
			return code, true
		}
		return code, false
	} else if c == Latin1Type {
		code := Latin1(input)
		if len(input) > UCS2_SINGLE_MAX_LEN {
			return code, true
		}
		return code, false
	}
	code := UCS2(input)
	if len(input) > UCS2_SINGLE_MAX_LEN {
		return code, true
	}
	return code, false
}

func FindCoding(input []byte) (Codec, bool) {
	codings := []DataCoding{DefaultType, Latin1Type}
	for _, coding := range codings {
		if coding.Validate(input) {
			return coding.Encode(input)
		}
	}
	code := UCS2(input)
	if len(input) > UCS2_SINGLE_MAX_LEN {
		return code, true
	}
	return code, false
}
