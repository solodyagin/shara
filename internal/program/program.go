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
	exit   chan struct{} // Канал для остановки работы
	config *viper.Viper
	engine *engine.Engine
}

// New создаёт новую программу
func New(cfg *viper.Viper, db database.Database) *Program {
	p := new(Program)
	p.config = cfg
	p.engine = engine.New(cfg, db)
	return p
}

// Start вызывается при запуске службы
func (p *Program) Start(s service.Service) error {
	p.exit = make(chan struct{})
	go p.run()
	return nil
}

// Stop вызывается при остановке службы
func (p *Program) Stop(s service.Service) error {
	close(p.exit)
	return nil
}

// run основная работа программы
func (p *Program) run() {
	addr := fmt.Sprintf("%s:%d", p.config.GetString("server.host"), p.config.GetInt("server.port"))
	p.engine.Run(addr)
	log.Printf("Server is running at %s\n", addr)

	<-p.exit

	_ = p.engine.Stop()
}
