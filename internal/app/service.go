package app

import (
	"context"
	"database/sql"

	"fmt"

		"log"

		"share_word/internal/db"

		"sync"

		"time"

	"github.com/nats-io/nats-server/v2/server"

	"github.com/nats-io/nats.go"
)

type Service struct {
	Queries *db.Queries

	db *sql.DB

	SkipCooldown bool

	NatsServer *server.Server

	NC *nats.Conn

	StartTime int64

	// SessionToken:ClientID -> ClueID
	EditingClues sync.Map

	// SessionToken:ClientID -> X,Y
	FocusedCells sync.Map

	// SessionToken:ClientID -> Direction
	CurrentDirections sync.Map
}

func NewService(queries *db.Queries, dbConn *sql.DB) *Service {

	s := &Service{

		Queries: queries,

		db: dbConn,

		StartTime: time.Now().UnixMilli(),
	}

	s.startNats()

	return s

}

func (s *Service) startNats() {

	opts := &server.Options{

		Port: -1,

		NoLog: true,
	}

	ns, err := server.NewServer(opts)

	if err != nil {

		log.Printf("Failed to create NATS server: %v", err)

		return

	}

	go ns.Start()

	if ns.ReadyForConnections(2 * time.Second) {

		log.Printf("NATS server ready at %s", ns.ClientURL())

		s.NatsServer = ns

		nc, err := nats.Connect(ns.ClientURL())

		if err == nil {

			log.Printf("NATS client connected")

			s.NC = nc

		} else {

			log.Printf("NATS client failed to connect: %v", err)

		}

	} else {

		log.Printf("NATS server failed to become ready")

	}

}

func (s *Service) Shutdown() {

	if s.NC != nil {

		s.NC.Close()

	}

	if s.NatsServer != nil {

		s.NatsServer.Shutdown()

		s.NatsServer.WaitForShutdown()

	}

}

func (s *Service) BroadcastUpdate(puzzleID string, structural bool) {

	if s.NC == nil {

		log.Printf("Broadcast skipped: NATS connection is nil")

		return

	}

	subject := fmt.Sprintf("puzzles.%s", puzzleID)

	msg := "signal"

	if structural {

		msg = "structural"

	}

	log.Printf("Publishing to NATS: %s -> %s", subject, msg)

	_ = s.Queries.UpdatePuzzleUpdatedAt(context.Background(), puzzleID)

	_ = s.NC.Publish(subject, []byte(msg))
}
