package engine

import (
	"context"
	"log"
	"net/http"
	"time"

	"shara/internal/database"
	"shara/web"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"
)

type Engine struct {
	config *viper.Viper
	db     database.Database
	client *minio.Client
	server *http.Server
}

func New(cfg *viper.Viper, db database.Database) *Engine {
	e := new(Engine)
	e.config = cfg
	e.db = db

	// Инициализируем Minio клиент
	client, err := minio.New(e.config.GetString("minio.endpoint"), &minio.Options{
		Creds: credentials.NewStaticV4(
			e.config.GetString("minio.access_key"),
			e.config.GetString("minio.secret_key"),
			"",
		),
		Secure: e.config.GetBool("minio.use_ssl"),
	})
	if err != nil {
		log.Fatalln(err)
	}
	e.client = client

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Файлы интерфейса
	// router.Use(static.Serve("/", static.LocalFile(path.Join("web", "public"), true)))
	router.Use(static.Serve("/", EmbedFolder(web.FS, "public")))

	// router.NoRoute(func(c *gin.Context) {
	// 	log.Printf("%s doesn't exists, redirect on /\n", c.Request.URL.Path)
	// 	c.Redirect(http.StatusMovedPermanently, "/")
	// })

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

	// Обработчики
	router.POST("/upload", e.HandleUpload())
	router.GET("/download/:fileId", e.HandleDownload())

	e.server = &http.Server{
		Handler:      router,
		WriteTimeout: 5 * time.Minute, // Таймаут ответа от сервера
	}

	return e
}

// Run
func (e *Engine) Run(addr string) {
	e.server.Addr = addr

	go func() {
		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error: %s\n", err)
		}
	}()
}

// Stop
func (e *Engine) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return e.server.Shutdown(ctx)
}
