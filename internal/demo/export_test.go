package demo

// Test-only exports for unexported functions.
// Used by *_test.go files in package demo_test.

// ResolveRoundIDForTest exposes resolveRoundID for external tests.
var ResolveRoundIDForTest = resolveRoundID

// MarshalExtraDataForTest exposes marshalExtraData for external tests.
var MarshalExtraDataForTest = marshalExtraData

// ChunkTickParamsForTest exposes chunkTickParams for external tests.
var ChunkTickParamsForTest = chunkTickParams
