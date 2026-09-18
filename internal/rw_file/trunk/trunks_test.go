package trunk

import (
	"errors"
	"testing"
	"time"
)

// P0-2 — the bitmap is sized by capacity, so a bigger file has to be refused
// rather than indexed.
//
// Note that this is deliberately file free: the guard has to run before anything
// touches the cache file, and a File with a nil file would panic if it did not.
// Moving the check below the state writes therefore turns this test into a
// crash rather than a quiet pass.
func TestInitRejectsContentLargerThanTheBitmap(t *testing.T) {
	file := &File{}

	if got := MaxSize(); got <= 0 {
		t.Fatalf("MaxSize() = %d, want a positive limit", got)
	}

	err := file.Init(MaxSize()+1, time.Time{})
	if !errors.Is(err, ErrContentTooLarge) {
		t.Fatalf("Init(MaxSize()+1) = %v, want ErrContentTooLarge", err)
	}
}

// The clamp is the second line of defence: without it, any fragment past the
// bitmap indexes out of range.
func TestTrunkIndexRangeIsClampedToTheBitmap(t *testing.T) {
	const bitmap = 8

	file := &File{trunks: make([]byte, bitmap)}

	tests := []struct {
		name      string
		frag      *Fragment
		wantStart int64
		wantEnd   int64
	}{
		{"inside the bitmap", NewFragment(bytesPerTrunk*2, bytesPerTrunk*3), 2, 5},
		{"ending exactly on the boundary", NewFragment(0, bytesPerTrunk*bitmap), 0, bitmap},
		{"straddling the end", NewFragment(bytesPerTrunk*6, bytesPerTrunk*10), 6, bitmap},
		{"entirely past the end", NewFragment(bytesPerTrunk*100, bytesPerTrunk*4), 100, bitmap},
		{"empty fragment", NewFragment(bytesPerTrunk*3, 0), 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := file.trunkIndexRange(tt.frag)

			if start != tt.wantStart || end != tt.wantEnd {
				t.Fatalf("trunkIndexRange = (%d, %d), want (%d, %d)",
					start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

// FillTrunks is the call the download loop made for a file above capacity: it
// used to index past the end of the bitmap and panic. The fragment here covers
// nothing the bitmap can hold, so the clamped range is empty and the call has to
// do nothing at all — including not reaching writeTrunks, which is why a nil
// cache file is fine.
func TestFillTrunksPastTheBitmapIsANoOp(t *testing.T) {
	file := &File{trunks: make([]byte, 8)}

	file.FillTrunks(NewFragment(bytesPerTrunk*100, bytesPerTrunk*4))

	for i, b := range file.trunks {
		if b != 0 {
			t.Fatalf("trunks[%d] = %d, want 0: the fragment is outside the bitmap", i, b)
		}
	}
}
