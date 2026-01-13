package command

import (
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type ExampleHandler struct {
	logger *zap.Logger
}

func NewExampleHandler(logger *zap.Logger) *ExampleHandler {
	return &ExampleHandler{
		logger: logger,
	}
}

func (handler *ExampleHandler) Hello(cmd *cobra.Command, args []string) {
	cmd.Println(cmd.Use, "測試命令調用成功")
	cmd.Printf("Hello %s\n", strings.Join(args, ","))
}
