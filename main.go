package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
)

func main() {
	facelist = make(map[uint64]faces)

	var wg sync.WaitGroup

	wg.Add(1)

	// consumer("/ndn/coba")

	// //Serve /hello interest
	go serve_hello("R1")

	// //hello protocol every 5 second
	go consume_hello(5)

	wg.Wait()
}

func serve_hello(router string) {
	openUplink()
	producer("/hello", router, 10)
}

func consume_hello(delay time.Duration) {
	openUplink()
	interval := delay * time.Second
	for {
		//update facelist
		update_facelist()
		fmt.Println(facelist)

		//create route
		for k, v := range facelist {
			register_route(v.tkn, 0, int(k))

			fmt.Println(k, v.tkn)
			//send hello interest to every face
			interest := ndn.MakeInterest(ndn.ParseName("/hello"), ndn.ForwardingHint{ndn.ParseName(v.tkn), ndn.ParseName("/hello")}, ndn.tMustBeFresh{}, ndn.tCanBePrefix{})

			data, rtt, thg, e := consumer_interest(interest)

			if e != nil {
				continue
			}

			fmt.Println(data)

			v.ngb = data
			v.rtt = rtt
			v.thg = thg
			facelist[k] = v
		}
		fmt.Println(facelist)

		time.Sleep(interval)
	}
}
