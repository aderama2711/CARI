package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/gqlmgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/nfdmgmt"
)

var (
	gqlserver string
	mtuFlag   int
	useNfd    bool
	enableLog bool

	client mgmt.Client
	face   mgmt.Face
	fwFace l3.FwFace
)

func openUplink() (e error) {
	client, e = nfdmgmt.New()

	switch client := client.(type) {
	case *gqlmgmt.Client:
		var loc memiftransport.Locator
		loc.Dataroom = mtuFlag
		face, e = client.OpenMemif(loc)
	default:
		face, e = client.OpenFace()
	}
	if e != nil {
		return e
	}
	l3face := face.Face()

	fw := l3.GetDefaultForwarder()
	if fwFace, e = fw.AddFace(l3face); e != nil {
		return e
	}
	fwFace.AddRoute(ndn.Name{})
	fw.AddReadvertiseDestination(face)

	log.Printf("uplink opened, state is %s", l3face.State())
	l3face.OnStateChange(func(st l3.TransportState) {
		log.Printf("uplink state changes to %s", l3face.State())
	})
	return nil
}

func main() {
	openUplink()
	payload := make([]byte, 1024)
	rand.New(rand.NewSource(rand.Int63())).Read(payload)

	p, e := endpoint.Produce(context.Background(), endpoint.ProducerOptions{
		Prefix:      ndn.ParseName("/ndn/coba"),
		NoAdvertise: false,
		Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
			fmt.Print(interest)
			return ndn.MakeData(interest, payload), nil
		},
	})

	if e != nil {
		fmt.Print(e)
	}

	defer p.Close()

}
