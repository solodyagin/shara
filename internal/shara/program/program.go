package program

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/kardianos/service"
	"github.com/spf13/viper"
	"gorm.io/gorm"

	"github.com/gin-contrib/secure"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"

	"github.com/tenrok/filestore"
	"github.com/tenrok/filestore/remote"
	"github.com/tenrok/filestore/remote/miniostorage"

	"application/internal/shara/mode"
	"application/web"
)

type Program struct {
	exit    chan struct{} // Канал для остановки работы
	cfg     *viper.Viper
	db      *gorm.DB
	httpFS  *filestore.HttpFS
	httpSrv *http.Server
}

// NewProgram создаёт новую программу
func NewProgram(cfg *viper.Viper) (*Program, error) {
	db, err := gorm.Open(sqlite.Open(cfg.GetString("database")), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Migrate the schema
	db.AutoMigrate(&Record{})

	p := &Program{}
	p.cfg = cfg
	p.db = db

	opts := []filestore.HttpFSOption{}

	if p.cfg.GetString("storage") == "minio" {
		connString := miniostorage.ConnString(miniostorage.Config{
			Endpoint:    p.cfg.GetString("storages.minio.endpoint"),
			AccessKeyID: p.cfg.GetString("storages.minio.accessKey"),
			SecretKey:   p.cfg.GetString("storages.minio.secretKey"),
			Token:       "",
			BucketName:  p.cfg.GetString("storages.minio.bucketMame"),
			Prefix:      "",
			Region:      p.cfg.GetString("storages.minio.region"),
			Secure:      p.cfg.GetBool("storages.minio.secure"),
		})
		remoteStorage, err := remote.NewStorage(context.Background(), connString)
		if err != nil {
			return nil, err
		}
		opts = append(opts, filestore.WithRemoteStorage(remoteStorage))
	}

	httpFS, err := filestore.NewHttpFS(p.cfg.GetString("storages.local.endpoint"), opts...)
	if err != nil {
		return nil, err
	}
	p.httpFS = httpFS

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Файлы интерфейса
	if mode.Debug {
		router.Use(static.Serve("/", static.LocalFile(path.Join("web", "public"), true)))
	} else {
		webFS, err := static.EmbedFolder(web.FS, "public")
		if err != nil {
			return nil, err
		}
		router.Use(static.Serve("/", webFS))
	}

	// Некоторые настройки, связанные с безопасностью
	router.Use(secure.New(secure.Config{
		// AllowedHosts:          []string{"example.com", "ssl.example.com"},
		// SSLRedirect:           true,
		// SSLHost:               "ssl.example.com",
		// STSSeconds:            315360000,
		// STSIncludeSubdomains:  true,
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		BrowserXssFilter:      true,
		ContentSecurityPolicy: "default-src 'self'",
		// IENoOpen:              true,
		// ReferrerPolicy:        "strict-origin-when-cross-origin",
		// SSLProxyHeaders:       map[string]string{"X-Forwarded-Proto": "https"},
	}))

	// Обработчики
	router.POST("/upload", p.HandleUpload())
	router.GET("/download/:link", p.HandleDownload())

	p.httpSrv = &http.Server{
		Handler:      router,
		WriteTimeout: 5 * time.Minute, // Таймаут ответа от сервера
	}

	return p, nil
}

// Start вызывается при запуске службы
func (p *Program) Start(_ service.Service) error {
	p.exit = make(chan struct{})

	// Основная работа программы
	go func() {
		p.httpSrv.Addr = fmt.Sprintf("%s:%d", p.cfg.GetString("server.host"), p.cfg.GetInt("server.port"))

		go func() {
			if err := p.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalln(err)
			}
		}()

		log.Printf("Server is running at http://%s\n", p.httpSrv.Addr)

		<-p.exit

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		p.httpSrv.Shutdown(ctx)
	}()

	return nil
}

// Stop вызывается при остановке службы
func (p *Program) Stop(_ service.Service) error {
	close(p.exit)
	return nil
}
