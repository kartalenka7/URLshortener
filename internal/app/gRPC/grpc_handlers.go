package grpc

import (
	context "context"

	"example.com/shortener/internal/app/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GrpcHandlers поддерживает все необходимые методы gRPC сервера
type GrpcHandlers struct {
	UnimplementedHandlersServer

	service *service.Service
}

// NewGrpcHandlers - конструктор
func NewGrpcHandlers(service *service.Service) *GrpcHandlers {
	return &GrpcHandlers{
		service: service,
	}
}

// GetUserFromContext возвращает значение токена пользователя из контекста
func GetUserFromContext(ctx context.Context) string {
	var user string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get("User")
		if len(values) > 0 {
			user = values[0]
		}
	}
	return user
}

// ShortenURL возвращает сокращенный токен
func (g *GrpcHandlers) ShortenURL(ctx context.Context, in *ShortenURLRequest) (
	*ShortenURLResponse, error) {
	var response ShortenURLResponse

	token, err := g.service.AddLink(ctx, "", in.LongURL, GetUserFromContext(ctx))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error in adding link to storage")
	}
	response.Token = token
	return &response, nil
}

// DeleteURLs принимает строку с токенами и запускает горутину на удаление записей
func (g *GrpcHandlers) DeleteURLs(ctx context.Context, in *DeleteURLsRequest) (
	*DeleteURLsResponse, error) {
	var response DeleteURLsResponse

	go g.service.AddDeletedTokens(in.Token, GetUserFromContext(ctx))

	return &response, nil
}

// GetFullURL возвращает исходный URL
func (g *GrpcHandlers) GetFullURL(ctx context.Context, in *GetFullURLRequest) (
	*GetFullURLResponse, error) {
	var response GetFullURLResponse

	lToken := g.service.GetLongToken(in.Token)
	longURL, err := g.service.GetLongURL(ctx, lToken)
	if err != nil {
		return nil, err
	}
	response.LongURL = longURL
	return &response, nil
}

// getUserURLs возвращает все URL, сокращенным пользвателем
func (g *GrpcHandlers) GetUserURLs(ctx context.Context, in *GetUserURLsRequest) (
	*GetUserURLsResponse, error) {
	var response GetUserURLsResponse

	_, err := g.service.GetAllURLS(ctx, GetUserFromContext(ctx))
	if err != nil {
		return nil, err
	}
	return &response, nil
}
