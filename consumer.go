package main

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
)

func consumer(name string) {
	openUplink()
	// seqNum := rand.Uint64()
	var nData, nErrors atomic.Int64

	interest := ndn.ParseName(name)

	data, e := endpoint.Consume(context.Background(), ndn.MakeInterest(interest),
		endpoint.ConsumerOptions{})

	if e == nil {
		nDataL, nErrorsL := nData.Add(1), nErrors.Load()
		fmt.Println(data.Content)
		content := string(data.Content[:])
		fmt.Printf("%6.2f%% D %s\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), content)
	} else {
		nDataL, nErrorsL := nData.Load(), nErrors.Add(1)
		fmt.Printf("%6.2f%% E %v\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), e)
	}

}
