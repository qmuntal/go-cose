package cose

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/fxamacker/cbor/v2"
	ccbor "github.com/qmuntal/cbor"
)

// COSE Header labels registered in the IANA "COSE Header Parameters" registry.
//
// Reference: https://www.iana.org/assignments/cose/cose.xhtml#header-parameters
const (
	HeaderLabelAlgorithm         int64 = 1
	HeaderLabelCritical          int64 = 2
	HeaderLabelContentType       int64 = 3
	HeaderLabelKeyID             int64 = 4
	HeaderLabelCounterSignature  int64 = 7
	HeaderLabelCounterSignature0 int64 = 9
	HeaderLabelX5Bag             int64 = 32
	HeaderLabelX5Chain           int64 = 33
	HeaderLabelX5T               int64 = 34
	HeaderLabelX5U               int64 = 35
)

func encodeBasicType(b *ccbor.Builder, v interface{}) bool {
	switch v := v.(type) {
	case string:
		b.AddString(v)
	case int:
		b.AddInt(v)
	case int8:
		b.AddInt8(v)
	case int16:
		b.AddInt16(v)
	case int32:
		b.AddInt32(v)
	case int64:
		b.AddInt64(v)
	case uint:
		b.AddUint(v)
	case uint8:
		b.AddUint8(v)
	case uint16:
		b.AddUint16(v)
	case uint32:
		b.AddUint32(v)
	case uint64:
		b.AddUint64(v)
	case Algorithm:
		b.AddInt64(int64(v))
	default:
		return false
	}
	return true
}

func encodeMapValue(b *ccbor.Builder, v interface{}) {
	if encodeBasicType(b, v) {
		return
	}
	switch v := v.(type) {
	case []interface{}:
		b.AddArray(uint(len(v)), func(b *ccbor.Builder) {
			for _, item := range v {
				encodeMapValue(b, item)
			}
		})
	default:
		b.SetError(errors.New("cbor: unsupported type"))
	}
}

// ProtectedHeader contains parameters that are to be cryptographically
// protected.
type ProtectedHeader map[interface{}]interface{}

func (h ProtectedHeader) MarshalCBOR2(b *ccbor.Builder) {
	if len(h) == 0 {
		b.AddBytes([]byte{})
	} else {
		err := validateHeaderLabel(h)
		if err != nil {
			b.SetError(err)
			return
		}
		if err = h.ensureCritical(); err != nil {
			b.SetError(err)
			return
		}
		b.AddBytesUnknownLength(func(b *ccbor.Builder) {
			b.AddMap(len(h))
			for k, v := range h {
				b.AddMapItem(func(b *ccbor.Builder) {
					if !encodeBasicType(b, k) {
						b.SetError(errors.New("cbor: unsupported type"))
					}
				}, func(b *ccbor.Builder) {
					encodeMapValue(b, v)
				})
			}
		})
	}
}

// MarshalCBOR encodes the protected header into a CBOR bstr object.
// A zero-length header is encoded as a zero-length string rather than as a
// zero-length map (encoded as h'a0').
func (h ProtectedHeader) MarshalCBOR() ([]byte, error) {
	var b ccbor.Builder
	h.MarshalCBOR2(&b)
	return b.Bytes()
}

// UnmarshalCBOR decodes a CBOR bstr object into ProtectedHeader.
//
// ProtectedHeader is an empty_or_serialized_map where
// 	 empty_or_serialized_map = bstr .cbor header_map / bstr .size 0
func (h *ProtectedHeader) UnmarshalCBOR(data []byte) error {
	if h == nil {
		return errors.New("cbor: UnmarshalCBOR on nil ProtectedHeader pointer")
	}
	var encoded byteString
	if err := encoded.UnmarshalCBOR(data); err != nil {
		return err
	}
	if encoded == nil {
		return errors.New("cbor: nil protected header")
	}
	if len(encoded) == 0 {
		*h = make(ProtectedHeader)
	} else {
		if encoded[0]>>5 != 5 { // major type 5: map
			return errors.New("cbor: protected header: require map type")
		}
		if err := validateHeaderLabelCBOR(encoded); err != nil {
			return err
		}
		var header map[interface{}]interface{}
		if err := decMode.Unmarshal(encoded, &header); err != nil {
			return err
		}
		candidate := ProtectedHeader(header)
		if err := candidate.ensureCritical(); err != nil {
			return err
		}

		// cast to type Algorithm if `alg` presents
		if alg, err := candidate.Algorithm(); err == nil {
			candidate.SetAlgorithm(alg)
		}

		*h = candidate
	}
	return nil
}

