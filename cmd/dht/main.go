package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/network"
	peerstore "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/multiformats/go-multiaddr"
)

func main() {
	// Get Args
	ip, port, dest, refresh := "", "", "", false
	flag.StringVar(&ip, "ip", "127.0.0.1", "ip")
	flag.StringVar(&port, "port", "8080", "port")
	flag.StringVar(&dest, "dest", "", "destination Addr")
	flag.BoolVar(&refresh, "refresh", false, "refresh")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup Host
	host, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/%s/tcp/%s", ip, port)),
	)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Listen Addr: %+v\n", host.Addrs())

	// Setup DHT
	dht, err := NewDHT(ctx, host)
	if err != nil {
		panic(err)
	}
	dht.RoutingTable().PeerAdded = func(pid peerstore.ID) {
		isConnected := host.Network().Connectedness(pid) == network.Connected
		fmt.Printf("New Peer Added: %+v already connected => %t\n", pid, isConnected)
	}
	dht.RoutingTable().PeerRemoved = func(pid peerstore.ID) {
		isConnected := host.Network().Connectedness(pid) == network.Connected
		fmt.Printf("New Peer Removed: %+v already connected => %t\n", pid, isConnected)
	}

	// Get local addr
	addrs, err := peerstore.AddrInfoToP2pAddrs(&peerstore.AddrInfo{
		ID:    host.ID(),
		Addrs: host.Addrs(),
	})
	fmt.Println("libp2p node address:", addrs[0])

	// Connect to peer if needed
	if dest != "" {
		addr, err := multiaddr.NewMultiaddr(dest)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("addr: %+v\n", addr)
		peer, err := peerstore.AddrInfoFromP2pAddr(addr)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("peer: %+v\n", peer)
		ctxWithTimeout, _ := context.WithTimeout(ctx, time.Second*5)
		if err := host.Connect(ctxWithTimeout, *peer); err != nil {
			fmt.Print(err)
			return
		}
	}

	go func() {
		tick := time.Tick(time.Second * 30)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick:
				if refresh {
					reqCtx, cancel := context.WithCancel(ctx)
					var evtCh <-chan *routing.QueryEvent
					subCtx, evtCh := routing.RegisterForQueryEvents(reqCtx)

					_, err := dht.GetClosestPeers(subCtx, string(host.ID()))
					if err != nil {
						cancel()
						fmt.Println(err)
						continue
					}

					go func() {
						for e := range evtCh {
							if e.Type == routing.PeerResponse {
								peers := e.Responses
								for _, peer := range peers {
									dht.Host().Connect(ctx, *peer)
									// todo original negotiation
									s, e := dht.RoutingTable().TryAddPeer(peer.ID, false, true)
									fmt.Printf("TryAddPeer: %+v => %t, %+v\n", peer.ID, s, e)
								}
							}
						}
					}()
					fmt.Printf("RoutingTable: %+v\n", dht.RoutingTable().ListPeers())
					cancel()
				}
			}
		}
	}()

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}
