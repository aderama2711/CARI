package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
)

func main() {
	for {
		var nData, nErrors atomic.Int64

		p, e := endpoint.Consume(context.Background(), ndn.MakeInterest("/ndn/coba", 200*time.Millisecond),
			endpoint.ConsumerOptions{})

		if e == nil {
			nDataL, nErrorsL := nData.Add(1), nErrors.Load()
			fmt.Printf("%6.2f%% D %6dus", 100*float64(nDataL)/float64(nDataL+nErrorsL))
		} else {
			nDataL, nErrorsL := nData.Load(), nErrors.Add(1)
			fmt.Printf("%6.2f%% E %v", 100*float64(nDataL)/float64(nDataL+nErrorsL), e)
		}

		defer p.Close()
	}

}
