package main

import (
	"context"
	"fmt"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/nfdmgmt"
)

func main() {

	c, _ := nfdmgmt.New()

	cr, e := c.Invoke(context.TODO(), nfdmgmt.RibRegisterCommand{
		Name:   ndn.ParseName("/"),
		Origin: 0,
		Cost:   0,
		FaceId: 289,
	})

	if e != nil {
		fmt.Println(e)
	}
	if cr.StatusCode != 200 {
		fmt.Println("unexpected response status %d", cr.StatusCode)
	}

	// var sigNonce [8]byte
	// rand.Read(sigNonce[:])

	// interest := ndn.Interest{
	// 	Name:        ndn.ParseName("/localhost/nfd/faces/list"),
	// 	MustBeFresh: true,
	// 	CanBePrefix: true,
	// 	SigInfo: &ndn.SigInfo{
	// 		Nonce: sigNonce[:],
	// 		Time:  uint64(time.Now().UnixMilli()),
	// 	},
	// }

	// c.Signer.Sign(&interest)

	// data, e := endpoint.Consume(context.Background(), interest,
	// 	endpoint.ConsumerOptions{})

	// if e != nil {
	// 	fmt.Println(e)
	// } else {
	// 	parse_facelist(data.Content)
	// }

	// consumer("/ndn/coba")

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
