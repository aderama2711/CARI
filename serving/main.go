package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
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

	if len(os.Args) < 1 {
		fmt.Println("Usage: main.go routerID")
		return
	}

	routerID := os.Args[1]

	wg.Add(1)

	producer("hello", routerID, 100)

	wg.Wait()

}

func producer(name string, content string, fresh int) {
	asciicontent := ""
	data := ""

	for _, char := range content {
		asciicontent += fmt.Sprintf("%d", char)
	}

	// Stuffing
	fmt.Println(len(asciicontent))

	if len(asciicontent) < 8192 {
		data = strings.Repeat("A", 8192-len(asciicontent))
		data = asciicontent + data
	} else {
		data = asciicontent
	}

	payload := []byte(data)
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
