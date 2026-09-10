package main

import (
	"go.internal/business-data-api/pkg/database"

	http_handler "go.internal/business-data-api/internal/handler/http"
	"go.internal/business-data-api/internal/repository"
	"go.internal/business-data-api/internal/usecase"
)

func BuildModules(dbRegistry *database.DBRegistry) ([]http_handler.Module, map[string]bool, map[string][]string) {
	networkMatrix := map[string]bool{
		"*":  true,
		"sa": false,
	}

	roleMatrix := map[string][]string{
		"ManageUsers": {"sa"},
		"ReadInbox":   {"sa", "op", "opout"},
		"WriteInbox":  {"sa"},
		"ReadOutbox":  {"sa", "op", "opout"},
		"WriteOutbox": {"sa"},
	}

	authRepo := repository.NewUserRepository(dbRegistry)
	authUsecase := usecase.NewAuthUsecase(authRepo)
	authHandler := http_handler.NewAuthHandler(authUsecase, networkMatrix)

	inboxRepo := repository.NewInboxRepository(dbRegistry)
	inboxUsecase := usecase.NewInboxUsecase(inboxRepo)
	inboxHandler := http_handler.NewInboxHandler(inboxUsecase)

	outboxRepo := repository.NewOutboxRepository(dbRegistry)
	outboxUsecase := usecase.NewOutboxUsecase(outboxRepo)
	outboxHandler := http_handler.NewOutboxHandler(outboxUsecase)

	activeModules := []http_handler.Module{
		authHandler,
		inboxHandler,
		outboxHandler,
	}

	return activeModules, networkMatrix, roleMatrix
}
