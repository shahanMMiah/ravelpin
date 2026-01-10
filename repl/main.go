package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/shahanmmiah/ravelpin/repl/replCommand"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	cmds := replCommand.Commands{Cmds: make(replCommand.CommandMap, 0), Helps: make(map[string]string)}
	cmds.Register("find", "find raverly post from pin", replCommand.HandlerFindRavelFromPin)
	cmds.Register("help", "help for tool commands", replCommand.MiddleWareHelp(replCommand.HandlerHelp, cmds))
	cmds.Register("quit", "quit the repl", replCommand.HandlerQuit)
	for {
		fmt.Print("ravelpin REPL > ")

		cmd, err := replCommand.CreateCommand(scanner)

		if err != nil {
			log.Println(err)
			continue
		}

		err = cmds.Run(cmd)
		if err != nil {
			log.Printf("error - %v", err.Error())
			continue
		}

	}

}
