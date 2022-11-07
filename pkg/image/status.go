package image

// Status indicates the operational state of an Image
type Status string

// PENDING Status indicates that further processing is required (e.g. uploading a file to S3)
const PENDING Status = "PENDING"

// UPLOADED Status indicates that the image has been uploaded to S3
const UPLOADED Status = "UPLOADED"

// COMPLETE Status indicates that the Image file exists and has been fully analyzed
const COMPLETE Status = "COMPLETE"

// ERROR Status indicates that an error occurred processing the Image
const ERROR Status = "ERROR"

// Statuses is the complete list of valid Image statuses
var Statuses = []Status{PENDING, UPLOADED, COMPLETE, ERROR}

// IsValid returns true if the supplied Status is recognized
func (s Status) IsValid() bool {
	for _, v := range Statuses {
		if s == v {
			return true
		}
	}
	return false
}

// String returns a string representation of the Status
func (s Status) String() string {
	return string(s)
}
