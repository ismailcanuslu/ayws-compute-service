package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	dockerclient "github.com/docker/docker/client"
	"github.com/ismailcanuslu/ayws-compute-service/config"
	"github.com/ismailcanuslu/ayws-compute-service/internal/container"
	"github.com/ismailcanuslu/ayws-compute-service/internal/router"
	"github.com/ismailcanuslu/ayws-compute-service/internal/serverless"
	"github.com/ismailcanuslu/ayws-compute-service/internal/vm"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// ── Logger ───────────────────────────────────────────────────────────────
	zerolog.TimeFieldFormat = time.RFC3339
	if os.Getenv("ENV") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}

	// ── Config ───────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config yüklenemedi")
	}

	log.Info().
		Int("port", cfg.Server.Port).
		Msg("ayws-compute-service başlatılıyor")

	// ── VM (Proxmox) ──────────────────────────────────────────────────────────
	var proxmoxClient *vm.ProxmoxClient
	if cfg.Proxmox.TokenSecret != "" {
		px, err := vm.NewProxmoxClient(&cfg.Proxmox)
		if err != nil {
			log.Warn().Err(err).Msg("Proxmox bağlantısı kurulamadı — VM modülü devre dışı")
		} else {
			proxmoxClient = px
			log.Info().Str("host", cfg.Proxmox.Host).Msg("Proxmox bağlantısı kuruldu")
		}
	} else {
		log.Warn().Msg("Proxmox yapılandırılmamış — VM modülü devre dışı")
	}
	vmSvc := vm.NewService(proxmoxClient)
	vmHandler := vm.NewHandler(vmSvc)

	// ── Docker ────────────────────────────────────────────────────────────────
	// Mac Docker Desktop soket yolu otomatik keşfedilir.
	var dockerCli *dockerclient.Client
	dockerHosts := []string{
		cfg.Docker.Host,
		"unix:///var/run/docker.sock",
		"unix://" + os.Getenv("HOME") + "/.docker/run/docker.sock",
	}

	for _, host := range dockerHosts {
		if host == "" {
			continue
		}
		cli, err := dockerclient.NewClientWithOpts(
			dockerclient.WithHost(host),
			dockerclient.WithAPIVersionNegotiation(),
		)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, pingErr := cli.Ping(ctx)
		cancel()
		if pingErr == nil {
			dockerCli = cli
			log.Info().Str("host", host).Msg("Docker bağlantısı kuruldu")
			break
		}
		cli.Close()
	}
	if dockerCli == nil {
		log.Warn().Msg("Docker socket bulunamadı — Container/Serverless modülleri devre dışı")
	}

	// ── Serverless ────────────────────────────────────────────────────────────
	var slRunner *serverless.Runner
	if dockerCli != nil {
		slRunner = serverless.NewRunner(dockerCli, cfg.Serverless.MemoryLimitMB, cfg.Serverless.TimeoutSeconds)
	}
	slSvc := serverless.NewService(slRunner)
	slHandler := serverless.NewHandler(slSvc)

	// ── Container & K8s ───────────────────────────────────────────────────────
	var dockerSvc *container.DockerService
	if dockerCli != nil {
		dockerSvc = container.NewDockerService(dockerCli)
	}

	var k8sSvc *container.K8sService
	k8sSvc, err = container.NewK8sService(cfg.Kubernetes.Kubeconfig, cfg.Kubernetes.Namespace)
	if err != nil {
		log.Warn().Err(err).Msg("Kubernetes bağlantısı kurulamadı — K8s modülü devre dışı")
		k8sSvc = nil
	} else {
		log.Info().Msg("Kubernetes bağlantısı kuruldu")
	}

	ctHandler := container.NewHandler(dockerSvc, k8sSvc)

	// ── Fiber app ─────────────────────────────────────────────────────────────
	app := router.Setup(vmHandler, slHandler, ctHandler)

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		log.Info().Str("addr", addr).Msg("dinleniyor")
		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("sunucu başlatılamadı")
		}
	}()

	<-quit
	log.Info().Msg("kapatma sinyali alındı...")

	if err := app.Shutdown(); err != nil {
		log.Error().Err(err).Msg("graceful shutdown başarısız")
	}
	log.Info().Msg("ayws-compute-service kapatıldı")
}
