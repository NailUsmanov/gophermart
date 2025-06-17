package interfaces

import "context"

type Auth interface {
	Registration(ctx context.Context, login string, password string) error
	GetUserByLogin(ctx context.Context, login string) (string, error)
	GetUserIDByLogin(ctx context.Context, login string) (int, error)
	CheckHashMatch(ctx context.Context, login, password string) error
}