// SetAlgorithm sets the algorithm value to the algorithm header.
func (h ProtectedHeader) SetAlgorithm(alg Algorithm) {
	h[HeaderLabelAlgorithm] = alg
}

// Algorithm gets the algorithm value from the algorithm header.
func (h ProtectedHeader) Algorithm() (Algorithm, error) {
	value, ok := h[HeaderLabelAlgorithm]
	if !ok {
		return 0, ErrAlgorithmNotFound
	}
	switch alg := value.(type) {
	case Algorithm:
		return alg, nil
	case int:
		return Algorithm(alg), nil
	case int8:
		return Algorithm(alg), nil
	case int16:
		return Algorithm(alg), nil
	case int32:
		return Algorithm(alg), nil
	case int64:
		return Algorithm(alg), nil
	default:
		return 0, ErrInvalidAlgorithm
	}
}

// Critical indicates which protected header labels an application that is
// processing a message is required to understand.
//
// Reference: https://datatracker.ietf.org/doc/html/rfc8152#section-3.1
func (h ProtectedHeader) Critical() ([]interface{}, error) {
	value, ok := h[HeaderLabelCritical]
	if !ok {
		return nil, nil
	}
	criticalLabels, ok := value.([]interface{})
	if !ok {
		return nil, errors.New("invalid crit header")
	}
	// if present, the array MUST have at least one value in it.
	if len(criticalLabels) == 0 {
		return nil, errors.New("empty crit header")
	}
	return criticalLabels, nil
}

// ensureCritical ensures all critical headers present in the protected bucket.
func (h ProtectedHeader) ensureCritical() error {
	labels, err := h.Critical()
	if err != nil {
		return err
	}
	for _, label := range labels {
		if _, ok := h[label]; !ok {
			return fmt.Errorf("missing critical header: %v", label)
		}
	}
	return nil
}

// UnprotectedHeader contains parameters that are not cryptographically
// protected.
type UnprotectedHeader map[interface{}]interface{}

// MarshalCBOR encodes the unprotected header into a CBOR map object.
// A zero-length header is encoded as a zero-length map (encoded as h'a0').
func (h UnprotectedHeader) MarshalCBOR2(b *ccbor.Builder) {
	if len(h) == 0 {
		b.AddMap(0)
		return
	}
	if err := validateHeaderLabel(h); err != nil {
		b.SetError(err)
		return
	}
	b.AddMap(len(h))
	for k, v := range h {
		b.AddMapItem(func(b *ccbor.Builder) {
			if !encodeBasicType(b, k) {
				b.SetError(errors.New("cbor: unsupported type"))
			}
		}, func(b *ccbor.Builder) {
			encodeMapValue(b, v)
		})
	}
}

// MarshalCBOR encodes the unprotected header into a CBOR map object.
// A zero-length header is encoded as a zero-length map (encoded as h'a0').
func (h UnprotectedHeader) MarshalCBOR() ([]byte, error) {
	var b ccbor.Builder
	h.MarshalCBOR2(&b)
	return b.Bytes()
}

// UnmarshalCBOR decodes a CBOR map object into UnprotectedHeader.
//
// UnprotectedHeader is a header_map.
func (h *UnprotectedHeader) UnmarshalCBOR(data []byte) error {
	if h == nil {
		return errors.New("cbor: UnmarshalCBOR on nil UnprotectedHeader pointer")
	}
	if data == nil {
		return errors.New("cbor: nil unprotected header")
	}
	if len(data) == 0 {
		return errors.New("cbor: unprotected header: missing type")
	}
	if data[0]>>5 != 5 { // major type 5: map
		return errors.New("cbor: unprotected header: require map type")
	}
	if err := validateHeaderLabelCBOR(data); err != nil {
		return err
	}
	var header map[interface{}]interface{}
	if err := decMode.Unmarshal(data, &header); err != nil {
		return err
	}
	*h = header
	return nil
}

