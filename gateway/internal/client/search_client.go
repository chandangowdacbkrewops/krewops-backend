package client

import (
	searchv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/search/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type SearchClient struct {
	Client searchv1.SearchServiceClient
	Conn   *grpc.ClientConn
}

func NewSearchClient(
	address string,
) (*SearchClient, error) {

	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)

	if err != nil {
		return nil, err
	}

	return &SearchClient{
		Client: searchv1.NewSearchServiceClient(conn),
		Conn:   conn,
	}, nil
}
