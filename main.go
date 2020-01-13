package main

import "log"

func main() {
	d, err := NewDevelop()
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(d.Start())
}
