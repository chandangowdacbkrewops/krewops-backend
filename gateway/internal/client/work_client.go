package client

import (
	workv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/work/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type WorkClient struct {
	Client workv1.WorkServiceClient
	Conn   *grpc.ClientConn
}

func NewWorkClient(
	address string,
) (*WorkClient, error) {

	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)

	if err != nil {
		return nil, err
	}

	return &WorkClient{
		Client: workv1.NewWorkServiceClient(conn),
		Conn:   conn,
	}, nil
}
