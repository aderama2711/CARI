package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/nfdmgmt"
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

	// go producer("hello", "Hello World!", 10)

	// data, _, _, e := consumer("hello")
	// fmt.Println(data)
	// fmt.Println(e)

	wg.Wait()

}

func serve_hello(router string) {
	producer("hello", router, 10)
}

func consume_hello(delay time.Duration) {
	var (
		client mgmt.Client
		face   mgmt.Face
		fwFace l3.FwFace
	)

	client, e := nfdmgmt.New()

	face, e = client.OpenFace()
	if e != nil {
		fmt.Println(e)
	}
	l3face := face.Face()

	fw := l3.GetDefaultForwarder()
	if fwFace, e = fw.AddFace(l3face); e != nil {
		fmt.Println(e)
	}
	fwFace.AddRoute(ndn.Name{})
	fw.AddReadvertiseDestination(face)

	log.Printf("uplink opened, state is %s", l3face.State())
	l3face.OnStateChange(func(st l3.TransportState) {
		log.Printf("uplink state changes to %s", l3face.State())
	})

	interval := delay * time.Second
	interval_interest := 50 * time.Millisecond
	for {
		//update facelist
		update_facelist()
		fmt.Println(facelist)

		//create route
		for k, v := range facelist {
			register_route(v.tkn, 0, int(k))

			fmt.Println(k, v.tkn)
			//send hello interest to every face
			interest := ndn.MakeInterest(ndn.ParseName("hello"), ndn.ForwardingHint{ndn.ParseName(v.tkn), ndn.ParseName("hello")})

			data, rtt, thg, e := consumer_interest(interest)

			if e != nil {
				continue
			}

			fmt.Println(data)

			v.ngb = data
			v.rtt = rtt
			v.thg = thg
			facelist[k] = v

			time.Sleep(interval_interest)
		}
		fmt.Println(facelist)

		time.Sleep(interval)
	}
}
