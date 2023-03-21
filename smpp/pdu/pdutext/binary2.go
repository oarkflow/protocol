// just for mock

package pdutext

// Raw text codec, no encoding.
type Binary2 []byte

// Type implements the Codec interface.
func (Binary2) Type() DataCoding {
	return Binary2Type
}

// Encode binary2 content.
func (b Binary2) Encode() []byte {
	return b
}

// Decode binary2 content.
func (b Binary2) Decode() []byte {
	return b
}

// Value binary2 content.
func (b Binary2) Value() []byte {
	return b
}
