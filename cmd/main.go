package main

import (
	"context"
	"net/http"
	"sync"

	"github.com/rs/zerolog/log"
	"order_service/internal/api"
	"order_service/internal/cache"
	"order_service/internal/config"
	"order_service/internal/db/postgres"
	"order_service/internal/processor"
	"order_service/internal/repository"
	"order_service/internal/service"
)

func main() {
	cf, err := config.Load()
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}

	ctx := context.Background()
	pool, err := postgres.Pool(ctx, cf.DatabaseURL)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}
	defer pool.Close()

	svc := service.New(repository.New(pool), cache.New(cf.Capacity, cf.TTL), cf.Limit)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		if err = svc.WarmUpCache(ctx); err != nil {
			log.Fatal().Stack().Err(err).Send()
		}
	}()

	a := api.New(svc)
	go func() {
		defer wg.Done()
		if err = http.ListenAndServe(cf.Addr, a); err != nil {
			log.Fatal().Stack().Err(err).Send()
		}
	}()

	p := processor.New(cf.Brokers, cf.Topics, cf.GroupID, svc)
	go func() {
		defer wg.Done()
		defer func() { _ = p.Stop() }()
		if err = p.Start(ctx); err != nil {
			log.Fatal().Stack().Err(err).Send()
		}
	}()

	wg.Wait()

}
