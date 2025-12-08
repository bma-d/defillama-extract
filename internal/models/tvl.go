package models

// CustomProtocol represents a single custom protocol entry loaded from
// config/custom-protocols.json. Fields mirror the JSON schema exactly, using
// hyphenated keys where specified. Optional fields use pointers so absence
// can be distinguished from zero values.
type CustomProtocol struct {
	Slug           string  `json:"slug"`
	IsOngoing      bool    `json:"is-ongoing"`
	Live           bool    `json:"live"`
	Date           *int64  `json:"date,omitempty"`         // Unix timestamp, optional
	SimpleTVSRatio float64 `json:"simple-tvs-ratio"`       // TVS multiplier in range [0,1]
	DocsProof      *string `json:"docs_proof,omitempty"`   // Optional documentation URL
	GitHubProof    *string `json:"github_proof,omitempty"` // Optional repository link proving integration
}
