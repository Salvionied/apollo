package constants

// Tunable safety limits for Plutus data decoding.
// Change these values here when a deployment needs a different bound.
const (
	// MaxContainerPrealloc is the largest initial slice or map capacity used
	// when decoding or unmarshaling untrusted Plutus data. Larger containers
	// grow as elements pass validation.
	MaxContainerPrealloc = 1024
)
