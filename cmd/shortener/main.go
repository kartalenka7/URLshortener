package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/shortener/internal/app/handlers"
	"example.com/shortener/internal/app/service"
	"example.com/shortener/internal/app/storage"
	"example.com/shortener/internal/config"
	"example.com/shortener/internal/config/utils"
	"example.com/shortener/internal/logger"
	"github.com/sirupsen/logrus"
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
	router := handlers.NewRouter(service, log)

	srv := http.Server{
		Addr:    cfg.Server,
		Handler: router}

	// через этот канал сообщим основному потоку, что соединения закрыты
	idleConnsClosed := make(chan struct{})
	// канал для перенаправления прерываний
	// поскольку нужно отловить всего одно прерывание,
	// ёмкости 1 для канала будет достаточно
	sigint := make(chan os.Signal, 1)
	// регистрируем перенаправление прерываний
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, time.Minute*1)
	defer cancel()

	// запускаем горутину обработки пойманных прерываний
	go func() {
		// читаем из канала прерываний
		// поскольку нужно прочитать только одно прерывание,
		// можно обойтись без цикла
		select {
		case <-sigint:
			fmt.Println("Syscall error")
		case <-ctx.Done():
			fmt.Println("Context exeeded")
		}
		if err := srv.Shutdown(ctx); err != nil {
			// ошибки закрытия Listener
			log.Printf("HTTP server Shutdown: %v", err)
		}
		// сообщаем основному потоку,
		// что все сетевые соединения обработаны и закрыты
		close(idleConnsClosed)
	}()

	log.WithFields(logrus.Fields{"server": cfg.Server})
	if cfg.HTTPS == "" {
		//log.Fatal(http.ListenAndServe(cfg.Server, router))
		log.Fatal(srv.ListenAndServe())
	} else {
		// включение HTTPS
		err = utils.GenerateCertTSL(log)
		if err == nil {
			//log.Fatal(http.ListenAndServeTLS(cfg.Server, `cert.pem`, `key.pem`, router))
			log.Fatal(srv.ListenAndServeTLS(`cert.pm`, `key.pm`))
		}
	}

	// ждём завершения процедуры graceful shutdown
	<-idleConnsClosed
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
