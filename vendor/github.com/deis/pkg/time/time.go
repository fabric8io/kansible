package time

import "time"

// DeisDatetimeFormat is the standard date/time representation used in Deis.
const DeisDatetimeFormat = "2006-01-02T15:04:05MST"

// Different format to deal with the pyopenssl formatting
// http://www.pyopenssl.org/en/stable/api/crypto.html#OpenSSL.crypto.X509.get_notAfter
const PyOpenSSLTimeDateTimeFormat = "2006-01-02T15:04:05"

// Time represents the standard datetime format used across the Deis Platform.
type Time struct {
	*time.Time
}

func (t *Time) format() string {
	return t.Format(DeisDatetimeFormat)
}

// MarshalJSON implements the json.Marshaler interface.
// The time is a quoted string in Deis' datetime format.
func (t *Time) MarshalJSON() ([]byte, error) {
	return []byte(t.Format(`"` + DeisDatetimeFormat + `"`)), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
// The time is expected to be in Deis' datetime format.
func (t *Time) UnmarshalText(data []byte) error {
	tt, err := time.Parse(time.RFC3339, string(data))
	if _, ok := err.(*time.ParseError); ok {
		tt, err = time.Parse(DeisDatetimeFormat, string(data))
		if _, ok := err.(*time.ParseError); ok {
			tt, err = time.Parse(PyOpenSSLTimeDateTimeFormat, string(data))
		}
	}
	*t = Time{&tt}
	return err
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// The time is expected to be a quoted string in Deis' datetime format.
func (t *Time) UnmarshalJSON(data []byte) error {
	// Fractional seconds are handled implicitly by Parse.
	tt, err := time.Parse(`"`+time.RFC3339+`"`, string(data))
	if _, ok := err.(*time.ParseError); ok {
		tt, err = time.Parse(`"`+DeisDatetimeFormat+`"`, string(data))
		if _, ok := err.(*time.ParseError); ok {
			tt, err = time.Parse(`"`+PyOpenSSLTimeDateTimeFormat+`"`, string(data))
		}
	}
	*t = Time{&tt}
	return err
}
