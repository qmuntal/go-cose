package cose

import (
	"errors"
)

var (
	ErrInvalidAlg               = errors.New("Invalid algorithm")
	ErrAlgNotFound              = errors.New("Error fetching alg")
	ErrECDSAVerification        = errors.New("verification failed ecdsa.Verify")
	ErrRSAPSSVerification       = errors.New("verification failed rsa.VerifyPSS err crypto/rsa: verification error")
	ErrMissingCOSETagForLabel   = errors.New("No common COSE tag for label")
	ErrMissingCOSETagForTag     = errors.New("No common COSE label for tag")
	ErrNilSigHeader             = errors.New("Signature.headers is nil")
	ErrNilSigProtectedHeaders   = errors.New("Signature.headers.protected is nil")
	ErrNilSignatures            = errors.New("SignMessage.signatures is nil. Use AddSignature to add one")
	ErrNoSignatures             = errors.New("No signatures to sign the message. Use AddSignature to add them")
	ErrNoSignerFound            = errors.New("No signer found")
	ErrNoVerifierFound          = errors.New("No verifier found")
	ErrUnknownPrivateKeyType    = errors.New("Unrecognized private key type")
	ErrUnknownPublicKeyType     = errors.New("Unrecognized public key type")
	ErrNilSign1Headers          = errors.New("Sign1Message.headers is nil")
	ErrNilSign1ProtectedHeaders = errors.New("Sign1Message.headers.protected is nil")
)

// Common errors
var (
	ErrAlgorithmNotFound   = errors.New("algorithm not found")
	ErrAlgorithmRegistered = errors.New("algorithm registered")
	ErrInvalidAlgorithm    = errors.New("invalid algorithm")
	ErrUnavailableHashFunc = errors.New("hash function is not available")
	ErrUnknownAlgorithm    = errors.New("unknown algorithm")
)
