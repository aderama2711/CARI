package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/nfdmgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func main() {
	cr := nfdmgmt.ControlResponse
	var sigNonce [8]byte
	rand.Read(sigNonce[:])
	name := ndn.ParseName("/localhost/nfd/faces/list")
	interest := ndn.Interest{
		Name:        name,
		MustBeFresh: true,
		SigInfo: &ndn.SigInfo{
			Nonce: sigNonce[:],
			Time:  uint64(time.Now().UnixMilli()),
		}}

	Signer := ndn.DigestSigning

	Signer.Sign(&interest)

	data, e := endpoint.Consume(context.Background(), interest, endpoint.ConsumerOptions{})
	if e != nil {
		fmt.Println("consumer error: %w", e)
	}

	e = tlv.Decode(data.Content, &cr)
	fmt.Println(e)

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
