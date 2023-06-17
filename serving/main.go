package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/nfdmgmt"
)

func main() {
	var wg sync.WaitGroup

	wg.Add(1)

	// consumer("/ndn/coba")

	// //Serve /hello interest
	producer("hello", "R1", 10)

	// go producer("hello", "Hello World!", 10)

	// time.Sleep(1 * time.Second)

	// var (
	// 	client mgmt.Client
	// 	face   mgmt.Face
	// 	fwFace l3.FwFace
	// )

	// client, e := nfdmgmt.New()

	// face, e = client.OpenFace()
	// if e != nil {
	// 	fmt.Println(e)
	// }
	// l3face := face.Face()

	// fw := l3.GetDefaultForwarder()
	// if fwFace, e = fw.AddFace(l3face); e != nil {
	// 	fmt.Println(e)
	// }
	// fwFace.AddRoute(ndn.Name{})
	// fw.AddReadvertiseDestination(face)

	// log.Printf("uplink opened, state is %s", l3face.State())
	// l3face.OnStateChange(func(st l3.TransportState) {
	// 	log.Printf("uplink state changes to %s", l3face.State())
	// })

	// data, _, _, e := consumer("hello")
	// fmt.Println(data)
	// fmt.Println(e)

	wg.Wait()

}

func producer(name string, content string, fresh int) {
	payload := []byte(content)
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

	var signer ndn.Signer

	for {
		ctx := context.Background()
		p, e := endpoint.Produce(ctx, endpoint.ProducerOptions{
			Prefix:      ndn.ParseName(name),
			NoAdvertise: false,
			Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
				// fmt.Println(interest)
				return ndn.MakeData(interest, payload, time.Duration(fresh)*time.Millisecond), nil
			},
			DataSigner: signer,
		})

		if e != nil {
			fmt.Println(e)
		}

		<-ctx.Done()
		defer p.Close()
	}

}
