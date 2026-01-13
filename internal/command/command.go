package command

import (
	commandHandler "interchange/internal/command/handler"

	"github.com/google/wire"
	"github.com/spf13/cobra"
)

var ProviderSet = wire.NewSet(NewCommand, commandHandler.NewExampleHandler)

type Command struct {
	exampleCommandHandler *commandHandler.ExampleHandler
}

// NewCommand .
func NewCommand(
	exampleCommandHandler *commandHandler.ExampleHandler,
) *Command {
	return &Command{
		exampleCommandHandler: exampleCommandHandler,
	}
}

func Register(rootCmd *cobra.Command, newCmd func() (*Command, func(), error)) {
	rootCmd.AddCommand(
		&cobra.Command{
			Use:   "example",
			Short: "example command",
			Run: func(cmd *cobra.Command, args []string) {
				command, cleanup, err := newCmd()
				if err != nil {
					panic(err)
				}
				defer cleanup()

				command.exampleCommandHandler.Hello(cmd, args)
			},
		},
	)
}
