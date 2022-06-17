package log

import (
	"fmt"

	"go.uber.org/zap"
)

var Logger *zap.Logger

func init() {
	var err error
	Logger, err = zap.NewDevelopment()

	if err != nil {
		panic(fmt.Sprintf("failed to init default logger: %v", err))
	}
}
