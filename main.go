package main

import (
	"fmt"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
)

func main() {

	// consumer("/ndn/coba")

	// //Serve /hello interest
	go serve_hello("R1")

	// //hello protocol every 5 second
	go consum_hello(5)
}

func serve_hello(router string) {
	producer("/hello", router)
}

func consum_hello(delay int) {
	interval := 5 * time.Second
	for {
		//update facelist
		update_facelist()
		//create route
		for k, v := range facelist {
			register_route(v.tkn, 0, int(k))

			//send hello interest to every face
			interest := ndn.MakeInterest(ndn.ParseName("/hello", ndn.ForwardingHint{ndn.ParseName(v.tkn)}))

			data, rtt, thg, e := consumer_interest(interest)

			if e != nil {
				continue
			}

			v.ngb = data
			v.rtt = rtt
			v.thg = thg
			facelist[k] = v
		}
		fmt.Println(facelist)

		time.Sleep(interval)

	}
}
