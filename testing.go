package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
	b64 "encoding/base64"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/nfdmgmt"
)

type faces struct {
	Ngb  string  `json:"Ngb"`
	Rtt  float64 `json:"Rtt"`
	Thg  float64 `json:"Thg"`
	Tkn  string  `json:"Tkn"`
	N_oi uint64  `json:"N_oi"`
	N_in uint64  `json:"N_in"`
}

var facelist map[uint64]faces

var mutex sync.Mutex

func main() {

	// consumer("/ndn/coba")

	// //Serve /hello interest
	// time.Sleep(1 * time.Second)
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

	// Testing hello
	interest := ndn.MakeInterest(ndn.ParseName("hello"))

	data, _, _, err := consumer_interest(interest)
	fmt.Println(data)
	fmt.Println(err)

	// Testing facelist
	interest := ndn.MakeInterest(ndn.ParseName("hello"))

	data, _, _, err := consumer_interest(interest)
	fmt.Println(data)
	fmt.Println(err)

	// Testing update route
	interest := ndn.MakeInterest(ndn.ParseName("update"), []byte("testing"))
	interest.MustBeFresh = true
	interest.UpdateParamsDigest() //Update SHA256 params

	data, _, _, err := consumer_interest(interest)
	fmt.Println(data)
	fmt.Println(err)

}

func consumer_interest(Interest ndn.Interest) (content string, Rtt float64, Thg float64, e error) {
	// seqNum := rand.Uint64()
	// var nData, nErrors atomic.Int64

	t0 := time.Now()

	data, e := endpoint.Consume(context.Background(), Interest,
		endpoint.ConsumerOptions{})

	raw_Rtt := time.Since(t0)

	Rtt = float64(raw_Rtt / time.Millisecond)

	fmt.Println(Rtt)

	if e == nil {
		// nDataL, nErrorsL := nData.Add(1), nErrors.Load()
		// fmt.Println(data.Content)
		content = string(data.Content[:])
		// fmt.Printf("%6.2f%% D %s\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), content)
		if Rtt != 0 {
			Thg = float64(len(content)) / float64(Rtt/1000)
		} else {
			Thg = 0
		}

	} else {
		// nDataL, nErrorsL := nData.Load(), nErrors.Add(1)
		// fmt.Printf("%6.2f%% E %v\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), e)
		return content, 0, 0, e
	}

	return content, Rtt, Thg, nil
}