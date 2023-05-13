package main

import (
	"crypto/rand"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
)

func main() {

	var sigNonce [8]byte
	rand.Read(sigNonce[:])
	consumer_interest(ndn.Interest{
		Name:        "/localhost/nfd/faces/list",
		MustBeFresh: true,
		SigInfo: &ndn.SigInfo{
			Nonce: sigNonce[:],
			Time:  uint64(time.Now().UnixMilli()),
		}})

	// consumer("/localhost/nfd/face/list")

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
