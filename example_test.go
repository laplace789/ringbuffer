package ringbuffer_test

import (
	"fmt"
	"log"

	"github.com/laplace789/ringbuffer"
)

func Example() {
	rb, err := ringbuffer.New[int](3)
	if err != nil {
		log.Fatal(err)
	}

	_, writeSlot, err := rb.Set()
	if err != nil {
		log.Fatal(err)
	}
	*writeSlot = 42
	if err := rb.SetAdv(); err != nil {
		log.Fatal(err)
	}

	_, readSlot, err := rb.Get()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(*readSlot)
	if err := rb.GetAdv(); err != nil {
		log.Fatal(err)
	}

	fmt.Println(rb.Capacity())
	// Output:
	// 42
	// 4
}

func ExampleRing_struct() {
	type message struct {
		ID   int
		Text string
	}

	rb, err := ringbuffer.New[message](8)
	if err != nil {
		log.Fatal(err)
	}

	_, writeSlot, err := rb.Set()
	if err != nil {
		log.Fatal(err)
	}
	*writeSlot = message{ID: 7, Text: "ready"}
	if err := rb.SetAdv(); err != nil {
		log.Fatal(err)
	}

	_, readSlot, err := rb.Get()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d: %s\n", readSlot.ID, readSlot.Text)
	if err := rb.GetAdv(); err != nil {
		log.Fatal(err)
	}
	// Output: 7: ready
}
