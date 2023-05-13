package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/nfdmgmt"
)

func main() {

	get_face()

	// //Serve /hello interest
	// go serve_hello("R1")

	// //hello protocol every 5 second
	// go consum_hello(5)
}

func serve_hello(router string) {
	producer("/hello", router)
}

func consum_hello(delay int) {
	interval := 5 * time.Second
	for {
		consumer("/hello")
		time.Sleep(interval)
	}
}

func get_face() {
	client, e := nfdmgmt.New()
	if e != nil {
		fmt.Println(e)
	}

	client.Prefix = ndn.ParseName("/localhost/nfd")

	client.Inv

	cr, e := client.Client.Invoke(context.Background(), ndn.ParseName("/face/list"))
	if e != nil {
		log.Printf("%v", e)
	} else {
		log.Printf("%d", cr.StatusCode)
	}
}
