package time

import (
	"fmt"
	"testing"
)

func TestUnMarshalText(t *testing.T) {
	dummyTime := Time{}

	standardTimeFormats := []string{
		"2006-01-02T15:04:05MST",
		"2006-01-02T15:04:05UTC",
		"2006-01-02T15:04:05PST",
		"2006-01-02T15:04:05Z",
	}
	for _, goodTime := range standardTimeFormats {
		if dummyTime.UnmarshalText([]byte(goodTime)) != nil {
			t.Error("expected " + goodTime + " to be marshal-able.")
		}
		if dummyTime.Year() != 2006 {
			t.Error(fmt.Sprintf("expected year to be 2006; got %d.", dummyTime.Year()))
		}
	}

	alternateTimeFormats := []string{
		"2007-01-02T15:04:05",
		"2007-01-02T15:04:05",
		"2007-01-02T15:04:05",
	}

	for _, goodTime := range alternateTimeFormats {
		if dummyTime.UnmarshalText([]byte(goodTime)) != nil {
			t.Error("expected " + goodTime + " to be marshal-able.")
		}
		if dummyTime.Year() != 2007 {
			t.Error(fmt.Sprintf("expected year to be 2007; got %d.", dummyTime.Year()))
		}
	}

	badTime := "this is a bad time, isn't it?"
	if dummyTime.UnmarshalText([]byte(badTime)) == nil {
		t.Error("expected " + badTime + "to be unmarshal-able.")
	}
}
