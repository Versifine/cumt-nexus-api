package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Versifine/cumt-nexus-api/internal/auth/authpassword"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authtoken"
	"github.com/Versifine/cumt-nexus-api/internal/auth/authusecase"
	"github.com/Versifine/cumt-nexus-api/internal/auth/delivery/authhttp"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentrepository"
	"github.com/Versifine/cumt-nexus-api/internal/comment/commentusecase"
	"github.com/Versifine/cumt-nexus-api/internal/comment/delivery/commenthttp"
	"github.com/Versifine/cumt-nexus-api/internal/community/communityrepository"
	"github.com/Versifine/cumt-nexus-api/internal/community/communityusecase"
	"github.com/Versifine/cumt-nexus-api/internal/community/delivery/communityhttp"
	"github.com/Versifine/cumt-nexus-api/internal/media/delivery/mediahttp"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediarepository"
	"github.com/Versifine/cumt-nexus-api/internal/media/mediausecase"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/delivery/moderationhttp"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationrepository"
	"github.com/Versifine/cumt-nexus-api/internal/moderation/moderationusecase"
	"github.com/Versifine/cumt-nexus-api/internal/notification/delivery/notificationhttp"
	"github.com/Versifine/cumt-nexus-api/internal/notification/notificationrepository"
	"github.com/Versifine/cumt-nexus-api/internal/notification/notificationusecase"
	"github.com/Versifine/cumt-nexus-api/internal/platform/config"
	"github.com/Versifine/cumt-nexus-api/internal/platform/db"
	"github.com/Versifine/cumt-nexus-api/internal/platform/httpserver"
	"github.com/Versifine/cumt-nexus-api/internal/platform/logger"
	"github.com/Versifine/cumt-nexus-api/internal/post/delivery/posthttp"
	"github.com/Versifine/cumt-nexus-api/internal/post/postrepository"
	"github.com/Versifine/cumt-nexus-api/internal/post/postusecase"
	"github.com/Versifine/cumt-nexus-api/internal/search/delivery/searchhttp"
	"github.com/Versifine/cumt-nexus-api/internal/search/searchrepository"
	"github.com/Versifine/cumt-nexus-api/internal/search/searchusecase"
	"github.com/Versifine/cumt-nexus-api/internal/storage"
	"github.com/Versifine/cumt-nexus-api/internal/user/delivery/userhttp"
	"github.com/Versifine/cumt-nexus-api/internal/user/userrepository"
	"github.com/Versifine/cumt-nexus-api/internal/user/userusecase"
	"github.com/Versifine/cumt-nexus-api/internal/vote/delivery/votehttp"
	"github.com/Versifine/cumt-nexus-api/internal/vote/voterepository"
	"github.com/Versifine/cumt-nexus-api/internal/vote/voteusecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	log := logger.New(cfg.Log).With(
		"app", cfg.App.Name,
		"env", cfg.App.Env,
	)

	if err := run(cfg, log); err != nil {
		log.Error("service exited", "error", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, log *slog.Logger) error {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.App.StartupTimeout)
	defer cancel()

	pool, err := openDB(ctx, cfg.Postgres)
	if err != nil {
		return err
	}
	defer db.Close(pool)
	log.Info("database connected")
	userRepo := userrepository.NewPostgresUserRepository(pool)
	communityRepo := communityrepository.NewPostgresCommunityRepository(pool)
	communityApplicationRepo := communityrepository.NewPostgresApplicationRepository(pool)
	platformStaffRepo := communityrepository.NewPostgresPlatformStaffRepository(pool)
	communityTxManager := communityrepository.NewPostgresCommunityTransactionManager(pool)
	postRepo := postrepository.NewPostgresPostRepository(pool)
	commentRepo := commentrepository.NewPostgresCommentRepository(pool)
	voteRepo := voterepository.NewPostgresPostVoteRepository(pool)
	moderationRepo := moderationrepository.NewPostgresModerationRepository(pool)
	searchRepo := searchrepository.NewPostgresSearchRepository(pool)
	notificationRepo := notificationrepository.NewPostgresNotificationRepository(pool)
	mediaRepo := mediarepository.NewPostgresMediaRepository(pool)
	objectStorage, err := newObjectStorage(ctx, cfg.Storage)
	if err != nil {
		return err
	}
	passwordHasher := authpassword.NewBcryptHasher()
	tokenIssuer := authtoken.NewJWTIssuer(cfg.Auth.TokenSecret, cfg.App.Name, cfg.Auth.AccessTokenTTL)
	registerUC := authusecase.NewRegisterUserCase(userRepo, passwordHasher, tokenIssuer, time.Now)
	loginUC := authusecase.NewLoginUserCase(userRepo, passwordHasher, tokenIssuer, time.Now)
	currentUserUC := userusecase.NewCurrentUserUseCase(userRepo)
	publicCommunityUC := communityusecase.NewPublicCommunityBootstrapUseCase(communityRepo, time.Now)
	communityReadUC := communityusecase.NewCommunityReadUseCase(communityRepo)
	communityApplicationUC := communityusecase.NewCommunityApplicationUseCase(
		communityRepo,
		communityApplicationRepo,
		platformStaffRepo,
		communityTxManager,
		time.Now,
	)
	postUC := postusecase.NewPostUseCaseWithAttachments(postRepo, communityReadUC, mediaRepo, cfg.Upload.ImageMaxCountPerPost, time.Now, voteRepo)
	commentUC := commentusecase.NewCommentUseCaseWithAttachments(commentRepo, postRepo, mediaRepo, cfg.Upload.ImageMaxCountPerComment, time.Now)
	voteUC := voteusecase.NewPostVoteUseCase(postRepo, voteRepo, time.Now)
	reportUC := moderationusecase.NewReportUseCase(moderationRepo, postRepo, commentRepo, time.Now)
	removeUC := moderationusecase.NewRemoveUseCase(moderationRepo, platformStaffRepo, time.Now)
	consoleUC := moderationusecase.NewConsoleUseCase(moderationRepo, moderationRepo, moderationRepo, platformStaffRepo, time.Now)
	searchUC := searchusecase.NewUseCase(searchRepo)
	notificationUC := notificationusecase.NewUseCase(notificationRepo, time.Now)
	mediaUC := mediausecase.NewUseCase(mediaRepo, objectStorage, mediausecase.UploadLimits{
		ImageMaxBytes: cfg.Upload.ImageMaxBytes,
	}, time.Now)
	if err := publicCommunityUC.EnsurePublicCommunity(ctx); err != nil {
		return fmt.Errorf("ensure public community: %w", err)
	}
	authHandler := authhttp.NewHandler(registerUC, loginUC)
	userHandler := userhttp.NewHandler(currentUserUC)
	communityHandler := communityhttp.NewHandler(communityReadUC, communityApplicationUC)
	postHandler := posthttp.NewHandler(postUC)
	commentHandler := commenthttp.NewHandler(commentUC)
	voteHandler := votehttp.NewHandler(voteUC)
	moderationHandler := moderationhttp.NewHandler(reportUC, removeUC, consoleUC)
	searchHandler := searchhttp.NewHandler(searchUC)
	notificationHandler := notificationhttp.NewHandler(notificationUC)
	mediaHandler := mediahttp.NewHandler(mediaUC)

	router := httpserver.NewRouter(log, cfg.HTTP)
	if cfg.Storage.Provider == "local" {
		router.Static("/uploads", cfg.Storage.LocalRoot)
	}
	apiV1 := router.Group("/api/v1")
	authhttp.RegisterRoutes(apiV1.Group("/auth"), authHandler)
	protectedV1 := apiV1.Group("")
	protectedV1.Use(authhttp.RequireAuth(tokenIssuer))
	userhttp.RegisterRoutes(protectedV1, userHandler)
	communityhttp.RegisterRoutes(protectedV1, communityHandler)
	posthttp.RegisterRoutes(protectedV1, postHandler)
	commenthttp.RegisterRoutes(protectedV1, commentHandler)
	votehttp.RegisterRoutes(protectedV1, voteHandler)
	moderationhttp.RegisterRoutes(protectedV1, moderationHandler)
	searchhttp.RegisterRoutes(protectedV1, searchHandler)
	notificationhttp.RegisterRoutes(protectedV1, notificationHandler)
	mediahttp.RegisterRoutes(protectedV1, mediaHandler)
	server := httpserver.NewServer(cfg.HTTP, router)

	return serveHTTP(server, cfg.HTTP, log)
}

func newObjectStorage(ctx context.Context, cfg config.ObjectStorageConfig) (mediausecase.ObjectStorage, error) {
	switch cfg.Provider {
	case "local":
		return storage.NewLocalObjectStorage(cfg), nil
	case "r2":
		client, err := storage.NewR2ObjectStorage(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("create r2 object storage: %w", err)
		}
		return client, nil
	default:
		return nil, fmt.Errorf("unsupported object storage provider %q", cfg.Provider)
	}
}

func openDB(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	pool, err := db.Open(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(ctx, pool); err != nil {
		db.Close(pool)
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

func serveHTTP(server *http.Server, cfg config.HTTPConfig, log *slog.Logger) error {
	serverErr := make(chan error, 1)

	go func() {
		log.Info("http server listening", "addr", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdownSignal)

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	case sig := <-shutdownSignal:
		log.Info("shutdown signal received", "signal", sig.String())
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server:%w", err)
	}
	if err := <-serverErr; err != nil {
		return fmt.Errorf("server close: %w", err)
	}
	return nil
}
