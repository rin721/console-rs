package initapp

import "context"

type backgroundGroup struct {
	services []BackgroundService
}

func newBackgroundGroup(services ...BackgroundService) BackgroundService {
	filtered := make([]BackgroundService, 0, len(services))
	for _, service := range services {
		if service != nil {
			filtered = append(filtered, service)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return backgroundGroup{services: filtered}
}

func (g backgroundGroup) Start(ctx context.Context) error {
	started := make([]BackgroundService, 0, len(g.services))
	for _, service := range g.services {
		if err := service.Start(ctx); err != nil {
			for i := len(started) - 1; i >= 0; i-- {
				_ = started[i].Shutdown(ctx)
			}
			return err
		}
		started = append(started, service)
	}
	return nil
}

func (g backgroundGroup) Shutdown(ctx context.Context) error {
	var firstErr error
	for i := len(g.services) - 1; i >= 0; i-- {
		if err := g.services[i].Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
