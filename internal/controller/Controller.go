package controller

import (
	"context"
	"go.uber.org/dig"
)

type Controller interface {
	Start(ctx context.Context)
}

type Result struct {
	dig.Out

	Controller Controller `group:"controller"`
}
type List struct {
	dig.In

	Controllers []Controller `group:"controller"`
}
