package main

import (
	"crypto/rand"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

func main() {

	var sigNonce [8]byte
	rand.Read(sigNonce[:])
	name := ndn.ParseName("/localhost/nfd/faces/list")
	name = append(name, ndn.NameComponentFrom(an.TtGenericNameComponent))
	interest := ndn.Interest{
		Name:        name,
		MustBeFresh: true,
		SigInfo: &ndn.SigInfo{
			Nonce: sigNonce[:],
			Time:  uint64(time.Now().UnixMilli()),
		}}

	Signer := ndn.DigestSigning

	Signer.Sign(&interest)

	consumer_interest(interest)

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
