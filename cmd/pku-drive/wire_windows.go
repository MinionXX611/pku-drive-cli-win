//go:build windows

package main

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/Findddx/pku-drive-cli/internal/anyshare"
	"github.com/Findddx/pku-drive-cli/internal/apperr"
	"github.com/Findddx/pku-drive-cli/internal/auth"
	"github.com/Findddx/pku-drive-cli/internal/cli"
	"github.com/Findddx/pku-drive-cli/internal/config"
	"github.com/Findddx/pku-drive-cli/internal/httpx"
	"github.com/Findddx/pku-drive-cli/internal/oauth"
	"github.com/Findddx/pku-drive-cli/internal/remote"
)

// wireOptions permits deterministic construction tests without requiring a
// networked login. Windows intentionally omits upload and download services
// until their local filesystem safety model has native implementations.
type wireOptions struct {
	defaultPaths func() (config.Paths, error)
	transport    http.RoundTripper
	browser      oauth.Browser
}

type wiredRuntime struct {
	dependencies cli.Dependencies
	httpClient   *httpx.Client
}

type authOperations interface {
	Login(context.Context, io.Reader, io.Writer) (auth.Status, error)
	Status(context.Context) (auth.Status, error)
	Logout(context.Context, bool) error
}

type identityOperations interface {
	CurrentUser(context.Context) (anyshare.User, error)
}

type remoteOperations interface {
	List(context.Context, string) ([]anyshare.Item, error)
	Delete(context.Context, string, bool) (remote.DeleteResult, error)
}

type dependencyServices struct {
	manager  authOperations
	identity identityOperations
	remote   remoteOperations
}

func wireDependencies() cli.Dependencies {
	return wireDependenciesWith(wireOptions{defaultPaths: config.DefaultPaths, browser: oauth.NewSystemBrowser()})
}

func wireDependenciesWith(options wireOptions) cli.Dependencies {
	runtime, err := newWiredRuntime(options)
	if err != nil {
		return unavailableDependencies(err)
	}
	return runtime.dependencies
}

func newWiredRuntime(options wireOptions) (*wiredRuntime, error) {
	paths := options.defaultPaths
	if paths == nil {
		paths = config.DefaultPaths
	}
	storagePaths, err := paths()
	if err != nil {
		return nil, err
	}
	store := config.NewStore(storagePaths)
	configuration, err := store.LoadConfig()
	if err != nil {
		return nil, err
	}
	server := configuration.Server
	if server == "" {
		server = config.DefaultServer
	}
	httpClient := httpx.New(options.transport, httpx.Policy{MaxAttempts: 5})
	oauthClient := oauth.NewClient(server, httpClient)
	authorizer := oauth.NewBrowserAuthorizer(oauthClient, options.browser)
	manager := &auth.Manager{Store: store, OAuth: oauthClient, Authorizer: authorizer}
	api := anyshare.NewClient(server, httpClient, manager)
	dependencies := dependenciesFromServices(dependencyServices{manager: manager, identity: api, remote: remote.NewService(api)})
	return &wiredRuntime{dependencies: dependencies, httpClient: httpClient}, nil
}

func dependenciesFromServices(services dependencyServices) cli.Dependencies {
	return cli.Dependencies{
		Login: func(ctx context.Context, callbackInput io.Reader, notice io.Writer, pasteCallback bool) error {
			if !pasteCallback {
				callbackInput = nil
			}
			_, err := services.manager.Login(ctx, callbackInput, notice)
			return err
		},
		Status: func(ctx context.Context) (cli.StatusResult, error) {
			status, err := services.manager.Status(ctx)
			if err != nil {
				return cli.StatusResult{}, err
			}
			if !status.LoggedIn {
				return cli.StatusResult{}, apperr.Wrap(apperr.Auth, "status", "login required", errors.New("credentials are missing"))
			}
			user, err := services.identity.CurrentUser(ctx)
			if err != nil {
				return cli.StatusResult{}, err
			}
			refreshed, err := services.manager.Status(ctx)
			if err != nil {
				return cli.StatusResult{}, err
			}
			expires := refreshed.ExpiresAt
			return cli.StatusResult{LoggedIn: refreshed.LoggedIn, Server: refreshed.Server, Account: user.Account, ExpiresAt: &expires}, nil
		},
		List: func(ctx context.Context, path string) ([]cli.ItemResult, error) {
			items, err := services.remote.List(ctx, path)
			if err != nil {
				return nil, err
			}
			results := make([]cli.ItemResult, len(items))
			for index, item := range items {
				results[index] = itemResult(item)
			}
			return results, nil
		},
		Delete: func(ctx context.Context, remotePath string, recursive bool) (cli.DeleteResult, error) {
			result, err := services.remote.Delete(ctx, remotePath, recursive)
			if err != nil {
				return cli.DeleteResult{}, err
			}
			remoteID := result.Item.ID
			if remoteID == "" {
				remoteID = result.Item.DocID
			}
			return cli.DeleteResult{RemotePath: result.Item.Path, RemoteID: remoteID, Type: result.Item.Type, Status: string(result.Status), PendingReview: result.Status == anyshare.DeleteStatusPendingReview}, nil
		},
		Logout: func(ctx context.Context, localOnly bool) error { return services.manager.Logout(ctx, localOnly) },
	}
}

func itemResult(item anyshare.Item) cli.ItemResult {
	remoteID := item.ID
	if remoteID == "" {
		remoteID = item.DocID
	}
	return cli.ItemResult{Name: item.Name, RemotePath: item.Path, Type: item.Type, RemoteID: remoteID, Size: item.Size, Modified: item.Modified}
}

func unavailableDependencies(cause error) cli.Dependencies {
	failure := func(ctx context.Context, operation string) error {
		if err := ctx.Err(); err != nil {
			return apperr.Wrap(apperr.Interrupted, operation, "interrupted", err)
		}
		return apperr.Wrap(apperr.Local, operation, "local dependency unavailable", cause)
	}
	return cli.Dependencies{
		Login:  func(ctx context.Context, _ io.Reader, _ io.Writer, _ bool) error { return failure(ctx, "login") },
		Status: func(ctx context.Context) (cli.StatusResult, error) { return cli.StatusResult{}, failure(ctx, "status") },
		List:   func(ctx context.Context, _ string) ([]cli.ItemResult, error) { return nil, failure(ctx, "list") },
		Delete: func(ctx context.Context, _ string, _ bool) (cli.DeleteResult, error) {
			return cli.DeleteResult{}, failure(ctx, "rm")
		},
		Logout: func(ctx context.Context, _ bool) error { return failure(ctx, "logout") },
	}
}
