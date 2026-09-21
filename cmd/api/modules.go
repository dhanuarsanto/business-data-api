package main

import (
	"go.internal/business-data-api/pkg/database"

	"go.internal/business-data-api/internal/config"
	http_handler "go.internal/business-data-api/internal/handler/http"
	api_middleware "go.internal/business-data-api/internal/middleware"
	"go.internal/business-data-api/internal/repository"
	"go.internal/business-data-api/internal/usecase"
)

func BuildModules(dbRegistry *database.DBRegistry, cfg *config.Config, ipResolver *api_middleware.TrustedProxyResolver) ([]http_handler.Module, map[string]bool, map[string][]string) {
	networkMatrix := map[string]bool{
		"*":  true,
		"sa": false,
	}

	roleMatrix := map[string][]string{
		"ManageUsers":        {"sa"},
		"ReadResellerDropdown": {"sa", "op", "opout"},
		"ReadInbox":          {"sa", "op", "opout"},
		"WriteInbox":         {"sa"},
		"ReadOutbox":         {"sa", "op", "opout"},
		"WriteOutbox":        {"sa"},
	}

	userRepos := repository.NewUserRepositories(dbRegistry)
	authUsecase := usecase.NewAuthUsecase(userRepos.PG, userRepos.MS)
	authHandler := http_handler.NewAuthHandler(authUsecase, networkMatrix, ipResolver, cfg.CookieSecure)

	inboxRepos := repository.NewInboxRepositories(dbRegistry)
	inboxUsecase := usecase.NewInboxUsecase(inboxRepos.PG, inboxRepos.MS)
	inboxHandler := http_handler.NewInboxHandler(inboxUsecase)

	outboxRepos := repository.NewOutboxRepositories(dbRegistry)
	outboxUsecase := usecase.NewOutboxUsecase(outboxRepos.PG, outboxRepos.MS)
	outboxHandler := http_handler.NewOutboxHandler(outboxUsecase)

	resellerRepos := repository.NewResellerRepositories(dbRegistry)
	masterUsecase := usecase.NewMasterUsecase(resellerRepos.PG, resellerRepos.MS)
	masterHandler := http_handler.NewMasterHandler(masterUsecase)

	activeModules := []http_handler.Module{
		authHandler,
		inboxHandler,
		outboxHandler,
		masterHandler,
	}

	return activeModules, networkMatrix, roleMatrix
}
