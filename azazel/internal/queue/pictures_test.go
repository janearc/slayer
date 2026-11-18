package queue

import "testing"

// pictureRequest is a complete request whose pictures field is given.
func pictureRequest(t *testing.T, pictures *int64) Request {
	return Request{Agent: "studio", Key: "pictures", Command: "/usr/bin/true",
		Args: []string{}, Directory: t.TempDir(), Lane: "gpu",
		TimeoutSeconds: 10, Pictures: pictures}
}

// TestPicturesMustBePositiveWhenGiven refuses a count of none or fewer.
func TestPicturesMustBePositiveWhenGiven(t *testing.T) {
	for _, given := range []int64{0, -1} {
		if err := pictureRequest(t, &given).validate(); err == nil {
			t.Fatalf("pictures %d accepted", given)
		}
	}
}

// TestPicturesAbsentOrPositiveAreAccepted is the anti-test: an absent
// count and a positive one both pass.
func TestPicturesAbsentOrPositiveAreAccepted(t *testing.T) {
	positive := int64(2005) // picked: the frames of a heard v7 film
	for _, given := range []*int64{nil, &positive} {
		if err := pictureRequest(t, given).validate(); err != nil {
			t.Fatal(given, err)
		}
	}
}
