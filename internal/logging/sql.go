package logging

import (
	"strings"

	"go.uber.org/zap"
)

func LogSQLQuery(logger *zap.Logger, sql string) {
	logger.Debug(strings.Join(strings.Fields(sql), " "))
}