// Headers represents "two buckets of information that are not
// considered to be part of the payload itself, but are used for
// holding information about content, algorithms, keys, or evaluation
// hints for the processing of the layer."
//
// It is represented by CDDL fragments:
//
//   Headers = (
//       protected : empty_or_serialized_map,
//       unprotected : header_map
//   )
//
//   header_map = {
//       Generic_Headers,
//       * label => values
//   }
//
//   label  = int / tstr
//   values = any
//
//   empty_or_serialized_map = bstr .cbor header_map / bstr .size 0
//
// See Also
//
// https://tools.ietf.org/html/rfc8152#section-3
type Headers struct {
	// RawProtected contains the raw CBOR encoded data for the protected header.
	// It is populated when decoding.
	// Applications can use this field for customized encoding / decoding of
	// the protected header in case the default decoder provided by this library
	// is not preferred.
	RawProtected cbor.RawMessage

	// Protected contains parameters that are to be cryptographically protected.
	// When encoding or signing, the protected header is encoded using the
	// default CBOR encoder if RawProtected is set to nil. Otherwise,
	// RawProtected will be used with Protected ignored.
	Protected ProtectedHeader

	// RawUnprotected contains the raw CBOR encoded data for the unprotected
	// header. It is populated when decoding.
	// Applications can use this field for customized encoding / decoding of
	// the unprotected header in case the default decoder provided by this
	// library is not preferred.
	RawUnprotected cbor.RawMessage

	// Unprotected contains parameters that are not cryptographically protected.
	// When encoding, the unprotected header is encoded using the default CBOR
	// encoder if RawUnprotected is set to nil. Otherwise, RawUnprotected will
	// be used with Unprotected ignored.
	Unprotected UnprotectedHeader
}

// MarshalProtected encodes the protected header.
// RawProtected is returned if it is not set to nil.
func (h *Headers) MarshalProtected() ([]byte, error) {
	if len(h.RawProtected) > 0 {
		return h.RawProtected, nil
	}
	return encMode.Marshal(h.Protected)
}

// MarshalProtected encodes the protected header.
// RawProtected is returned if it is not set to nil.
func (h *Headers) MarshalProtected2(b *ccbor.Builder) {
	if len(h.RawProtected) > 0 {
		b.AddRawBytes(h.RawProtected)
	} else {
		h.Protected.MarshalCBOR2(b)
	}
}

// MarshalUnprotected encodes the unprotected header.
// RawUnprotected is returned if it is not set to nil.
func (h *Headers) MarshalUnprotected() ([]byte, error) {
	if len(h.RawUnprotected) > 0 {
		return h.RawUnprotected, nil
	}
	return encMode.Marshal(h.Unprotected)
}

func (h *Headers) MarshalUnprotected2(b *ccbor.Builder) {
	if len(h.RawUnprotected) > 0 {
		b.AddRawBytes(h.RawUnprotected)
	} else {
		h.Unprotected.MarshalCBOR2(b)
	}
}

// UnmarshalFromRaw decodes Protected from RawProtected and Unprotected from
// RawUnprotected.
func (h *Headers) UnmarshalFromRaw() error {
	if err := decMode.Unmarshal(h.RawProtected, &h.Protected); err != nil {
		return fmt.Errorf("cbor: invalid protected header: %w", err)
	}
	if err := decMode.Unmarshal(h.RawUnprotected, &h.Unprotected); err != nil {
		return fmt.Errorf("cbor: invalid unprotected header: %w", err)
	}
	return nil
}

// ensureSigningAlgorithm ensures the presence of the `alg` header if there is
// no externally supplied data for signing.
//
// Reference: https://datatracker.ietf.org/doc/html/rfc8152#section-4.4
func (h *Headers) ensureSigningAlgorithm(alg Algorithm, external []byte) error {
	candidate, err := h.Protected.Algorithm()
	switch err {
	case nil:
		if candidate != alg {
			return fmt.Errorf("%w: signer %v: header %v", ErrAlgorithmMismatch, alg, candidate)
		}
		return nil
	case ErrAlgorithmNotFound:
		if len(external) > 0 {
			return nil
		}
		if h.RawProtected != nil {
			return ErrAlgorithmNotFound
		}
		if h.Protected == nil {
			h.Protected = make(ProtectedHeader)
		}
		h.Protected.SetAlgorithm(alg)
		return nil
	}
	return err
}

