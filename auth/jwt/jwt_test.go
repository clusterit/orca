package jwt

import "bytes"

import "testing"

var googleTestData = `
{
 "kind": "plus#person",
 "etag": "a tag",
 "occupation": "blubber",
 "gender": "male",
 "emails": [
  {
   "value": "max.mustermann@gmail.com",
   "type": "account"
  }
 ],
 "urls": [
  {
   "value": "https://github.com/clusterit",
   "type": "otherProfile",
   "label": "Github"
  }
 ],
 "objectType": "person",
 "id": "4711",
 "displayName": "Max Mustermann",
 "name": {
  "familyName": "Mustermann",
  "givenName": "Max"
 },
 "relationshipStatus": "married",
 "url": "https://plus.google.com/+MaxMustermann",
 "image": {
  "url": "https://www.google.de/images/nav_logo195.png",
  "isDefault": false
 },
 "isPlusUser": true,
 "language": "de",
 "circledByCount": 0,
 "verified": false,
 "cover": {
  "layout": "banner",
  "coverPhoto": {
   "url": "http://ichglotz.tv/desktop/testbild/1920x1200.jpg",
   "height": 705,
   "width": 940
  },
  "coverInfo": {
   "topImageOffset": 0,
   "leftImageOffset": 0
  }
 }
}`

func TestJsonPathGoogleData(t *testing.T) {
	r := bytes.NewBufferString(googleTestData)
	m, err := parse(r)
	if err != nil {
		t.Errorf("cannot parse testdata:%s", err)
	}
	testvalues := [][]string{
		[]string{"displayName", "Max Mustermann"},
		[]string{"emails[0].value", "max.mustermann@gmail.com"},
		[]string{"cover.coverPhoto.url", "http://ichglotz.tv/desktop/testbild/1920x1200.jpg"},
		[]string{"image.url", "https://www.google.de/images/nav_logo195.png"},
	}
	for _, tv := range testvalues {
		v, err := getValue(tv[0], m)
		if err != nil {
			t.Errorf("cannot read %s:%s", tv[0], err)
		}
		if v != tv[1] {
			t.Errorf("wrong value read: '%s' should be '%s'", v, tv[1])
		}
	}
}
