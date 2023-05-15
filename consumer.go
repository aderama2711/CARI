package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt/nfdmgmt"
)

func consumer(name string) (content string, rtt float64, thg float64, e error) {
	openUplink()
	// seqNum := rand.Uint64()
	var nData, nErrors atomic.Int64

	interest := ndn.ParseName(name)

	t0 := time.Now()

	data, e := endpoint.Consume(context.Background(), ndn.MakeInterest(interest),
		endpoint.ConsumerOptions{})

	rtt = float64(time.Since(t0).Milliseconds())

	if e == nil {
		nDataL, nErrorsL := nData.Add(1), nErrors.Load()
		fmt.Println(data.Content)
		content = string(data.Content[:])
		fmt.Printf("%6.2f%% D %s\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), content)
		thg = float64(len(content)) / float64(rtt/1000)
	} else {
		nDataL, nErrorsL := nData.Load(), nErrors.Add(1)
		fmt.Printf("%6.2f%% E %v\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), e)
		return content, 0, 0, e
	}

	return content, rtt, thg, nil
}

func consumer_interest(Interest ndn.Interest) (content string, rtt float64, thg float64, e error) {
	openUplink()
	// seqNum := rand.Uint64()
	var nData, nErrors atomic.Int64

	t0 := time.Now()

	data, e := endpoint.Consume(context.Background(), Interest,
		endpoint.ConsumerOptions{})

	rtt = float64(time.Since(t0).Milliseconds())

	if e == nil {
		nDataL, nErrorsL := nData.Add(1), nErrors.Load()
		fmt.Println(data.Content)
		content = string(data.Content[:])
		fmt.Printf("%6.2f%% D %s\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), content)
		thg = float64(len(content)) / float64(rtt/1000)
	} else {
		nDataL, nErrorsL := nData.Load(), nErrors.Add(1)
		fmt.Printf("%6.2f%% E %v\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), e)
		return content, 0, 0, e
	}

	return content, rtt, thg, nil
}

func update_facelist() {
	openUplink()
	c, _ := nfdmgmt.New()

	var sigNonce [8]byte
	rand.Read(sigNonce[:])

	interest := ndn.Interest{
		Name:        ndn.ParseName("/localhost/nfd/faces/list"),
		MustBeFresh: true,
		CanBePrefix: true,
		SigInfo: &ndn.SigInfo{
			Nonce: sigNonce[:],
			Time:  uint64(time.Now().UnixMilli()),
		},
	}

	c.Signer.Sign(&interest)

	data, e := endpoint.Consume(context.Background(), interest,
		endpoint.ConsumerOptions{})

	if e != nil {
		fmt.Println(e)
	} else {
		parse_facelist(data.Content)
	}
}

func register_route(name string, cost int, faceid int) {
	openUplink()

	c, _ := nfdmgmt.New()

	cr, e := c.Invoke(context.TODO(), nfdmgmt.RibRegisterCommand{
		Name:   ndn.ParseName(name),
		Origin: 0,
		Cost:   cost,
		FaceID: faceid,
	})

	if e != nil {
		fmt.Println(e)
	}
	if cr.StatusCode != 200 {
		fmt.Println("unexpected response status %d", cr.StatusCode)
	} else {
		fmt.Println("Route registered")
	}
}
