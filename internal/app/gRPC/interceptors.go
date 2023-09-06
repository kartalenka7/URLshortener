package grpc

import (
	"context"

	"github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// jwtToken структура для парсинга jwt токенов
type jwtToken struct {
	Login string
	jwt.StandardClaims
}

// AuthInterceptor функция для интерсептора, которая проверяет jwt токены
func AuthInterceptor(ctx context.Context) (context.Context, error) {
	var user string
	var tk jwtToken

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get("User")
		if len(values) > 0 {
			user = values[0]
		} else {
			token := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), tk)
			tokenString, err := token.SignedString([]byte("secret"))
			if err != nil {
				return ctx, status.Errorf(codes.Unauthenticated, "error while creating token")
			}
			md := metadata.New(map[string]string{"User": tokenString})
			ctx = metadata.NewOutgoingContext(context.Background(), md)
			return ctx, nil
		}

		token, err := jwt.ParseWithClaims(user, tk, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		})
		if err != nil {
			return ctx, status.Errorf(codes.Unauthenticated, "not valid token")
		}

		if !token.Valid {
			return ctx, status.Errorf(codes.Unauthenticated, "not valid token")
		}

	} else {
		return ctx, status.Errorf(codes.Unauthenticated, "no metadata")
	}
	return ctx, nil
}
