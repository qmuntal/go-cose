module github.com/veraison/go-cose

go 1.18

require (
	github.com/fxamacker/cbor/v2 v2.4.0
	github.com/qmuntal/cbor v0.0.0-00010101000000-000000000000
)

require github.com/x448/float16 v0.8.4 // indirect

replace github.com/qmuntal/cbor => ../cbor
