package agent

import (
	"context"
	"encoding/json"
	"fmt"

	deepseek "github.com/cohesion-org/deepseek-go"
	"github.com/promacanthus/code-editing-agent/pkg/tool"
)

type Agent interface {
	Run(ctx context.Context) error
}

var _ Agent = (*agent)(nil)

// agent is the agent that runs the inference.
type agent struct {
	client          *deepseek.Client
	getUserMessage  func() (string, bool)
	toolDefinitions []tool.Definition
	tools           []deepseek.Tool
}

// New creates a new agent.
func New(client *deepseek.Client, fn func() (string, bool), toolDefinitions []tool.Definition) Agent {
	tools := make([]deepseek.Tool, 0, len(toolDefinitions))
	for _, t := range toolDefinitions {
		tool := deepseek.Tool{
			Type: "function",
			Function: deepseek.Function{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		}
		tools = append(tools, tool)
	}

	return &agent{
		client:          client,
		getUserMessage:  fn,
		toolDefinitions: toolDefinitions,
		tools:           tools,
	}
}

// Run starts the agent.
func (a *agent) Run(ctx context.Context) error {
	conversation := []deepseek.ChatCompletionMessage{}
	fmt.Println("Chat with DeepSeek (use 'ctrl-c' to quit)")

	readUserInput := true
	for {
		if readUserInput {
			fmt.Print("\u001b[94mYou\u001b[0m: ")
			userInput, ok := a.getUserMessage()
			if !ok {
				break
			}

			userMessage := deepseek.ChatCompletionMessage{
				Role:    deepseek.ChatMessageRoleUser,
				Content: userInput,
			}
			conversation = append(conversation, userMessage)
		}

		message, err := a.runInference(ctx, conversation)
		if err != nil {
			return err
		}
		conversation = append(conversation, *message)

		toolResults := []deepseek.ChatCompletionMessage{}
		for _, toolCall := range message.ToolCalls {
			result := a.executeTool(toolCall.ID, toolCall.Function.Name, toolCall.Function.Arguments)
			toolResults = append(toolResults, result)
		}

		if len(toolResults) == 0 {
			readUserInput = true
			fmt.Printf("\u001b[92mDeepSeek\u001b[0m: %s\n", message.Content)
			continue
		}
		readUserInput = false
		conversation = append(conversation, toolResults...)
	}
	return nil
}

// runInference runs the inference and returns the message.
func (a *agent) runInference(ctx context.Context, conversation []deepseek.ChatCompletionMessage) (*deepseek.ChatCompletionMessage, error) {
	request := &deepseek.ChatCompletionRequest{
		Model:    deepseek.DeepSeekChat,
		Messages: conversation,
		Tools:    a.tools,
	}

	resp, err := a.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return nil, err
	}

	message := resp.Choices[0].Message
	return &deepseek.ChatCompletionMessage{
		Role:      message.Role,
		Content:   message.Content,
		ToolCalls: message.ToolCalls,
	}, nil
}

// executeTool executes the tool and returns the result.
func (a *agent) executeTool(id, name, args string) deepseek.ChatCompletionMessage {
	var toolDef tool.Definition
	var found bool
	for _, t := range a.toolDefinitions {
		if t.Name == name {
			toolDef = t
			found = true
			break
		}
	}
	if !found {
		return deepseek.ChatCompletionMessage{
			Role:       deepseek.ChatMessageRoleTool,
			Content:    fmt.Sprintf("tool not found: %s", name),
			ToolCallID: id,
		}
	}

	fmt.Printf("\u001b[92mtool\u001b[0m: %s (%s)\n", name, args)
	response, err := toolDef.Function(json.RawMessage(args))
	if err != nil {
		return deepseek.ChatCompletionMessage{
			Role:       deepseek.ChatMessageRoleTool,
			Content:    err.Error(),
			ToolCallID: id,
		}
	}
	return deepseek.ChatCompletionMessage{
		Role:       deepseek.ChatMessageRoleTool,
		Content:    response,
		ToolCallID: id,
	}
}
