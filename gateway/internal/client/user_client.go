package client

import (
	userv1 "github.com/chandangowdacbkrewops/krewops-backend/gen/go/user/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type UserClient struct {
	Client userv1.UserServiceClient
	Conn   *grpc.ClientConn
}

func NewUserClient(
	address string,
) (*UserClient, error) {

	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)

	if err != nil {
		return nil, err
	}

	return &UserClient{
		Client: userv1.NewUserServiceClient(conn),
		Conn:   conn,
	}, nil
}
