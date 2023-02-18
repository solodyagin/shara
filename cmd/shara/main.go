package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"shara/internal/program"
	"shara/internal/utils"
	"shara/migrations"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/kardianos/service"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

func main() {
	// Парсируем командную строку
	svcFlag := flag.String("service", "", "Control the system service.")
	flag.Parse()

	// Определяем директории
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	execDir, _ := filepath.Split(execPath)
	execDir = filepath.Clean(execDir)

	// Читаем конфигурационный файл
	cfg := viper.New()
	cfg.SetDefault("service.name", "shara")                                      // Имя службы
	cfg.SetDefault("service.display_name", "Shara Service")                      // Отображаемое имя службы
	cfg.SetDefault("service.description", "Shara Service")                       // Описание службы
	cfg.SetDefault("server.host", "localhost")                                   // Хост сервера
	cfg.SetDefault("server.port", 8032)                                          // Порт сервера
	cfg.SetDefault("minio.endpoint", "myhost.company.lan:9000")                  //
	cfg.SetDefault("minio.access_key", "MwIBRCEEcfS7dOKZ")                       //
	cfg.SetDefault("minio.secret_key", "oxxQ2HUY1XpOY8SgEqiJR3FG7ZpFWGEL")       //
	cfg.SetDefault("minio.bucket_name", "shara")                                 //
	cfg.SetDefault("minio.location", "us-east-1")                                //
	cfg.SetDefault("minio.use_ssl", false)                                       // HTTPS
	cfg.SetDefault("max_upload_size", 104857600)                                 // 100 МБ
	cfg.SetDefault("pathes.temp_dir", os.TempDir())                              // Путь до временной директории
	cfg.SetDefault("pathes.database", filepath.Join(execDir, "database.sqlite")) // Путь до базы данных SQLite

	cfg.SetConfigName("shara")
	cfg.SetConfigType("yaml")

	switch runtime.GOOS {
	case "linux":
		cfg.AddConfigPath("/etc/shara")
		cfg.AddConfigPath("$HOME/.config/shara")
	case "windows":
		cfg.AddConfigPath(filepath.Join(os.Getenv("PROGRAMDATA"), "Shara"))
	}
	cfg.AddConfigPath(filepath.Join(execDir, "configs"))

	if err := cfg.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	// Открываем БД
	db, err := sql.Open("sqlite3", cfg.GetString("pathes.database"))
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	}
	defer db.Close()

	// Выполняем миграцию БД
	ds, err := iofs.New(migrations.FS, "sqlite")
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	}
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	}
	m, err := migrate.NewWithInstance("iofs", ds, "sqlite3", driver)
	if err != nil {
		log.Fatalf("Error: %s\n", err)
	}
	m.Up()

	// Создаём программу
	p := program.New(cfg, db)

	// Задаём настройки для службы
	options := make(service.KeyValue)
	options["Restart"] = "on-success"
	options["SuccessExitStatus"] = "1 2 8 SIGKILL"

	svcConfig := &service.Config{
		Name:        cfg.GetString("service.name"),
		DisplayName: cfg.GetString("service.displayname"),
		Description: cfg.GetString("service.description"),
		Option:      options,
	}
	if runtime.GOOS == "linux" {
		svcConfig.Dependencies = []string{
			"Requires=network.target",
			"After=network-online.target syslog.target",
		}
	}

	// Создаём службу
	svc, err := service.New(p, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	errs := make(chan error, 5)

	// Открываем системный логгер
	logger, err := svc.Logger(errs)
	if err != nil {
		log.Fatal(err)
	}

	// Вывод ошибок
	go func() {
		for {
			if err := <-errs; err != nil {
				log.Println(err)
			}
		}
	}()

	// Управление службой
	if len(*svcFlag) != 0 {
		if !utils.Contains(service.ControlAction[:], *svcFlag, true) {
			fmt.Fprintf(os.Stdout, "Valid actions: %q\n", service.ControlAction)
		} else if err := service.Control(svc, *svcFlag); err != nil {
			fmt.Fprintln(os.Stdout, err)
		}
		return
	}

	log.Printf("Used config file \"%s\"\n", cfg.ConfigFileUsed())

	// Запускаем службу
	if err := svc.Run(); err != nil {
		logger.Error(err)
	}
}
