package api

import (
	"net/http"
	"time"

	"context"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/mi-raf/comment-project/graph"
	"github.com/rs/zerolog/log"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type (
	API struct {
		srv    *handler.Server
		s      *http.Server
		listen string
		ctx    context.Context
	}

	Config struct {
		GraphCfg graph.Config
		Listen   string
	}
)

func NewApi(ctx context.Context, c *Config) *API {
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(c.GraphCfg))
	srv.SetErrorPresenter(errorHandler)
	server := &http.Server{
		Addr: c.Listen,
	}
	http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	http.Handle("/query", logMiddleware(srv))
	return &API{
		srv: srv,
		s:   server,
		ctx: ctx,
	}
}

func (a *API) Start() error {
	log.Debug().Msgf("listening on %v", a.listen)
	return a.s.ListenAndServe()
}

func (a *API) Close() {
	log.Debug().Msg("start graceful server shutdown")
	err := a.s.Shutdown(a.ctx)
	if err != nil {
		log.Error().Err(err).Msg("error while shutdowning server")
		return
	}
	log.Debug().Msg("server graceful shutdowned")
}

func errorHandler(ctx context.Context, e error) *gqlerror.Error {
	err := graphql.DefaultErrorPresenter(ctx, e)
	if err != nil {
		log.Error().Err(err).Msg("error in response")
	}
	return err
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := r
		start := time.Now()

		next.ServeHTTP(w, r)
		stop := time.Now()

		log.Debug().
			Str("remote", req.RemoteAddr).
			Str("user_agent", req.UserAgent()).
			Str("method", req.Method).
			Str("request uri", r.RequestURI).
			Dur("duration", stop.Sub(start)).
			Str("duration_human", stop.Sub(start).String()).
			Msgf("called url %s", req.URL)
	})
}
