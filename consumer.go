package main

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/eric135/go-ndn"
	"github.com/eric135/go-ndn/endpoint"
)

func main() {
	for {
		var nData, nErrors atomic.Int64

		_, e := endpoint.Consume(context.Background(), go-ndn.MakeInterest("/ndn/coba"),
			endpoint.ConsumerOptions{})

		if e == nil {
			nDataL, nErrorsL := nData.Add(1), nErrors.Load()
			fmt.Printf("%6.2f%% D %6dus", 100*float64(nDataL)/float64(nDataL+nErrorsL))
		} else {
			nDataL, nErrorsL := nData.Load(), nErrors.Add(1)
			fmt.Printf("%6.2f%% E %v", 100*float64(nDataL)/float64(nDataL+nErrorsL), e)
		}
	}

}
