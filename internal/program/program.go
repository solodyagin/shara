package program

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"shara/internal/handlers"
	"time"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/gin"

	"github.com/kardianos/service"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/spf13/viper"
)

type Program struct {
	exit   chan struct{} // Канал для остановки работы
	config *viper.Viper  // Конфиг
	db     *sql.DB
	client *minio.Client // Клиент Minio
	server *http.Server  // Вебсервер
}

// New создаёт новую программу
func New(cfg *viper.Viper, db *sql.DB, workDir string) *Program {
	p := new(Program)
	p.config = cfg
	p.db = db

	// Инициализируем Minio клиент
	client, err := minio.New(p.config.GetString("minio.endpoint"), &minio.Options{
		Creds: credentials.NewStaticV4(
			p.config.GetString("minio.access_key"),
			p.config.GetString("minio.secret_key"),
			"",
		),
		Secure: p.config.GetBool("minio.use_ssl"),
	})
	if err != nil {
		log.Fatalln(err)
	}
	p.client = client

	// Инициализируем вебсервер
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Путь до статики
	router.Use(static.Serve("/", static.LocalFile(filepath.Join(workDir, "web"), true)))

	// Некоторые настройки, связанные с безопасностью
	router.Use(secure.Secure(secure.Options{
		// AllowedHosts:          []string{"example.com", "ssl.example.com"},
		// SSLRedirect:           true,
		// SSLHost:               "ssl.example.com",
		// SSLProxyHeaders:       map[string]string{"X-Forwarded-Proto": "https"},
		// STSSeconds:            315360000,
		// STSIncludeSubdomains:  true,
		FrameDeny:             true, // Запрещает показывать сайт во фрейме
		ContentTypeNosniff:    true,
		BrowserXssFilter:      true,
		ContentSecurityPolicy: "default-src 'self'",
	}))

	// API
	v1 := router.Group("/api/v1")
	v1.POST("/upload", handlers.HandleUpload(p))
	v1.GET("/download/:fileId", handlers.HandleDownload(p))

	addr := fmt.Sprintf("%s:%d", p.config.GetString("server.host"), p.config.GetInt("server.port"))

	p.server = &http.Server{
		Addr:         addr,
		Handler:      router,
		WriteTimeout: 5 * time.Minute, // Таймаут ответа от сервера
	}

	return p
}

// Start вызывается при запуске службы
func (p *Program) Start(s service.Service) error {
	p.exit = make(chan struct{})
	go func() {
		p.listen() // Запускаем вебсервер
		<-p.exit
		p.shutdown() // Останавливаем вебсервер
	}()
	return nil
}

// Stop вызывается при остановке службы
func (p *Program) Stop(s service.Service) error {
	close(p.exit)
	return nil
}

// listen запускает вебсервер
func (p *Program) listen() {
	go func() {
		if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server failed to start due to err: %v\n", err)
			panic(err)
		}
	}()
	log.Printf("Server is running at %s\n", p.server.Addr)
}

// shutdown останавливает вебсервер
func (p *Program) shutdown() error {
	// Попытка корректного завершения работы
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return p.server.Shutdown(ctx)
}

// GetConfig возвращает указатель на config
func (p *Program) GetConfig() *viper.Viper {
	return p.config
}

// GetDB
func (p *Program) GetDB() *sql.DB {
	return p.db
}

// GetClient
func (p *Program) GetClient() *minio.Client {
	return p.client
}
