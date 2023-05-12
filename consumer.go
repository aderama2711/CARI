package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
)

func main() {
	for {
		seqNum := rand.Uint64()

		var nData, nErrors atomic.Int64

		interest := ndn.MakeInterest(fmt.Sprintf("/ndn/coba/%016X", seqNum), ndn.MustBeFreshFlag)

		_, e := endpoint.Consume(context.Background(), interest, endpoint.ConsumerOptions{})

		if e == nil {
			nDataL, nErrorsL := nData.Add(1), nErrors.Load()
			fmt.Printf("%6.2f%% D %016X %6dus", 100*float64(nDataL)/float64(nDataL+nErrorsL), seqNum)
		} else {
			nDataL, nErrorsL := nData.Load(), nErrors.Add(1)
			fmt.Printf("%6.2f%% E %016X %v", 100*float64(nDataL)/float64(nDataL+nErrorsL), seqNum, e)
		}
	}

}
