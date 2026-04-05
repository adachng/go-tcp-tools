package main

import (
	"fmt"
	"sync"
)

type a struct {
}

func (aa *a) String() string {
	return "ABC"
}

func main() {
	fun1 := func(i int) {
		fmt.Printf("Hi: %d\n", i)
	}
	fun2 := func() {
		fun1(1)
	}

	var once sync.Once

	defer once.Do(fun2)

	fun2 = func() {
		fun1(2)
	}

	// Toggle comment the line below.
	// defer once.Do(fun2)

	defer fun1(3)
	defer fun1(4)

	var b a
	fmt.Print(b.String())
}
