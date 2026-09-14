package client

import (
	authv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/auth/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthClient struct {
	Client authv1.AuthServiceClient
	Conn   *grpc.ClientConn
}

func NewAuthClient(
	address string,
) (*AuthClient, error) {

	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)

	if err != nil {
		return nil, err
	}

	client :=
		authv1.NewAuthServiceClient(conn)

	return &AuthClient{
		Client: client,
		Conn:   conn,
	}, nil
}