// ensureVerificationAlgorithm ensures the presence of the `alg` header if there
// is no externally supplied data for verification.
//
// Reference: https://datatracker.ietf.org/doc/html/rfc8152#section-4.4
func (h *Headers) ensureVerificationAlgorithm(alg Algorithm, external []byte) error {
	candidate, err := h.Protected.Algorithm()
	switch err {
	case nil:
		if candidate != alg {
			return fmt.Errorf("%w: verifier %v: header %v", ErrAlgorithmMismatch, alg, candidate)
		}
		return nil
	case ErrAlgorithmNotFound:
		if len(external) > 0 {
			return nil
		}
	}
	return err
}

// validateHeaderLabel validates if all header labels are integers or strings.
//
//   label = int / tstr
//
// Reference: https://datatracker.ietf.org/doc/html/rfc8152#section-1.4
func validateHeaderLabel(h map[interface{}]interface{}) error {
	existingInt := make(map[int64]struct{}, len(h))
	var existingString map[string]struct{}
	for label := range h {
		var k int64
		var isString bool
		switch v := label.(type) {
		case int:
			k = int64(v)
		case int8:
			k = int64(v)
		case int16:
			k = int64(v)
		case int32:
			k = int64(v)
		case int64:
			k = int64(v)
		case uint:
			k = int64(v)
		case uint8:
			k = int64(v)
		case uint16:
			k = int64(v)
		case uint32:
			k = int64(v)
		case uint64:
			k = int64(v)
		case string:
			if existingString == nil {
				existingString = make(map[string]struct{})
			}
			isString = true
			if _, ok := existingString[v]; ok {
				return fmt.Errorf("cbor: header label: duplicated label: %v", v)
			} else {
				existingString[v] = struct{}{}
			}
			// no conversion
		default:
			return errors.New("cbor: header label: require int / tstr type")
		}
		if !isString {
			if _, ok := existingInt[k]; ok {
				return fmt.Errorf("cbor: header label: duplicated label: %v", label)
			} else {
				existingInt[k] = struct{}{}
			}
		}
	}
	return nil
}

// headerLabelValidator is used to validate the header label of a COSE header.
type headerLabelValidator struct {
	value interface{}
}

// String prints the value without brackets `{}`. Useful in error printing.
func (hlv headerLabelValidator) String() string {
	return fmt.Sprint(hlv.value)
}

// UnmarshalCBOR decodes the label value of a COSE header, and returns error if
// label is not a int (major type 0, 1) or string (major type 3).
func (hlv *headerLabelValidator) UnmarshalCBOR(data []byte) error {
	if len(data) == 0 {
		return errors.New("cbor: header label: missing type")
	}
	switch data[0] >> 5 {
	case 0, 1, 3:
		err := decMode.Unmarshal(data, &hlv.value)
		if err != nil {
			return err
		}
		if _, ok := hlv.value.(big.Int); ok {
			return errors.New("cbor: header label: int key must not be higher than 1<<63 - 1")
		}
		return nil
	}
	return errors.New("cbor: header label: require int / tstr type")
}

// discardedCBORMessage is used to read CBOR message and discard it.
type discardedCBORMessage struct{}

// UnmarshalCBOR discards the read CBOR object.
func (discardedCBORMessage) UnmarshalCBOR(data []byte) error {
	return nil
}

// validateHeaderLabelCBOR validates if all header labels are integers or
// strings of a CBOR map object.
//
//   label = int / tstr
//
// Reference: https://datatracker.ietf.org/doc/html/rfc8152#section-1.4
func validateHeaderLabelCBOR(data []byte) error {
	var header map[headerLabelValidator]discardedCBORMessage
	return decMode.Unmarshal(data, &header)
}
