package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/kardianos/service"
	"github.com/spf13/viper"

	"application/internal/shara/program"
)

func main() {
	// Парсируем командную строку
	svcFlag := flag.String("service", "", "Control the system service.")
	flag.Parse()

	// Определяем директории
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	// Читаем конфигурационный файл
	cfg := viper.New()
	cfg.SetDefault("service.name", "shara")                                         // Имя службы
	cfg.SetDefault("service.displayName", "Shara Service")                          // Отображаемое имя службы
	cfg.SetDefault("service.description", "Shara Service")                          // Описание службы
	cfg.SetDefault("server.host", "localhost")                                      // Хост сервера
	cfg.SetDefault("server.port", 8090)                                             // Порт сервера
	cfg.SetDefault("storage", "local")                                              // Выбор хранилища: local | minio
	cfg.SetDefault("storages.local.endpoint", filepath.Join(workingDir, "uploads")) // Путь до директории локального файлового хранилища
	cfg.SetDefault("storages.minio.endpoint", "myhost.company.local:9000")          // MinIO точка подключения
	cfg.SetDefault("storages.minio.accessKey", "MwIBRCEEcfS7dOKZ")                  // MinIO access key
	cfg.SetDefault("storages.minio.secretKey", "oxxQ2HUY1XpOY8SgEqiJR3FG7ZpFWGEL")  // MinIO secret key
	cfg.SetDefault("storages.minio.bucketName", "shara")                            // MinIO bucket name
	cfg.SetDefault("storages.minio.region", "us-east-1")                            // MinIO region
	cfg.SetDefault("storages.minio.secure", false)                                  // MinIO HTTPS
	cfg.SetDefault("maxUploadSize", 104857600)                                      // Максимальный размер файла в байтах
	cfg.SetDefault("database", filepath.Join(workingDir, "shara.sqlite"))           // Путь до базы данных SQLite

	cfg.SetConfigName("shara")
	cfg.SetConfigType("yaml")

	switch runtime.GOOS {
	case "linux":
		cfg.AddConfigPath("/etc/shara")
		cfg.AddConfigPath("$HOME/.config/shara")
	case "windows":
		cfg.AddConfigPath(filepath.Join(os.Getenv("PROGRAMDATA"), "Shara"))
	}
	cfg.AddConfigPath(filepath.Join(workingDir, "configs"))

	if err := cfg.ReadInConfig(); err != nil {
		log.Fatalln(err)
	}

	// Создаём программу
	prg, err := program.NewProgram(cfg)
	if err != nil {
		log.Fatalln(err)
	}

	// Задаём настройки для службы
	options := make(service.KeyValue)
	options["Restart"] = "on-success"
	options["SuccessExitStatus"] = "1 2 8 SIGKILL"

	svcConfig := &service.Config{
		Name:        cfg.GetString("service.name"),
		DisplayName: cfg.GetString("service.displayName"),
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
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatalln(err)
	}

	errs := make(chan error, 5)

	// Открываем системный логгер
	svcLogger, err := svc.Logger(errs)
	if err != nil {
		log.Fatalln(err)
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
		if !slices.Contains(service.ControlAction[:], *svcFlag) {
			fmt.Fprintf(os.Stdout, "Valid actions: %q\n", service.ControlAction)
		} else if err := service.Control(svc, *svcFlag); err != nil {
			fmt.Fprintln(os.Stdout, err)
		}
		return
	}

	log.Printf("Used config file %q\n", cfg.ConfigFileUsed())

	// Запускаем службу
	if err := svc.Run(); err != nil {
		svcLogger.Error(err)
	}
}
