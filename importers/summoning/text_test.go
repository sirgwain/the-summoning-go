package summoning

import (
	"reflect"
	"testing"
)

func TestReadText(t *testing.T) {
	// Code page 437 FEh is a small square; END is dropped.
	data := container(textRecord("Hello"), textRecord("v1 \xFE x"), textRecord("END"))
	got, err := ReadText(data)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Hello", "v1 ■ x"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
