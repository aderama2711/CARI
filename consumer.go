package main

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
)

func main() {
	openUplink()
	// seqNum := rand.Uint64()
	for {
		var nData, nErrors atomic.Int64

		name := ndn.ParseName("/ndn/coba")

		data, e := endpoint.Consume(context.Background(), ndn.MakeInterest(name),
			endpoint.ConsumerOptions{})

		if e == nil {
			nDataL, nErrorsL := nData.Add(1), nErrors.Load()
			fmt.Println(data.Content)
			fmt.Printf("%6.2f%% D %6dus\n", 100*float64(nDataL)/float64(nDataL+nErrorsL))
		} else {
			nDataL, nErrorsL := nData.Load(), nErrors.Add(1)
			fmt.Printf("%6.2f%% E %v\n", 100*float64(nDataL)/float64(nDataL+nErrorsL), e)
		}
	}

}
