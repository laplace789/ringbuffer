package ringbuffer

import "testing"

func FuzzRingStateMachine(f *testing.F) {
	f.Add(byte(0), []byte{0, 1, 2, 3})
	f.Add(byte(3), []byte{0, 0, 1, 2, 2, 3, 4})
	f.Add(byte(15), []byte{3, 1, 0, 1, 0, 1, 2, 3})

	f.Fuzz(func(t *testing.T, capacityByte byte, operations []byte) {
		capacity := int(capacityByte%16) + 1
		rb := mustNew[int](t, capacity)
		model := make([]int, 0, rb.Capacity())
		writeReserved := false
		readReserved := false
		writeValue := 0

		for step, operation := range operations {
			switch operation % 5 {
			case 0: // Reserve a write slot.
				idx, slot, err := rb.Set()
				switch {
				case writeReserved:
					if err != ErrWriteReservationActive || slot != nil || idx != 0 {
						t.Fatalf("step %d: repeated Set() = (%d, %p, %v)", step, idx, slot, err)
					}
				case uint64(len(model)) == rb.Capacity():
					if err != ErrRingFull || slot != nil || idx != 0 {
						t.Fatalf("step %d: full Set() = (%d, %p, %v)", step, idx, slot, err)
					}
				default:
					if err != nil || slot == nil {
						t.Fatalf("step %d: Set() = (%d, %p, %v)", step, idx, slot, err)
					}
					if want := rb.wp & rb.mask; idx != want {
						t.Fatalf("step %d: Set() index = %d, want %d", step, idx, want)
					}
					writeValue = step*256 + int(operation)
					*slot = writeValue
					writeReserved = true
				}

			case 1: // Commit a write reservation.
				err := rb.SetAdv()
				if !writeReserved {
					if err != ErrNoWriteReservation {
						t.Fatalf("step %d: SetAdv() error = %v, want %v", step, err, ErrNoWriteReservation)
					}
				} else {
					if err != nil {
						t.Fatalf("step %d: SetAdv(): %v", step, err)
					}
					model = append(model, writeValue)
					writeReserved = false
				}

			case 2: // Reserve a read slot.
				idx, slot, err := rb.Get()
				switch {
				case readReserved:
					if err != ErrReadReservationActive || slot != nil || idx != 0 {
						t.Fatalf("step %d: repeated Get() = (%d, %p, %v)", step, idx, slot, err)
					}
				case len(model) == 0:
					if err != ErrRingEmpty || slot != nil || idx != 0 {
						t.Fatalf("step %d: empty Get() = (%d, %p, %v)", step, idx, slot, err)
					}
				default:
					if err != nil || slot == nil {
						t.Fatalf("step %d: Get() = (%d, %p, %v)", step, idx, slot, err)
					}
					if want := rb.rp & rb.mask; idx != want {
						t.Fatalf("step %d: Get() index = %d, want %d", step, idx, want)
					}
					if *slot != model[0] {
						t.Fatalf("step %d: Get() value = %d, want %d", step, *slot, model[0])
					}
					readReserved = true
				}

			case 3: // Release a read reservation.
				err := rb.GetAdv()
				if !readReserved {
					if err != ErrNoReadReservation {
						t.Fatalf("step %d: GetAdv() error = %v, want %v", step, err, ErrNoReadReservation)
					}
				} else {
					if err != nil {
						t.Fatalf("step %d: GetAdv(): %v", step, err)
					}
					model = model[1:]
					readReserved = false
				}

			case 4: // Reset cancels both reservations and empties the queue.
				rb.Reset()
				model = model[:0]
				writeReserved = false
				readReserved = false
			}

			if got := committedLen(rb); got != uint64(len(model)) {
				t.Fatalf("step %d: committed count = %d, model length = %d", step, got, len(model))
			}
			if committedLen(rb) > rb.Capacity() {
				t.Fatalf("step %d: committed count exceeds capacity", step)
			}
			if rb.writeReserved != writeReserved || rb.readReserved != readReserved {
				t.Fatalf("step %d: reservation state mismatch: write=%v/%v read=%v/%v",
					step, rb.writeReserved, writeReserved, rb.readReserved, readReserved)
			}
		}
	})
}
