package engine

import (
	"context"
	"log"
	"net/http"
	"path"
	"time"

	"github.com/andoma-go/gin-contrib/static"
	"github.com/gin-gonic/contrib/secure"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/viper"

	"shara/internal/database"
	"shara/internal/debug"
	"shara/internal/embedfs"
	"shara/web"
)

type Server struct {
	cfg     *viper.Viper
	db      database.Database
	client  *minio.Client
	httpSrv *http.Server
}

// NewServer
func NewServer(cfg *viper.Viper, db database.Database) *Server {
	srv := new(Server)
	srv.cfg = cfg
	srv.db = db

	// Инициализируем Minio клиент
	client, err := minio.New(srv.cfg.GetString("minio.endpoint"), &minio.Options{
		Creds: credentials.NewStaticV4(
			srv.cfg.GetString("minio.access_key"),
			srv.cfg.GetString("minio.secret_key"),
			"",
		),
		Secure: srv.cfg.GetBool("minio.use_ssl"),
	})
	if err != nil {
		log.Fatalln(err)
	}
	srv.client = client

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Файлы интерфейса
	if debug.Active {
		router.Use(static.Serve("/", static.LocalFile(path.Join("web", "public"), true)))
	} else {
		router.Use(static.Serve("/", embedfs.EmbedFolder(web.FS, "public")))
	}

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
	router.POST("/upload", srv.HandleUpload())
	router.GET("/download/:name", srv.HandleDownload())

	srv.httpSrv = &http.Server{
		Handler:      router,
		WriteTimeout: 5 * time.Minute, // Таймаут ответа от сервера
	}

	return srv
}

// Run запускает HTTP-сервер
func (s *Server) Run(addr string) {
	s.httpSrv.Addr = addr
	go func() {
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalln(err)
		}
	}()
}

// Stop останавливает HTTP-сервер
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpSrv.Shutdown(ctx)
}
