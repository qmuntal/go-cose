package cose

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"fmt"
	"io"
	"math/big"

	"golang.org/x/crypto/cryptobyte"
	"golang.org/x/crypto/cryptobyte/asn1"
)

// I2OSP - Integer-to-Octet-String primitive converts a nonnegative integer to
// an octet string of a specified length `len(buf)`, and stores it in `buf`.
//
// Reference: https://datatracker.ietf.org/doc/html/rfc8017#section-4.1
func I2OSP(x *big.Int, buf []byte) error {
	if x.Sign() < 0 {
		return errors.New("I2OSP: negative integer")
	}
	if len(x.Bits()) > len(buf) {
		return errors.New("I2OSP: integer too large")
	}
	x.FillBytes(buf)
	return nil
}

// OS2IP - Octet-String-to-Integer primitive converts an octet string to a
// nonnegative integer.
//
// Reference: https://datatracker.ietf.org/doc/html/rfc8017#section-4.2
func OS2IP(x []byte) *big.Int {
	return new(big.Int).SetBytes(x)
}

// ecdsaKeySigner is a ECDSA based signer with golang built-in keys.
type ecdsaKeySigner struct {
	alg Algorithm
	key *ecdsa.PrivateKey
}

// Algorithm returns the signing algorithm associated with the private key.
func (es *ecdsaKeySigner) Algorithm() Algorithm {
	return es.alg
}

// Sign signs digest with the private key, possibly using entropy from rand.
// The resulting signature should follow RFC 8152 section 8.1.
//
// Reference: https://datatracker.ietf.org/doc/html/rfc8152#section-8.1
func (es *ecdsaKeySigner) Sign(rand io.Reader, digest []byte) ([]byte, error) {
	r, s, err := ecdsa.Sign(rand, es.key, digest)
	if err != nil {
		return nil, err
	}
	return encodeECDSASignature(es.key.Curve, r, s)
}

// ecdsaKeySigner is a ECDSA based signer with a generic crypto.Signer.
type ecdsaCryptoSigner struct {
	alg    Algorithm
	key    *ecdsa.PublicKey
	signer crypto.Signer
}

// Algorithm returns the signing algorithm associated with the private key.
func (es *ecdsaCryptoSigner) Algorithm() Algorithm {
	return es.alg
}

// Sign signs digest with the private key, possibly using entropy from rand.
// The resulting signature should follow RFC 8152 section 8.1.
//
// Reference: https://datatracker.ietf.org/doc/html/rfc8152#section-8.1
func (es *ecdsaCryptoSigner) Sign(rand io.Reader, digest []byte) ([]byte, error) {
	sig, err := es.signer.Sign(rand, digest, nil)
	if err != nil {
		return nil, err
	}
	r, s, err := decodeECDSASignatureASN1(sig)
	if err != nil {
		return nil, err
	}
	return encodeECDSASignature(es.key.Curve, r, s)
}

// decodeECDSASignatureASN1 decodes (r, s) from ASN.1 encoded signature.
//
// Code copied from https://github.com/golang/go/blob/go1.18/src/crypto/ecdsa/ecdsa.go#L338-L354
func decodeECDSASignatureASN1(sig []byte) (r, s *big.Int, err error) {
	r, s = &big.Int{}, &big.Int{}
	var inner cryptobyte.String
	input := cryptobyte.String(sig)
	if !input.ReadASN1(&inner, asn1.SEQUENCE) ||
		!input.Empty() ||
		!inner.ReadASN1Integer(r) ||
		!inner.ReadASN1Integer(s) ||
		!inner.Empty() {
		return nil, nil, errors.New("ecdsa: invalid signature: invalid ASN.1 encoding")
	}
	return
}

// encodeECDSASignature encodes (r, s) into a signature binary string using the
// method specified by RFC 8152 section 8.1.
func encodeECDSASignature(curve elliptic.Curve, r, s *big.Int) ([]byte, error) {
	n := (curve.Params().BitSize + 7) / 8
	sig := make([]byte, n*2)
	if err := I2OSP(r, sig[:n]); err != nil {
		return nil, err
	}
	if err := I2OSP(s, sig[n:]); err != nil {
		return nil, err
	}
	return sig, nil
}

// decodeECDSASignature decodes (r, s) from a signature binary string using the
// method specified by RFC 8152 section 8.1.
func decodeECDSASignature(curve elliptic.Curve, sig []byte) (r, s *big.Int, err error) {
	n := (curve.Params().BitSize + 7) / 8
	if len(sig) != n*2 {
		return nil, nil, fmt.Errorf("invalid signature length: %d", len(sig))
	}
	return OS2IP(sig[:n]), OS2IP(sig[n:]), nil
}

// ecdsaVerifier is a ECDSA based verifier with golang built-in keys.
type ecdsaVerifier struct {
	alg Algorithm
	key *ecdsa.PublicKey
}

// Algorithm returns the signing algorithm associated with the public key.
func (ev *ecdsaVerifier) Algorithm() Algorithm {
	return ev.alg
}

// Verify verifies digest with the public key, returning nil for success.
// Otherwise, it returns an error.
//
// Reference: https://datatracker.ietf.org/doc/html/rfc8152#section-8.1
func (ev *ecdsaVerifier) Verify(digest []byte, signature []byte) error {
	r, s, err := decodeECDSASignature(ev.key.Curve, signature)
	if err != nil {
		return ErrVerification
	}
	if verified := ecdsa.Verify(ev.key, digest, r, s); !verified {
		return ErrVerification
	}
	return nil
}
