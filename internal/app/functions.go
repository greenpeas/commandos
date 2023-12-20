package app

import (
	"commandos/internal/commands"
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"net"
	"time"

	commands_v1 "gitlab.kvant.online/seal/grpc-contracts/pkg/commands/v1"

	"google.golang.org/grpc"
)

func connectPg(databaseUrl string, attempt int8, ctx context.Context) (*pgxpool.Pool, error) {

	dbpool, err := pgxpool.New(ctx, databaseUrl)

	if err != nil {
		log.Println(err)
		time.Sleep(10 * time.Second)
		attempt--
		if attempt == 0 {
			return dbpool, err
		}
		return connectPg(databaseUrl, attempt, ctx)
	}

	return dbpool, err
}

func (r *App) Run() {
	log.Println("App start")
	defer log.Println("App shutdown")

	s := grpc.NewServer()
	srv := commands.NewGRPCServer(r.repo, r.logger)

	commands_v1.RegisterCommandsServiceServer(s, srv)

	var lConfig net.ListenConfig
	listen, err := lConfig.Listen(r.ctx, r.cfg.Grpc.Network, r.cfg.Grpc.Address)

	if err != nil {
		log.Fatal(err)
	}

	defer listen.Close()

	go func() {
		<-r.ctx.Done()
		log.Println("Listener service down...")
		listen.Close()
	}()

	log.Printf(`Try serve on %s %s`, r.cfg.Grpc.Network, r.cfg.Grpc.Address)

	if err := s.Serve(listen); err != nil {
		log.Fatal(err)
	}

}
