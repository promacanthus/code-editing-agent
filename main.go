package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	deepseek "github.com/cohesion-org/deepseek-go"
	"github.com/promacanthus/code-editing-agent/pkg/agent"
	"github.com/promacanthus/code-editing-agent/pkg/tool"
)

func main() {
	client := deepseek.NewClient("YOUR_DEEPSEEK_API_KEY")

	scanner := bufio.NewScanner(os.Stdin)
	getUserMessage := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	agent := agent.New(client, getUserMessage, tool.Definitions)
	err := agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
}
