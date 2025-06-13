package client

import (
	"context"
	pb "github.com/meetohin/web-chat/auth-service/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthClient struct {
	client pb.AuthServiceClient
	conn   *grpc.ClientConn
}

func NewAuthClient(address string) (*AuthClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewAuthServiceClient(conn)

	return &AuthClient{
		client: client,
		conn:   conn,
	}, nil
}

func (ac *AuthClient) Close() {
	if ac.conn != nil {
		ac.conn.Close()
	}
}

func (ac *AuthClient) ValidateToken(ctx context.Context, token string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	resp, err := ac.client.ValidateToken(ctx, &pb.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		return "", err
	}

	if !resp.Valid {
		return "", err
	}

	return resp.Username, nil
}

func (ac *AuthClient) Register(ctx context.Context, username, password string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	_, err := ac.client.Register(ctx, &pb.RegisterRequest{
		Username: username,
		Password: password,
	})
	return err
}

func (ac *AuthClient) Login(ctx context.Context, username, password string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	resp, err := ac.client.Login(ctx, &pb.LoginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return "", err
	}

	if !resp.Success {
		return "", err
	}

	return resp.Token, nil
}
