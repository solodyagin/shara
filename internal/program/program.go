package program

import (
	"fmt"
	"log"

	"shara/internal/database"
	"shara/internal/engine"

	"github.com/kardianos/service"
	"github.com/spf13/viper"
)

type Program struct {
	exit chan struct{} // Канал для остановки работы
	cfg  *viper.Viper
	srv  *engine.Server
}

// New создаёт новую программу
func New(cfg *viper.Viper, db database.Database) *Program {
	p := new(Program)
	p.cfg = cfg
	p.srv = engine.NewServer(cfg, db)
	return p
}

// Start вызывается при запуске службы
func (p *Program) Start(s service.Service) error {
	p.exit = make(chan struct{})
	// Основная работа программы
	go func() {
		addr := fmt.Sprintf("%s:%d", p.cfg.GetString("server.host"), p.cfg.GetInt("server.port"))
		p.srv.Run(addr)
		log.Printf("Server is running at %s\n", addr)
		<-p.exit
		_ = p.srv.Stop()
	}()
	return nil
}

// Stop вызывается при остановке службы
func (p *Program) Stop(s service.Service) error {
	close(p.exit)
	return nil
}
