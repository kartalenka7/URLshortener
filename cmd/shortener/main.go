package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "example.com/shortener/internal/app/gRPC"
	"example.com/shortener/internal/app/handlers"
	"example.com/shortener/internal/app/service"
	"example.com/shortener/internal/app/storage"
	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
	"example.com/shortener/internal/logger"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

var (
	localAddr    = "localhost:8080"
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	var storer service.Storer
	var err error

	showBuildData()

	log := logger.InitLog()

	// получаем структуру с конфигурацией приложения
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}
	log.WithFields(logrus.Fields{"cfg": cfg}).Debug("Конфигурация приложения")

	// создаем объект хранилища
	storer = storage.New(cfg, log)
	service := service.New(cfg, storer, log)

	// канал для перенаправления прерываний
	// поскольку нужно отловить всего одно прерывание,
	// ёмкости 1 для канала будет достаточно
	sigint := make(chan os.Signal, 1)
	// регистрируем перенаправление прерываний
	signal.Notify(sigint, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	router := handlers.NewRouter(service, log)

	srv := http.Server{
		Addr:    cfg.Server,
		Handler: router}

	go func() {
		log.WithFields(logrus.Fields{"server": cfg.Server})
		if !cfg.HTTPS {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %v\n", err)
			}
		} else {
			// включение HTTPS
			err = utils.GenerateCertTSL(log)
			if err == nil {
				if err := srv.ListenAndServeTLS(`cert.pem`, `key.pem`); err != nil && err != http.ErrServerClosed {
					log.Fatalf("listen: %v\n", err)
				}
			}
		}
	}()

	// Поддержка gRPC
	listen, err := net.Listen("tcp", ":9090")

	// создаем gRPC сервер
	creds := insecure.NewCredentials()
	server := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(auth.UnaryServerInterceptor(pb.AuthInterceptor)),
	)
	// рефлексия
	reflection.Register(server)
	// регистрируем сервис

	pb.RegisterHandlersServer(server, pb.NewGrpcHandlers(service))
	log.Info("gRPC server started")

	go func() {
		if err := server.Serve(listen); err != nil {
			log.Fatalf("listen: %v\n", err)
		}
	}()

	// читаем из канала прерываний
	// поскольку нужно прочитать только одно прерывание,
	// можно обойтись без цикла
	sig := <-sigint
	log.Printf("Received signal: %v\n", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		// ошибки закрытия Listener
		log.Printf("HTTP server Shutdown: %v", err)
	}

	//завершения процедуры graceful shutdown
	log.Println("Server shutdown gracefully")
	// закрываем ресурсы перед выходом
	err = service.Close()
	if err != nil {
		log.Fatal(err)
	}
}

// showBuildData выводит версию, время и последний коммит текущей сборки
func showBuildData() {
	if buildVersion == "" {
		buildVersion = "N/A"
	}
	if buildDate == "" {
		buildDate = "N/A"
	}
	if buildCommit == "" {
		buildCommit = "N/A"
	}
	fmt.Printf("Build version:%s\nBuild date:%s\nBuild commit:%s\n", buildVersion, buildDate, buildCommit)

}
