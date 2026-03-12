package envoygatewaysample

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"

	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"

	clusterService "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	listenerService "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeService "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"

	types "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	resource "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	server "github.com/envoyproxy/go-control-plane/pkg/server/v3"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

var snapshotCache cache.SnapshotCache

type ClusterUpdate struct {
	Port uint32 `json:"port"`
}

func main() {

	ctx := context.Background()

	snapshotCache = cache.NewSnapshotCache(false, cache.IDHash{}, nil)

	srv := server.NewServer(ctx, snapshotCache, nil)

	grpcServer := grpc.NewServer()

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, srv)
	clusterService.RegisterClusterDiscoveryServiceServer(grpcServer, srv)
	listenerService.RegisterListenerDiscoveryServiceServer(grpcServer, srv)
	routeService.RegisterRouteDiscoveryServiceServer(grpcServer, srv)

	lis, err := net.Listen("tcp", ":18000")
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Println("xDS server running on :18000")
		grpcServer.Serve(lis)
	}()

	updateConfig(9090)

	http.HandleFunc("/cluster", updateCluster)

	log.Println("REST API running on :9000")

	http.ListenAndServe(":9000", nil)
}

func updateCluster(w http.ResponseWriter, r *http.Request) {

	var req ClusterUpdate

	json.NewDecoder(r.Body).Decode(&req)

	updateConfig(req.Port)

	w.Write([]byte("cluster updated"))
}

func updateConfig(port uint32) {

	clusterConfig := &cluster.Cluster{
		Name:           "backend_service",
		ConnectTimeout: durationpb.New(5_000_000_000),
		ClusterDiscoveryType: &cluster.Cluster_Type{
			Type: cluster.Cluster_LOGICAL_DNS,
		},
		LoadAssignment: &endpoint.ClusterLoadAssignment{
			ClusterName: "backend_service",
			Endpoints: []*endpoint.LocalityLbEndpoints{
				{
					LbEndpoints: []*endpoint.LbEndpoint{
						{
							HostIdentifier: &endpoint.LbEndpoint_Endpoint{
								Endpoint: &endpoint.Endpoint{
									Address: &core.Address{
										Address: &core.Address_SocketAddress{
											SocketAddress: &core.SocketAddress{
												Address: "127.0.0.1",
												PortSpecifier: &core.SocketAddress_PortValue{
													PortValue: port,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	routeConfig := &route.RouteConfiguration{
		Name: "local_route",
		VirtualHosts: []*route.VirtualHost{
			{
				Name:    "service",
				Domains: []string{"*"},
				Routes: []*route.Route{
					{
						Match: &route.RouteMatch{
							PathSpecifier: &route.RouteMatch_Prefix{Prefix: "/"},
						},
						Action: &route.Route_Route{
							Route: &route.RouteAction{
								ClusterSpecifier: &route.RouteAction_Cluster{
									Cluster: "backend_service",
								},
							},
						},
					},
				},
			},
		},
	}

	hcmConfig := &hcm.HttpConnectionManager{
		StatPrefix: "ingress_http",
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: routeConfig,
		},
		HttpFilters: []*hcm.HttpFilter{
			{
				Name: "envoy.filters.http.router",
			},
		},
	}

	pbst, _ := anypb.New(hcmConfig)

	listenerConfig := &listener.Listener{
		Name: "listener_http",
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: 8080,
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{
			{
				Filters: []*listener.Filter{
					{
						Name: "envoy.filters.network.http_connection_manager",
						ConfigType: &listener.Filter_TypedConfig{
							TypedConfig: pbst,
						},
					},
				},
			},
		},
	}

	snap, _ := cache.NewSnapshot("1",
		map[resource.Type][]types.Resource{
			resource.ClusterType:  {clusterConfig},
			resource.ListenerType: {listenerConfig},
			resource.RouteType:    {routeConfig},
		},
	)

	snapshotCache.SetSnapshot(context.Background(), "test-node", snap)

	log.Println("cluster updated to port", port)
}
