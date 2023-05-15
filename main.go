package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
)

func main() {
	openUplink()

	var wg sync.WaitGroup

	wg.Add(1)

	// consumer("/ndn/coba")

	// //Serve /hello interest
	go serve_hello("R1")

	// //hello protocol every 5 second
	go consum_hello(5)

	wg.Wait()
}

func serve_hello(router string) {
	for {
		producer("/hello", router)
	}

}

func consum_hello(delay int) {
	var FH []ndn.Name
	interval := 5 * time.Second
	for {
		//update facelist
		update_facelist()
		//create route
		for k, v := range facelist {
			register_route(v.tkn, 0, int(k))

			fmt.Println(k, v.tkn)

			FH = append(FH, ndn.ParseName(v.tkn))
			FH = append(FH, ndn.ParseName("/hello"))

			//send hello interest to every face
			interest := ndn.MakeInterest(ndn.ParseName("/hello"), ndn.ForwardingHint{FH})

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
